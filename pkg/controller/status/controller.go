package status

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	informerrelease "github.com/caicloud/clientset/informers/release/v1alpha1"
	"github.com/caicloud/clientset/kubernetes"
	releasev1alpha1 "github.com/caicloud/clientset/kubernetes/typed/release/v1alpha1"
	"github.com/caicloud/clientset/listerfactory"
	listerrelease "github.com/caicloud/clientset/listers/release/v1alpha1"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/clientset/util/syncqueue"
	"github.com/caicloud/rudder-client/status"
	statusinterface "github.com/caicloud/rudder-client/status/universal"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/caicloud/rudder/pkg/store"
	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// StatusController listens all releases and all generated resources.
// It syncs resources status to normal releases.
type StatusController struct {
	codec         kube.Codec
	backend       storage.ReleaseBackend
	workqueue     *syncqueue.SyncQueue
	factory       informers.SharedInformerFactory
	store         store.IntegrationStore
	releaseLister listerrelease.ReleaseLister
	hasSynced     []cache.InformerSynced
	umpire        statusinterface.Umpire
	resources     kube.APIResources
}

func NewStatusController(
	kubeClient kubernetes.Interface,
	codec kube.Codec,
	store store.IntegrationStore,
	releaseClient releasev1alpha1.ReleaseV1alpha1Interface,
	releaseInformer informerrelease.ReleaseInformer,
	childResources []schema.GroupVersionKind,
	resources kube.APIResources,
) (*StatusController, error) {
	factory := store.SharedInformerFactory()
	extraResources := []schema.GroupVersionKind{
		appsv1.SchemeGroupVersion.WithKind("ReplicaSet"),
		appsv1.SchemeGroupVersion.WithKind("ControllerRevision"),
	}

	sc := &StatusController{
		codec:         codec,
		backend:       storage.NewReleaseBackend(releaseClient),
		store:         store,
		factory:       factory,
		releaseLister: releaseInformer.Lister(),
		hasSynced:     []cache.InformerSynced{releaseInformer.Informer().HasSynced},
		umpire:        status.NewUmpire(listerfactory.NewListerFactoryFromInformer(store.SharedInformerFactory())),
		resources:     resources,
	}

	sc.workqueue = syncqueue.NewSyncQueue(&releaseapi.Release{}, sc.syncRelease)
	// init release event handler
	releaseInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sc.workqueue.Enqueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRelease := oldObj.(*releaseapi.Release)
			newRelease := newObj.(*releaseapi.Release)
			if reflect.DeepEqual(oldRelease.Spec, newRelease.Spec) {
				return
			}
			sc.workqueue.Enqueue(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			sc.workqueue.Enqueue(obj)
		},
	})
	// init subresources event handler
	for _, gvk := range childResources {
		resource, err := resources.ResourceFor(gvk)
		if err != nil {
			return nil, err
		}
		informer, err := factory.ForResource(resource.GroupVersionResource())
		if err != nil {
			return nil, err
		}
		informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: sc.enqueueChildresource,
			UpdateFunc: func(oldObj, newObj interface{}) {
				sc.enqueueChildresource(newObj)
			},
		})
		sc.hasSynced = append(sc.hasSynced, informer.Informer().HasSynced)
	}

	// for lister
	for _, gvk := range extraResources {
		resource, err := resources.ResourceFor(gvk)
		if err != nil {
			return nil, err
		}
		informer, err := factory.ForResource(resource.GroupVersionResource())
		if err != nil {
			return nil, err
		}
		sc.hasSynced = append(sc.hasSynced, informer.Informer().HasSynced)
	}

	// init event handler
	factory.Core().V1().Events().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sc.enqueueEvent(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			sc.enqueueEvent(newObj)
		},
	})
	// init pod handler
	factory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sc.enqueueAccessor(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			sc.enqueueAccessor(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			sc.enqueueAccessor(obj)
		},
	})
	sc.hasSynced = append(sc.hasSynced,
		factory.Core().V1().Events().Informer().HasSynced,
		factory.Core().V1().Pods().Informer().HasSynced,
	)
	return sc, nil
}

func (sc *StatusController) enqueueChildresource(obj interface{}) {
	// deal with DeletedFinalStateUnknown
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		obj = tombstone.Obj
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		glog.Errorf("Error get type accessor from obj %v", obj)
		return
	}
	typeAccessor, err := meta.TypeAccessor(obj)
	if err != nil {
		glog.Errorf("Error get type accessor from obj %v", obj)
		return
	}

	apiVersion := typeAccessor.GetAPIVersion()
	kind := typeAccessor.GetKind()
	name := accessor.GetName()
	namespace := accessor.GetNamespace()
	owners := accessor.GetOwnerReferences()

	var releaseName string
	for _, owner := range owners {
		if owner.Kind == "Release" {
			releaseName = owner.Name
			break
		}
	}
	if releaseName == "" {
		// skip
		return
	}

	release, err := sc.releaseLister.Releases(namespace).Get(releaseName)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("Error find release %v/%v for resource %v/%v %v", namespace, releaseName, apiVersion, kind, name)
		return
	}
	if errors.IsNotFound(err) {
		// release does not exist, skip
		return
	}
	// one release may be triggered by many subresource, we only need the latest one
	sc.workqueue.Enqueue(release)
}

// Run starts controller and checks releases
func (sc *StatusController) Run(workers int32, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Info("Running StatusController")

	if !cache.WaitForCacheSync(stopCh, sc.hasSynced...) {
		glog.Errorf("Can't sync cache")
		return
	}
	glog.Info("Sync StatusController cache successfully")

	sc.workqueue.Run(int(workers))
	defer sc.workqueue.ShutDown()

	<-stopCh
	glog.Info("Shutting down StatusController")
}

