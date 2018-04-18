package assistants

import (
	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var gvkSecret = apiv1.SchemeGroupVersion.WithKind("Secret")

func SecretAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	return status.Available, nil
}
