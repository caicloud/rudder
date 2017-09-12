package status

import (
	"fmt"
	"sync"

	"github.com/caicloud/release-controller/pkg/store"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Sentence string

const (
	None        Sentence = ""
	Failure     Sentence = "Failure"
	Progressing Sentence = "Progressing"
	Available   Sentence = "Available"
)

type Assistant func(store store.IntegrationStore, obj runtime.Object) (Sentence, error)

type Umpire interface {
	Employ(gvk schema.GroupVersionKind, assistant Assistant)
	Judge(obj runtime.Object) (Sentence, error)
}

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

func (u *umpire) Employ(gvk schema.GroupVersionKind, assistant Assistant) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.assistants[gvk] = assistant
}

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
