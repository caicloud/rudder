package assistants

import (
	"fmt"

	"github.com/caicloud/release-controller/pkg/status"
	"github.com/caicloud/release-controller/pkg/store"
	"k8s.io/apimachinery/pkg/runtime"
	corev1 "k8s.io/client-go/pkg/api/v1"
)

var gvkPVC = corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim")

func PVCAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	pvc, ok := obj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return status.None, fmt.Errorf("unknown type for persistent volume claim: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	switch pvc.Status.Phase {
	case corev1.ClaimBound:
		return status.Available, nil
	case corev1.ClaimLost:
		return status.Failure, nil
	default:
		return status.Progressing, nil
	}
}
