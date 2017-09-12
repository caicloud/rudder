package assistants

import (
	"github.com/caicloud/release-controller/pkg/status"
	"github.com/caicloud/release-controller/pkg/store"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var gvkService = apiv1.SchemeGroupVersion.WithKind("Service")

func ServiceAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	return status.Available, nil
}
