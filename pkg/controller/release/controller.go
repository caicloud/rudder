package release

import (
	"time"

	"github.com/caicloud/clientset/informers"
	releasev1alpha1 "github.com/caicloud/clientset/kubernetes/typed/release/v1alpha1"
	listerrelease "github.com/caicloud/clientset/listers/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/release"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/caicloud/rudder/pkg/store"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var (
	helmNamespaceAnno = "helm.sh/namespace"
	helmReleaseAnno   = "helm.sh/release"
)

// ReleaseController watches all resource related release and release history.
type ReleaseController struct {
	queue         workqueue.RateLimitingInterface
	manager       release.ReleaseManager
	releaseLister listerrelease.ReleaseLister
	factory       informers.SharedInformerFactory
	cacheSyncs    []cache.InformerSynced
}

// NewReleaseController creates a release controller.
func NewReleaseController(
	clients kube.ClientPool,
	codec kube.Codec,
	store store.IntegrationStore,
	releaseClient releasev1alpha1.ReleaseV1alpha1Interface,
	factory informers.SharedInformerFactory,
	ignored []schema.GroupVersionKind,
) (*ReleaseController, error) {
	client, err := kube.NewClientWithCacheLayer(clients, codec, store)
	if err != nil {
		return nil, err
	}

	releaseInformer := factory.Release().V1alpha1().Releases()
	handler := release.NewReleaseHandler(render.NewRender(), client, ignored)
	backend := storage.NewReleaseBackendWithCacheLayer(releaseClient, store)
	rc := &ReleaseController{
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		manager:       release.NewReleaseManager(backend, handler),
		releaseLister: releaseInformer.Lister(),
		factory:       factory,
		cacheSyncs:    make([]cache.InformerSynced, 0),
	}

	rc.initDeletionEventHandler()

	return rc, nil
}

func (rc *ReleaseController) initDeletionEventHandler() {
	releaseInformer := rc.factory.Release().V1alpha1().Releases()
	serviceInformer := rc.factory.Core().V1().Services()
	deployInformer := rc.factory.Apps().V1beta1().Deployments()
	statefulInformer := rc.factory.Apps().V1beta1().StatefulSets()
	rcInformer := rc.factory.Core().V1().ReplicationControllers()
	cronjobInfomer := rc.factory.Batch().V2alpha1().CronJobs()
	jobInfomer := rc.factory.Batch().V1().Jobs()

	rc.cacheSyncs = append(rc.cacheSyncs,
		releaseInformer.Informer().HasSynced,
		serviceInformer.Informer().HasSynced,
		deployInformer.Informer().HasSynced,
		deployInformer.Informer().HasSynced,
		rcInformer.Informer().HasSynced,
		cronjobInfomer.Informer().HasSynced,
		jobInfomer.Informer().HasSynced,
	)

	releaseInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: rc.enqueueRelease,
		UpdateFunc: func(oldObj, newObj interface{}) {
			rc.enqueueRelease(newObj)
		},
		DeleteFunc: rc.enqueueRelease,
	})

	eventHandler := cache.ResourceEventHandlerFuncs{
		DeleteFunc: rc.deleteSubresource,
	}
	// unintended deletion of the following subresources will trigger a resync
	serviceInformer.Informer().AddEventHandler(eventHandler)
	deployInformer.Informer().AddEventHandler(eventHandler)
	statefulInformer.Informer().AddEventHandler(eventHandler)
	rcInformer.Informer().AddEventHandler(eventHandler)
	cronjobInfomer.Informer().AddEventHandler(eventHandler)
	jobInfomer.Informer().AddEventHandler(eventHandler)
}

func (rc *ReleaseController) deleteSubresource(obj interface{}) {
	// deal with DeletedFinalStateUnknown
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		obj = tombstone.Obj
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		glog.Errorf("Error get object accessor from obj %v", obj)
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

	release, err := rc.releaseLister.Releases(namespace).Get(releaseName)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("Error find release %v/%v for resource %v/%v %v", namespace, releaseName, apiVersion, kind, name)
		return
	}
	if errors.IsNotFound(err) {
		// release does not exist, skip
		return
	}

	rc.enqueueRelease(release)
}

// keyForObj returns the key of obj.
func (rc *ReleaseController) keyForObj(obj interface{}) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
}

// enqueueRelease enqueues the key of obj.
func (rc *ReleaseController) enqueueRelease(obj interface{}) {
	key, err := rc.keyForObj(obj)
	if err != nil {
		glog.Errorf("Can't get obj key: %v", err)
		return
	}

	glog.V(4).Infof("Enqueue: %s", key)
	// key must be a string
	rc.queue.Add(key)
}

// Run starts controller and checks releases
func (rc *ReleaseController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Info("Running ReleaseController")

	if !cache.WaitForCacheSync(stopCh, rc.cacheSyncs...) {
		glog.Errorf("Can't sync cache")
		return
	}

	glog.Info("Sync ReleaseController cache successfully")

	go wait.Until(rc.worker, time.Second, stopCh)

	<-stopCh
	glog.Info("Shutting down ReleaseController")
}

// worker checks improper resources. If controller unexpectedly terminated,
// some resources may not delete completely. worker should detect those
// resources and let them in a correct posture.
func (rc *ReleaseController) worker() {
	if err := rc.manager.Run(); err != nil {
		glog.Errorf("Can't run manager: %v", err)
	}
	glog.V(3).Infof("Processing ReleaseController releases")
	for rc.processNextWorkItem() {
	}
}

// processNextWorkItem processes next release
func (rc *ReleaseController) processNextWorkItem() bool {
	key, quit := rc.queue.Get()
	if quit {
		glog.Error("Unexpected quit of release queue")
		return false
	}
	defer rc.queue.Done(key)
	glog.V(4).Infof("Handle release by key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key.(string))
	if err != nil {
		glog.Errorf("Can't recognize key of release: %s", key)
		return false
	}
	release, err := rc.releaseLister.Releases(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("Can't get release: %s", key)
		return false
	}
	if err != nil {
		// Deleted
		err = rc.manager.Delete(namespace, name)
	} else {
		// Added or Updated
		err = rc.manager.Trigger(release)
	}
	if err != nil {
		// Re-enqueue
		rc.queue.AddRateLimited(key)
		glog.Errorf("Can't handle release: %+v", release)
		return false
	}
	glog.V(4).Infof("Handled release: %s", key)
	return true
}
