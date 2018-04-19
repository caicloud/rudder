package assistants

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkPVC = core.SchemeGroupVersion.WithKind("PersistentVolumeClaim")

func PVCAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	pvc, ok := obj.(*core.PersistentVolumeClaim)
	if !ok {
		return status.None, fmt.Errorf("unknown type for persistent volume claim: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	switch pvc.Status.Phase {
	case core.ClaimBound:
		return status.Available, nil
	case core.ClaimLost:
		return status.Failure, nil
	default:
		return status.Progressing, nil
	}
}
