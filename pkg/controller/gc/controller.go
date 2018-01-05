package gc

import (
	"time"

	informerrelease "github.com/caicloud/clientset/informers/release/v1alpha1"
	listerrelease "github.com/caicloud/clientset/listers/release/v1alpha1"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/release-controller/pkg/kube"
	"github.com/caicloud/release-controller/pkg/render"
	"github.com/caicloud/release-controller/pkg/store"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Kind for Release
var gvkRelease = releaseapi.SchemeGroupVersion.WithKind("Release")

// Kind for GrayRelease
var gvkReleaseHistory = releaseapi.SchemeGroupVersion.WithKind("ReleaseHistory")

// GarbageCollector collects garbage release histories
// and corresponding resources.
type GarbageCollector struct {
	queue         workqueue.RateLimitingInterface
	clients       kube.ClientPool
	codec         kube.Codec
	store         store.IntegrationStore
	releaseLister listerrelease.ReleaseLister
	synced        []cache.InformerSynced
	// ignored indicates which resources should be ignored
	ignored []schema.GroupVersionKind
}

// NewGarbageCollector creates a garbage collector.
func NewGarbageCollector(
	clients kube.ClientPool,
	codec kube.Codec,
	store store.IntegrationStore,
	releaseInformer informerrelease.ReleaseInformer,
	targets []schema.GroupVersionKind,
	ignored []schema.GroupVersionKind,
) (*GarbageCollector, error) {
	gc := &GarbageCollector{
		clients:       clients,
		codec:         codec,
		store:         store,
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		releaseLister: releaseInformer.Lister(),
		synced:        []cache.InformerSynced{releaseInformer.Informer().HasSynced},
		ignored:       ignored,
	}
	for _, target := range targets {
		gi, err := store.InformerFor(target)
		if err != nil {
			return nil, err
		}
		gc.synced = append(gc.synced, gi.Informer().HasSynced)
		gi.Informer().AddEventHandler(&resourceEventHandler{target, gc.queue})
	}
	return gc, nil
}

// binding keeps the relationship between  kind and objcet.
type binding struct {
	gvk    schema.GroupVersionKind
	object runtime.Object
}

// resourceEventHandler is a handler implements cache.ResourceEventHandler.
type resourceEventHandler struct {
	gvk   schema.GroupVersionKind
	queue workqueue.RateLimitingInterface
}

// OnAdd enqueues newObj.
func (rh *resourceEventHandler) OnAdd(newObj interface{}) {
	obj, ok := newObj.(runtime.Object)
	if ok {
		rh.queue.Add(&binding{rh.gvk, obj})
	}
}

// OnUpdate enqueues newObj. We don't need old one.
func (rh *resourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	rh.OnAdd(newObj)
}

// OnDelete is useless, we only need to handle living beings.
func (rh *resourceEventHandler) OnDelete(obj interface{}) {}

// Run starts workers to handle resource events.
func (gc *GarbageCollector) Run(workers int32, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Info("Running GarbageCollector")

	if !cache.WaitForCacheSync(stopCh, gc.synced...) {
		glog.Errorf("Can't sync cache")
		return
	}

	glog.Info("Sync GarbageCollector cache successfully")

	for i := int32(0); i < workers; i++ {
		go wait.Until(gc.worker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Shutting down GarbageCollector")
}

// worker only guarantee the real worker is alive.
func (gc *GarbageCollector) worker() {
	glog.V(3).Infof("Processing GarbageCollector resources")
	for gc.processNextWorkItem() {
	}
}

// processNextWorkItem processes next release
func (gc *GarbageCollector) processNextWorkItem() bool {
	obj, quit := gc.queue.Get()
	if quit {
		glog.Error("Unexpected quit of GarbageCollector resource queue")
		return false
	}
	defer gc.queue.Done(obj)
	binding, ok := obj.(*binding)
	if !ok {
		glog.Error("Unexpected binding of resource. May serious defect occur.")
		return false
	}
	gc.collect(binding.gvk, binding.object)
	return true
}

// collect handles existent resources. So it doesn't handle deletion events.
func (gc *GarbageCollector) collect(gvk schema.GroupVersionKind, obj runtime.Object) {
	if gc.ignore(gvk) {
		return
	}
	accessor, err := gc.codec.AccessorForObject(obj)
	if err != nil {
		glog.Errorf("Can't find out the accessor for resource: %v", err)
		return
	}
	owners := accessor.GetOwnerReferences()
	if len(owners) <= 0 || len(owners) >= 2 {
		// If the resource have no owner reference or have two or more references,
		// we can't handle it.
		// Even if the resource have a reference to a release, we leave it to the
		// other owners.
		// We can call the behavior as reference counter.
		return
	}
	owner := owners[0]
	if !(owner.APIVersion == gvkRelease.GroupVersion().String() && owner.Kind == gvkRelease.Kind) {
		// If the owner is not release, we don't need handle it.
		return
	}

	release, err := gc.releaseLister.Releases(accessor.GetNamespace()).Get(owner.Name)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("Can't find release %s refered by resource %s/%s: %v", owner.Name, accessor.GetNamespace(), accessor.GetName(), err)
		return
	}

	client, err := gc.clients.ClientFor(gvk, accessor.GetNamespace())
	if err != nil {
		glog.Errorf("Can't get a client for resource %s/%s: %v", accessor.GetNamespace(), accessor.GetName(), err)
		return
	}

	// A resource which conforms to one of following rules will be deleted:
	// 1. Target release is nonexistent (release history only can trigger the rule).
	// 2. The release is available and the resource is not in the manifest of release.

	policy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}

	if release == nil || release.GetUID() != owner.UID {
		// Log the release info.
		if release != nil {
			glog.V(4).Infof("%+v", release)
		}
		glog.V(4).Infof("%+v", obj)

		// Delete the resource if its target release is not exist.
		err = client.Delete(accessor.GetName(), options)
		if err != nil {
			glog.Errorf("Can't delete resource %s/%s: %v", accessor.GetNamespace(), accessor.GetName(), err)
			return
		}
		glog.V(2).Infof("Delete resource %s %s/%s successfully", gvk.Kind, accessor.GetNamespace(), accessor.GetName())
		return
	}
	if gvk == gvkReleaseHistory {
		// Ignore release history
		return
	}

	// Check whether the release is available.
	if !gc.isAvailable(release) {
		return
	}

	resources := render.SplitManifest(release.Status.Manifest)
	objs, accessors, err := gc.codec.AccessorsForResources(resources)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("Can't decode manifest of release %s/%s: %v", release.Namespace, release.Name, err)
		return
	}
	// Find resource
	found := false
	for i, obj := range objs {
		if obj.GetObjectKind().GroupVersionKind() == gvk && accessor.GetName() == accessors[i].GetName() {
			found = true
			break
		}
	}
	if !found {
		err = client.Delete(accessor.GetName(), options)
		if err != nil && !errors.IsNotFound(err) {
			glog.Errorf("Can't delete resource %s/%s: %v", accessor.GetNamespace(), accessor.GetName(), err)
			return
		}
		glog.V(2).Infof("Delete resource %s %s/%s successfully", gvk.Kind, accessor.GetNamespace(), accessor.GetName())
		return
	}
}

// ignore checks if an object should be ignored.
func (gc *GarbageCollector) ignore(gvk schema.GroupVersionKind) bool {
	if gvk == gvkRelease {
		// Ignore releases
		// We don't need handle release type
		return true
	}
	for _, i := range gc.ignored {
		if i == gvk {
			return true
		}
	}
	return false
}

// isAvailable checks if release is available.
func (gc *GarbageCollector) isAvailable(release *releaseapi.Release) bool {
	conditionsLen := len(release.Status.Conditions)
	return conditionsLen > 0 && release.Status.Conditions[conditionsLen-1].Type == releaseapi.ReleaseAvailable
}
