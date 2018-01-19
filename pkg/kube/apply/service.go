package apply

import (
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func init() {
	registerApplier(apiv1.SchemeGroupVersion.WithKind("Service"), applyService)
}

func applyService(current, desired runtime.Object) error {
	if current == nil || desired == nil {
		return nil
	}
	cs := current.(*apiv1.Service)
	ds := desired.(*apiv1.Service)
	ds.ResourceVersion = cs.ResourceVersion
	ds.Spec.ClusterIP = cs.Spec.ClusterIP
	return nil
}