func (sc *StatusController) syncRelease(obj interface{}) error {
	key := obj.(string)
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)
	release, err := sc.releaseLister.Releases(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		glog.Warningf("release %v not found", key)
		// deleted
		return nil
	}

	if release.Status.Manifest == "" {
		glog.Warningf("relesae(%v)'s manifest is empty", key)
		// No resource
		return nil
	}

	result, podStatistics, err := sc.detect(release)
	if err != nil {
		glog.Errorf("Can't detect status for %s/%s: %v", release.Namespace, release.Name, err)
		return err
	}

	_, err = sc.backend.ReleaseStorage(release).Patch(func(release *releaseapi.Release) {
		if release.Status.Details == nil {
			release.Status.Details = make(map[string]releaseapi.ReleaseDetailStatus)
		}
		// set kind to "", use node name as key
		kind := ""
		// Clear status for this kind
		for key := range release.Status.Details {
			k, _, err := ParseKey(key)
			if err != nil {
				glog.Errorf("Invalid key in details: %v", err)
			} else {
				if kind != k {
					continue
				}
			}
			delete(release.Status.Details, key)
		}

		// Set status for detector
		for node, status := range result {
			key, err := Key(kind, node)
			if err != nil {
				glog.Errorf("Can't get key for kind %s and node %s: %v", kind, node, err)
				continue
			}
			release.Status.Details[key] = status
		}
		release.Status.PodStatistics = *podStatistics
	})

	return nil
}

func (sc *StatusController) detect(release *releaseapi.Release) (map[string]releaseapi.ReleaseDetailStatus, *releaseapi.PodStatistics, error) {
	if release.Status.Manifest == "" {
		// No resource
		return map[string]releaseapi.ReleaseDetailStatus{}, nil, nil
	}
	carrier, err := render.CarrierForManifest(release.Status.Manifest)
	if err != nil {
		glog.Errorf("Can't parse manifest for %s/%s: %v", release.Namespace, release.Name, err)
		return nil, nil, err
	}

	details := make(map[string]releaseapi.ReleaseDetailStatus)
	podStatistics := releaseapi.PodStatistics{
		OldPods:     make(releaseapi.PodStatusCounter),
		UpdatedPods: make(releaseapi.PodStatusCounter),
	}
	err = carrier.Run(context.Background(), render.PositiveOrder, func(ctx context.Context, node string, resources []string) error {
		detail := releaseapi.ReleaseDetailStatus{
			Path:      node,
			Resources: make(map[string]releaseapi.ResourceCounter),
		}
		for _, resource := range resources {
			obj, accessor, err := sc.codec.AccessorForResource(resource)
			if err != nil {
				return err
			}
			gvk := obj.GetObjectKind().GroupVersionKind()
			informer, err := sc.store.InformerFor(gvk)
			if err != nil {
				glog.Errorf("Can't get informer for %s: %v", node, err)
				return err
			}

			runningObj, err := informer.Lister().ByNamespace(release.Namespace).Get(accessor.GetName())
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
			status := releaseapi.ResourceStatusFrom(releaseapi.ResourceProgressing)
			var statistics *releaseapi.PodStatistics
			if err == nil {
				// There is no gvk in runningObj. We set it here.
				runningObj.GetObjectKind().SetGroupVersionKind(gvk)
				status, err = sc.umpire.Judge(runningObj)
				if err != nil {
					// Log the error and mark as Failure
					glog.Errorf("Can't decode resource for %s: %v", node, err)
					status.Phase = releaseapi.ResourceFailed
					status.Reason = "ErrorJudgeResource"
					status.Message = err.Error()
				}
				statistics = status.PodStatistics
			}

			if statistics != nil {
				for k, v := range statistics.OldPods {
					podStatistics.OldPods[k] += v
				}
				for k, v := range statistics.UpdatedPods {
					podStatistics.UpdatedPods[k] += v
				}
			}

			key := gvk.String()
			counter, ok := detail.Resources[key]
			if !ok {
				counter = make(releaseapi.ResourceCounter)
			}
			_, ok = counter[status.Phase]
			if ok {
				counter[status.Phase]++
			} else {
				counter[status.Phase] = 1
			}

			detail.Reason = status.Reason
			detail.Message = status.Message
			detail.Resources[key] = counter
		}
		details[node] = detail
		return nil
	})
	if err != nil {
		glog.Errorf("Can't run resources for %s/%s: %v", release.Namespace, release.Name, err)
		return nil, nil, err
	}
	return details, &podStatistics, nil
}

// kindValidator and nameValidator validates kind and name of a key
var kindValidator = regexp.MustCompile(`[a-zA-Z0-9]*`)
var nameValidator = regexp.MustCompile(`[a-zA-Z0-9/\.]+`)

// Key generates a key for kind and name.
// use name if kind == ""
func Key(kind, name string) (string, error) {
	if !kindValidator.Match([]byte(kind)) {
		return "", fmt.Errorf("invalid kind of key")
	}
	if !nameValidator.Match([]byte(name)) {
		return "", fmt.Errorf("invalid name of key")
	}
	if kind == "" {
		return name, nil
	}
	return fmt.Sprintf("%s:%s", kind, name), nil
}

// keyValidator checks if a key is valid.
var keyValidator = regexp.MustCompile(`([a-zA-Z0-9]+:)?[a-zA-Z0-9/\.]+`)

// ParseKey parses key to kind and name.
func ParseKey(key string) (kind string, name string, err error) {
	if !keyValidator.Match([]byte(key)) {
		return "", "", fmt.Errorf("invalid key")
	}
	index := strings.IndexAny(key, ":")
	if index < 0 {
		return "", key, nil
	}
	return key[:index], key[index+1:], err
}
