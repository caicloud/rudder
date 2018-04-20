package v1

import (
	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkSecret = core.SchemeGroupVersion.WithKind("Secret")

func SecretAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	return status.Available, nil
}
