package status

import (
	"fmt"
	"sync"

	"github.com/caicloud/rudder/pkg/store"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Sentence is the final result of an object.
type Sentence string

const (
	// None is only used to identity an empty status.
	None Sentence = ""
	// Failure describes that the object is failed.
	Failure Sentence = "Failure"
	// Progressing describes that the object is progressing.
	// Some parts may be available, but not all parts are available.
	Progressing Sentence = "Progressing"
	// Available means that the object is available.
	Available Sentence = "Available"
)

// Assistant handles a kind of object. It will generates the sentence for the object.
type Assistant func(store store.IntegrationStore, obj runtime.Object) (Sentence, error)

// Umpire can employs many assistant to handle many kinds of objects.
type Umpire interface {
	// Employ employs an assistant for specified object kind.
	Employ(gvk schema.GroupVersionKind, assistant Assistant)
	// Judge judges the object and generates a sentence.
	Judge(obj runtime.Object) (Sentence, error)
}

// NewUmpire creates an umpire.
func NewUmpire(store store.IntegrationStore) Umpire {
	return &umpire{
		store:      store,
		assistants: make(map[schema.GroupVersionKind]Assistant),
	}
}

type umpire struct {
	lock       sync.RWMutex
	store      store.IntegrationStore
	assistants map[schema.GroupVersionKind]Assistant
}

// Employ employs an assistant for specified object kind.
func (u *umpire) Employ(gvk schema.GroupVersionKind, assistant Assistant) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.assistants[gvk] = assistant
}

// Judge judges the object and generates a sentence.
func (u *umpire) Judge(obj runtime.Object) (Sentence, error) {
	u.lock.RLock()
	defer u.lock.RUnlock()
	gvk := obj.GetObjectKind().GroupVersionKind()
	assistant, ok := u.assistants[gvk]
	if !ok {
		return None, fmt.Errorf("can't find an assistant for: %s", gvk.String())
	}
	return assistant(u.store, obj)
}
