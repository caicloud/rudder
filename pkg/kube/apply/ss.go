package apply

import (
	"k8s.io/apimachinery/pkg/runtime"
	appsv1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func init() {
	RegisterApplier(appsv1.SchemeGroupVersion.WithKind("StatefulSet"), applyStatefulSet)
}

func applyStatefulSet(current, desired runtime.Object) error {
	if current == nil || desired == nil {
		return nil
	}
	co := current.(*appsv1.StatefulSet)
	do := desired.(*appsv1.StatefulSet)
	replicas := do.Spec.Replicas
	template := do.Spec.Template
	updateStrategy := do.Spec.UpdateStrategy
	do.Spec = co.Spec
	do.Spec.Replicas = replicas
	do.Spec.Template = template
	do.Spec.UpdateStrategy = updateStrategy
	return nil
}
