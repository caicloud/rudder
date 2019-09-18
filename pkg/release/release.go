package release

import (
	"context"
	"fmt"
	"sync"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/golang/glog"
)

// Getter gets a specific release from channel.
type Getter interface {
	// Key returns the unique id of the getter.
	Key() string
	// Get gets a specific release. All releases have same unique id.
	// It means all release are same, but the data of releases may be different.
	Get() <-chan *releaseapi.Release
}

// Handler is used to handle a bundle of releases. It should not exit except
// that ctx has been canceled.
// If ctx was canceled, Handler should clean all resources created by the handler.
type Handler func(ctx context.Context, storage storage.ReleaseStorage, getter Getter)

// Manager manages the behavior of releases.
type Manager interface {
	// Run runs manager. It does nothing when manager is running.
	Run() error
	// Trigger submits a release to manager. Manager decides the next step
	// and dispatch a handler to execute.
	Trigger(obj *releaseapi.Release) error
	// Delete deletes All related resources.
	Delete(namespace, name string) error
}

// NewReleaseManager creates a release manager
func NewReleaseManager(backend storage.ReleaseBackend, handler Handler) Manager {
	return &releaseManager{
		backend:  backend,
		handler:  handler,
		handlers: make(map[string]*releaseHandler),
		running:  false,
	}
}

type releaseManager struct {
	sync.Mutex
	backend  storage.ReleaseBackend
	handler  Handler
	handlers map[string]*releaseHandler
	running  bool
}

// Run runs manager. It does nothing when manager is running.
func (rm *releaseManager) Run() error {
	rm.Lock()
	defer rm.Unlock()
	if rm.running {
		// Running
		return nil
	}
	rm.running = true
	return nil
}

// Trigger submits a release to manager. Manager decides the next step
// and dispatch a handler to execute.
func (rm *releaseManager) Trigger(obj *releaseapi.Release) error {
	if !rm.running {
		return fmt.Errorf("release manager does not run")
	}

	key := rm.keyForObj(obj)
	rm.Lock()
	defer rm.Unlock()
	rh, ok := rm.handlers[key]
	if !ok {
		// create handler
		rh = newReleaseHandler(key, rm.backend.ReleaseStorage(obj), rm.handler)
		rm.handlers[key] = rh
	}
	return rh.Enqueue(obj)
}

// Delete deletes All related resources.
func (rm *releaseManager) Delete(namespace, name string) error {
	if !rm.running {
		return fmt.Errorf("release manager does not run")
	}

	key := rm.keyForName(namespace, name)
	rm.Lock()
	defer rm.Unlock()
	rh, ok := rm.handlers[key]
	if !ok {
		// If came here, a covert bug exists.
		return fmt.Errorf("no existing resource for %s", key)
	}
	delete(rm.handlers, key)
	return rh.Stop()
}

// keyForObj returns unique key for obj
func (rm *releaseManager) keyForObj(obj *releaseapi.Release) string {
	return rm.keyForName(obj.Namespace, obj.Name)
}

// keyForName returns unique key for namespace and name
func (rm *releaseManager) keyForName(namespace, name string) string {
	return namespace + "/" + name
}

type releaseHandler struct {
	name     string
	storage  storage.ReleaseStorage
	handler  Handler
	queue    chan *releaseapi.Release
	cancel   context.CancelFunc
	canceled bool
}

func newReleaseHandler(name string, storage storage.ReleaseStorage, handler Handler) *releaseHandler {
	return &releaseHandler{
		name:    name,
		storage: storage,
		handler: handler,
		// Now we store events via a channel. But it's not a final solution.
		// May we should replace channel with a unbuffered list.
		queue: make(chan *releaseapi.Release, 10),
	}
}

// Enqueue sends a new obj to handler. Do not call the method in parallel.
func (rh *releaseHandler) Enqueue(obj *releaseapi.Release) error {
	if rh.canceled {
		// If came here, a covert bug exists.
		return fmt.Errorf("can't enqueue an obj to a canceled handler")
	}
	rh.queue <- obj
	if rh.cancel == nil {
		// create goroutine to handle the obj
		ctx, cancel := context.WithCancel(context.Background())
		rh.cancel = cancel
		go rh.handler(ctx, rh.storage, rh)
		glog.V(2).Infof("Start release handler of %s", rh.name)
	}
	return nil
}

// Stop stops the handler. Before stopping, the handler cleans all resources
// related by the handler. Do not call the method in parallel.
func (rh *releaseHandler) Stop() error {
	if !rh.canceled {
		rh.canceled = true
		rh.cancel()
		glog.V(2).Infof("Stop release handler of %s", rh.name)
	}
	return nil
}

// Key returns the unique id of the getter.
func (rh *releaseHandler) Key() string {
	return rh.name
}

// Get gets a specific release. All releases have same unique id.
// It means all release are same, but the data of releases may be different.
func (rh *releaseHandler) Get() <-chan *releaseapi.Release {
	return rh.queue
}
