package apply

import (
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func init() {
	RegisterApplier(apiv1.SchemeGroupVersion.WithKind("Service"), applyService)
}

func applyService(current, desired runtime.Object) error {
	if current == nil || desired == nil {
		return nil
	}
	co := current.(*apiv1.Service)
	do := desired.(*apiv1.Service)
	do.ResourceVersion = co.ResourceVersion
	do.Spec.ClusterIP = co.Spec.ClusterIP
	if do.Spec.Type == apiv1.ServiceTypeNodePort &&
		co.Spec.Type == apiv1.ServiceTypeNodePort {
		portsMap := map[int32]int32{}
		for _, port := range co.Spec.Ports {
			portsMap[port.Port] = port.NodePort
		}
		for i, port := range do.Spec.Ports {
			if port.NodePort <= 0 || port.NodePort > 65535 {
				// Set desired service's node port.
				do.Spec.Ports[i].NodePort = portsMap[port.Port]
			}
		}
	}
	return nil
}
