package apply

import (
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterApplier(apps.SchemeGroupVersion.WithKind("StatefulSet"), applyStatefulSet)
}

func applyStatefulSet(current, desired runtime.Object) error {
	if current == nil || desired == nil {
		return nil
	}
	co := current.(*apps.StatefulSet)
	do := desired.(*apps.StatefulSet)
	replicas := do.Spec.Replicas
	template := do.Spec.Template
	updateStrategy := do.Spec.UpdateStrategy
	do.Spec = co.Spec
	do.Spec.Replicas = replicas
	do.Spec.Template = template
	do.Spec.UpdateStrategy = updateStrategy
	return nil
}
