package assistants

import (
	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkConfigMap = core.SchemeGroupVersion.WithKind("ConfigMap")

func ConfigMapAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	return status.Available, nil
}
