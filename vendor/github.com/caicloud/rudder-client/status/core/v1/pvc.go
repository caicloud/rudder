package v1

import (
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func JudgePVC(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	pvc, ok := obj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for persistent volume claim: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	switch pvc.Status.Phase {
	case corev1.ClaimBound:
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceRunning), nil
	case corev1.ClaimLost:
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceFailed), nil
	}
	return releaseapi.ResourceStatusFrom(releaseapi.ResourceProgressing), nil
}
