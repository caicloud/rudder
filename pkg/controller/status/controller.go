package status

import (
	"sync"
	"time"

	informerrelease "github.com/caicloud/clientset/informers/release/v1alpha1"
	releasev1alpha1 "github.com/caicloud/clientset/kubernetes/typed/release/v1alpha1"
	listerrelease "github.com/caicloud/clientset/listers/release/v1alpha1"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/status/assistants"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/caicloud/rudder/pkg/store"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// StatusController listens all releases and all generated resources.
// It syncs resources status to normal releases.
type StatusController struct {
	queue            workqueue.RateLimitingInterface
	lock             sync.RWMutex
	releases         map[string]*releaseapi.Release
	store            store.IntegrationStore
	backend          storage.ReleaseBackend
	releaseLister    listerrelease.ReleaseLister
	releaseHasSynced cache.InformerSynced
	detectors        []Detector
}

func NewResourceStatusController(
	codec kube.Codec,
	store store.IntegrationStore,
	releaseClient releasev1alpha1.ReleaseV1alpha1Interface,
	releaseInformer informerrelease.ReleaseInformer,
) (*StatusController, error) {
	umpire := status.NewUmpire(store)
	assistants.Assist(umpire)
	rd := NewResourceDetector(codec, umpire)

	return NewStatusController(
		store,
		releaseClient,
		releaseInformer,
		[]Detector{rd},
	)
}

// NewStatusController creates a status controller.
func NewStatusController(
	store store.IntegrationStore,
	releaseClient releasev1alpha1.ReleaseV1alpha1Interface,
	releaseInformer informerrelease.ReleaseInformer,
	detectors []Detector,
) (*StatusController, error) {
	sc := &StatusController{
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		releases:         make(map[string]*releaseapi.Release),
		store:            store,
		backend:          storage.NewReleaseBackendWithCacheLayer(releaseClient, store),
		releaseLister:    releaseInformer.Lister(),
		releaseHasSynced: releaseInformer.Informer().HasSynced,
		detectors:        detectors,
	}
	releaseInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: sc.enqueueRelease,
		UpdateFunc: func(oldObj, newObj interface{}) {
			sc.enqueueRelease(newObj)
		},
		DeleteFunc: sc.enqueueRelease,
	})
	return sc, nil
}

// keyForObj returns the key of obj.
func (sc *StatusController) keyForObj(obj interface{}) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
}

// enqueueRelease enqueues the key of obj.
func (sc *StatusController) enqueueRelease(obj interface{}) {
	key, err := sc.keyForObj(obj)
	if err != nil {
		glog.Errorf("Can't get obj key: %v", err)
		return
	}

	glog.V(4).Infof("Enqueue: %s", key)
	// key must be a string
	sc.queue.Add(key)
}

// Run starts controller and checks releases
func (sc *StatusController) Run(workers int32, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Info("Running StatusController")

	if !cache.WaitForCacheSync(stopCh, sc.releaseHasSynced) {
		glog.Errorf("Can't sync cache")
		return
	}
	glog.Info("Sync StatusController cache successfully")

	for i := int32(0); i < workers; i++ {
		go wait.Until(sc.worker, time.Second, stopCh)
	}
	// TODO(kdada): Parallelize detection
	go wait.Until(sc.detect, time.Second, stopCh)

	<-stopCh
	glog.Info("Shutting down StatusController")
}

// worker starts to process releases.
func (sc *StatusController) worker() {
	glog.V(3).Infof("Processing status of releases")
	for sc.processNextWorkItem() {
	}
}

// processNextWorkItem processes next release
func (sc *StatusController) processNextWorkItem() bool {
	key, quit := sc.queue.Get()
	if quit {
		glog.Error("Unexpected quit of release queue")
		return false
	}
	defer sc.queue.Done(key)
	glog.V(4).Infof("Handle release by key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key.(string))
	if err != nil {
		glog.Errorf("Can't recognize key of release: %s", key)
		return false
	}
	release, err := sc.releaseLister.Releases(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("Can't get release: %s", key)
		return false
	}
	if err == nil && sc.isAvailable(release) {
		sc.putRelease(release)
	} else {
		sc.removeRelease(namespace, name)
	}
	glog.V(4).Infof("Handled release: %s", key)
	return true
}

// putRelease updates release.
func (sc *StatusController) putRelease(release *releaseapi.Release) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	key := sc.keyForRelease(release)
	glog.V(4).Infof("Put release: %s", key)
	sc.releases[key] = release
}

// putRelease removes release.
func (sc *StatusController) removeRelease(namespace, name string) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	key := sc.keyFor(namespace, name)
	glog.V(4).Infof("Remove release: %s", key)
	delete(sc.releases, key)
}

// keyForRelease returns the key of release.
func (sc *StatusController) keyForRelease(release *releaseapi.Release) string {
	return sc.keyFor(release.Namespace, release.Name)
}

// keyFor returns key for specified namespace and name.
func (sc *StatusController) keyFor(namespace, name string) string {
	return namespace + "/" + name
}

// isAvailable checks if release is available.
func (sc *StatusController) isAvailable(release *releaseapi.Release) bool {
	conditionsLen := len(release.Status.Conditions)
	return conditionsLen > 0 && release.Status.Conditions[conditionsLen-1].Type == releaseapi.ReleaseAvailable
}
