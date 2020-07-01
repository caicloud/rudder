package gc

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/store"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Kind for Release
var gvkRelease = releaseapi.SchemeGroupVersion.WithKind("Release")

// Kind for GrayRelease
var gvkReleaseHistory = releaseapi.SchemeGroupVersion.WithKind("ReleaseHistory")

type resource struct {
	gvk       schema.GroupVersionKind
	namespace string
	name      string
	uid       types.UID
	object    runtime.Object
}

type release struct {
	namespace string
	name      string
	uid       types.UID
	// resources is a map of resource uids <-> resources.
	resources map[types.UID]*resource
}

// resources is a map of release uids <-> release resources.
type releaseResources struct {
	lock     sync.RWMutex
	releases map[types.UID]*release
	ignored  map[schema.GroupVersionKind]bool
}

// newReleaseResources creates a releaseResources with given ignored GVK
func newReleaseResources(ignored []schema.GroupVersionKind) *releaseResources {
	r := &releaseResources{
		releases: make(map[types.UID]*release),
		ignored:  make(map[schema.GroupVersionKind]bool),
	}
	for _, gvk := range ignored {
		r.ignored[gvk] = true
	}
	return r
}

// duplicateReleases returns a list of releases with name, namespace and uid only
func (r *releaseResources) duplicateReleases() []*releaseapi.Release {
	r.lock.RLock()
	defer r.lock.RUnlock()

	ret := make([]*releaseapi.Release, 0, len(r.releases))
	for _, r := range r.releases {
		var rel releaseapi.Release
		rel.Namespace = r.namespace
		rel.Name = r.name
		rel.UID = r.uid
		ret = append(ret, &rel)
	}
	return ret
}

// resources returns the list of resources of the specified release uid
func (r *releaseResources) resources(uid types.UID) []*resource {
	r.lock.RLock()
	defer r.lock.RUnlock()

	rs, ok := r.releases[uid]
	if !ok {
		return nil
	}
	resources := make([]*resource, 0, len(rs.resources))
	for _, r := range rs.resources {
		resources = append(resources, r)
	}
	return resources
}

// set adds or updates object's owner's resource
func (r *releaseResources) set(gvk schema.GroupVersionKind, obj runtime.Object) {
	if r.ignored[gvk] {
		return
	}
	accessor, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		return
	}
	meta := accessor.GetObjectMeta()
	owners := meta.GetOwnerReferences()
	if len(owners) <= 0 || len(owners) >= 2 {
		// If the resource have no owner reference or have two or more, we can't handle it.
		// Even if the resource have a reference to a release, we leave it to the other owners.
		// We can call the behavior as reference counter.
		return
	}
	owner := owners[0]
	if !(owner.APIVersion == gvkRelease.GroupVersion().String() && owner.Kind == gvkRelease.Kind) {
		// If the owner is not release, we don't need handle it.
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	rs, ok := r.releases[owner.UID]
	if !ok {
		rs = &release{
			namespace: meta.GetNamespace(),
			name:      owner.Name,
			uid:       owner.UID,
			resources: map[types.UID]*resource{},
		}
	}
	rs.resources[meta.GetUID()] = &resource{
		gvk:       gvk,
		namespace: meta.GetNamespace(),
		name:      meta.GetName(),
		uid:       meta.GetUID(),
		object:    obj,
	}
	r.releases[owner.UID] = rs
}

func (r *releaseResources) remove(obj runtime.Object) {
	accessor, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		return
	}
	meta := accessor.GetObjectMeta()
	owners := meta.GetOwnerReferences()
	if len(owners) <= 0 || len(owners) >= 2 {
		return
	}
	owner := owners[0]
	if !(owner.APIVersion == gvkRelease.GroupVersion().String() && owner.Kind == gvkRelease.Kind) {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	rs, ok := r.releases[owner.UID]
	if !ok {
		return
	}
	delete(rs.resources, meta.GetUID())
	if len(rs.resources) == 0 {
		delete(r.releases, rs.uid)
	}
}

