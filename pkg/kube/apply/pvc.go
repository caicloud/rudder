package apply

import (
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func init() {
	RegisterApplier(apiv1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"), applyPVC)
}

func applyPVC(current, desired runtime.Object) error {
	if current == nil || desired == nil {
		return nil
	}
	// PVC's spec is immutable.
	co := current.(*apiv1.PersistentVolumeClaim)
	do := desired.(*apiv1.PersistentVolumeClaim)
	do.Spec = co.Spec
	return nil
}
