package release

import (
	"time"

	informerrelease "github.com/caicloud/clientset/informers/release/v1alpha1"
	releasev1alpha1 "github.com/caicloud/clientset/kubernetes/typed/release/v1alpha1"
	listerrelease "github.com/caicloud/clientset/listers/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/release"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/caicloud/rudder/pkg/store"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller watches all resource related release and release history.
type Controller struct {
	queue            workqueue.RateLimitingInterface
	manager          release.Manager
	releaseLister    listerrelease.ReleaseLister
	releaseHasSynced cache.InformerSynced
}

// NewReleaseController creates a release controller.
func NewReleaseController(
	clients kube.ClientPool,
	codec kube.Codec,
	store store.IntegrationStore,
	releaseClient releasev1alpha1.ReleaseV1alpha1Interface,
	releaseInformer informerrelease.ReleaseInformer,
	ignored []schema.GroupVersionKind,
	reSyncPeriod int32,
) (*Controller, error) {
	client, err := kube.NewClientWithCacheLayer(clients, codec, store)
	if err != nil {
		return nil, err
	}
	handler := release.NewReleaseHandler(client, ignored)
	backend := storage.NewReleaseBackendWithCacheLayer(releaseClient, store)
	rc := &Controller{
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		manager:          release.NewReleaseManager(backend, handler),
		releaseLister:    releaseInformer.Lister(),
		releaseHasSynced: releaseInformer.Informer().HasSynced,
	}
	releaseInformer.Informer().AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: rc.enqueueRelease,
		UpdateFunc: func(oldObj, newObj interface{}) {
			rc.enqueueRelease(newObj)
		},
		DeleteFunc: rc.enqueueRelease,
	}, time.Duration(reSyncPeriod)*time.Second)
	return rc, nil
}

// keyForObj returns the key of obj.
func (rc *Controller) keyForObj(obj interface{}) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
}

// enqueueRelease enqueues the key of obj.
func (rc *Controller) enqueueRelease(obj interface{}) {
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
func (rc *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Info("Running ReleaseController")

	if !cache.WaitForCacheSync(stopCh, rc.releaseHasSynced) {
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
func (rc *Controller) worker() {
	if err := rc.manager.Run(); err != nil {
		glog.Errorf("Can't run manager: %v", err)
	}
	glog.V(3).Infof("Processing ReleaseController releases")
	for rc.processNextWorkItem() {
	}
}

// processNextWorkItem processes next release
func (rc *Controller) processNextWorkItem() bool {
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
