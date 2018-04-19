package apply

import (
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterApplier(core.SchemeGroupVersion.WithKind("PersistentVolumeClaim"), applyPVC)
}

func applyPVC(current, desired runtime.Object) error {
	if current == nil || desired == nil {
		return nil
	}
	// PVC's spec is immutable.
	co := current.(*core.PersistentVolumeClaim)
	do := desired.(*core.PersistentVolumeClaim)
	do.Spec = co.Spec
	return nil
}