// GarbageCollector collects garbage release histories and corresponding resources.
type GarbageCollector struct {
	queue         workqueue.RateLimitingInterface
	clients       kube.ClientPool
	codec         kube.Codec
	store         store.IntegrationStore
	releaseLister cache.GenericLister
	resources     *releaseResources
	synced        []cache.InformerSynced
	ignored       []schema.GroupVersionKind // indicates which resources should be ignored
	workers       int32
	working       int32
	historyLimit  int32
}

// NewGarbageCollector creates a garbage collector.
func NewGarbageCollector(clients kube.ClientPool, codec kube.Codec,
	store store.IntegrationStore, targets, ignored []schema.GroupVersionKind,
	historyLimit int32,
) (*GarbageCollector, error) {
	gc := &GarbageCollector{
		clients:      clients,
		codec:        codec,
		store:        store,
		resources:    newReleaseResources(ignored),
		queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		ignored:      ignored,
		historyLimit: historyLimit,
	}
	releaseInformer, err := store.InformerFor(gvkRelease)
	if err != nil {
		return nil, err
	}
	gc.releaseLister = releaseInformer.Lister()
	for _, target := range targets {
		gi, err := store.InformerFor(target)
		if err != nil {
			return nil, err
		}
		if target == gvkRelease {
			// Release
			gi.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
				AddFunc: func(newObj interface{}) {
					gc.enqueue(newObj)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					gc.enqueue(newObj)
				},
				DeleteFunc: func(obj interface{}) {
					gc.enqueue(obj)
				},
			})
			continue
		}
		gi.Informer().AddEventHandler(&resourceEventHandler{target, gc.queue, gc.resources})
	}
	return gc, nil
}

// resourceEventHandler is a handler implementing cache.ResourceEventHandler.
type resourceEventHandler struct {
	gvk       schema.GroupVersionKind
	queue     workqueue.RateLimitingInterface
	resources *releaseResources
}

// OnAdd enqueues newObj.
func (rh *resourceEventHandler) OnAdd(newObj interface{}) {
	obj, ok := newObj.(runtime.Object)
	if ok {
		rh.resources.set(rh.gvk, obj)
	}
}

// OnUpdate enqueues newObj. We don't need old one.
func (rh *resourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	rh.OnAdd(newObj)
}

// OnDelete is useless, we only need to handle living beings.
func (rh *resourceEventHandler) OnDelete(obj interface{}) {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		if o, ok := d.Obj.(runtime.Object); ok {
			rh.resources.remove(o)
		}
		return
	}
	if o, ok := obj.(runtime.Object); ok {
		rh.resources.remove(o)
	}
}

// enqueue only can enqueue releases. Other types are not allowed.
func (gc *GarbageCollector) enqueue(obj interface{}) {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		gc.enqueue(d.Obj)
		return
	}
	gc.queue.Add(obj)
}

// Run starts workers to handle resource events.
func (gc *GarbageCollector) Run(workers int32, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Info("Running GarbageCollector")
	gc.workers += workers

	if !cache.WaitForCacheSync(stopCh, gc.synced...) {
		glog.Errorf("Can't sync cache")
		return
	}

	glog.Info("Sync GarbageCollector cache successfully")

	for i := int32(0); i < workers; i++ {
		go wait.Until(gc.worker, time.Second, stopCh)
	}

	go gc.resync()

	<-stopCh
	glog.Info("Shutting down GarbageCollector")
}

// resync syncs all releases if there is nothing in queue.
func (gc *GarbageCollector) resync() {
	for {
		fakeReleases := gc.resources.duplicateReleases()
		synced := 0
		for {
			resyncCount := gc.workers - gc.working
			if resyncCount > 0 {
				limit := synced + int(resyncCount)
				for ; synced < len(fakeReleases) && synced < limit; synced++ {
					rel := fakeReleases[synced]
					gc.queue.Add(rel)
				}
			}
			if synced >= len(fakeReleases) {
				break
			}
			// Check every second.
			time.Sleep(time.Second)
		}
		// Check every second.
		time.Sleep(time.Second)
	}
}

// worker only guarantees the real worker is alive.
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
	gc.working++
	defer func() { gc.working-- }()

	defer gc.queue.Done(obj)
	release, ok := obj.(*releaseapi.Release)
	if !ok {
		glog.Error("Unexpected release. May serious defect occur.")
		return false
	}
	if err := gc.collect(release); err != nil {
		gc.queue.AddRateLimited(obj)
	} else {
		gc.queue.Forget(obj)
	}
	return true
}

func keyForResource(gk schema.GroupKind, name string) string {
	return gk.String() + ":" + name
}

// collect handles existent resources. So it doesn't handle deletion events.
func (gc *GarbageCollector) collect(release *releaseapi.Release) error {
	// The parameter release may be a fake release.
	// For safety, only use its Namespace, Name and UID.
	rel, err := gc.releaseLister.ByNamespace(release.Namespace).Get(release.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	desired := map[string]bool{}
	releaseAlived := false
	if err == nil {
		current, err := gc.codec.AccessorForObject(rel)
		if err != nil {
			return err
		}
		if release.UID == current.GetUID() {
			release = rel.(*releaseapi.Release)
			// the manifest should not be empty. If the manifest be empty, release may not be latest,
			// ignore this performing.
			if release.Status.Manifest == "" {
				glog.Warningf("release(%s/%s)'s manifest is empty, ignore", release.Namespace, release.Name)
				return nil
			}
			resources := render.SplitManifest(release.Status.Manifest)
			objs, accessors, err := gc.codec.AccessorsForResources(resources)
			if err != nil {
				return err
			}
			for i, obj := range objs {
				key := keyForResource(obj.GetObjectKind().GroupVersionKind().GroupKind(), accessors[i].GetName())
				desired[key] = true
			}
			releaseAlived = true
		}
	}

	resources := gc.resources.resources(release.UID)
	for _, res := range resources {
		switch {
		case res.gvk == gvkReleaseHistory:
			// Check history
			ifRetain, err := gc.ifRetainHistory(release, res.name)
			if err != nil {
				glog.Errorf("get retain info for resource %s/%s failed %v", res.namespace, res.name, err)
				return err
			}
			if releaseAlived && ifRetain {
				continue
			}
			fallthrough
		case !desired[keyForResource(res.gvk.GroupKind(), res.name)]:
			// Delete resource
			client, err := gc.clients.ClientFor(res.gvk, res.namespace)
			if err != nil {
				glog.Errorf("Can't get a client for resource %s/%s: %v", res.namespace, res.name, err)
				return err
			}
			policy := metav1.DeletePropagationBackground
			options := &metav1.DeleteOptions{
				PropagationPolicy: &policy,
				Preconditions: &metav1.Preconditions{
					UID: &res.uid,
				},
			}
			err = client.Delete(res.name, options)
			if err != nil && !errors.IsNotFound(err) {
				glog.Errorf("Can't delete resource %s/%s[%s]: %v", res.namespace, res.name, res.uid, err)
				return err
			}
			gc.resources.remove(res.object)
			glog.V(2).Infof("Delete resource %s %s/%s[%s] successfully", res.gvk.Kind, res.namespace, res.name, res.uid)
			glog.V(2).Infof("Relevant release %s/%s desired resource [%v]", release.Namespace, release.Name, desired)
		}
	}

	return nil
}

// ifRetainHistory tell if retain the release history.
// It will retain the history which between [curVersion - historyLimit + 1: + infinity).
// Why don't use (latestVersion - historyLimit), if you want acquire latestVersion, you need list all histories
// first, that's not graceful. On the other hand in a sentence, the current policy also satisfies the demand of limit
// history number because the current version will be equal with the latest version after updating the release.
func (gc *GarbageCollector) ifRetainHistory(rls *releaseapi.Release, rlsHistoryName string) (bool, error) {
	if strings.Index(rlsHistoryName, rls.Name) == -1 {
		return false, fmt.Errorf("cur rlshistory %v is not belong the rls %v", rlsHistoryName, rls.Name)
	}
	version, err := strconv.Atoi(rlsHistoryName[len(rls.Name)+2:])
	if err != nil {
		return false, err
	}
	if version <= 0 {
		return false, fmt.Errorf("cur rlshistory %v version %v is  invalid", rlsHistoryName, version)
	}
	rlsVersion := int(rls.Status.Version)
	if version+int(gc.historyLimit) > rlsVersion {
		return true, nil
	}
	return false, nil
}
