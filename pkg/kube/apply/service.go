package apply

import (
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	RegisterApplier(core.SchemeGroupVersion.WithKind("Service"), applyService)
}

func applyService(current, desired runtime.Object) error {
	if current == nil || desired == nil {
		return nil
	}
	co := current.(*core.Service)
	do := desired.(*core.Service)
	do.ResourceVersion = co.ResourceVersion
	do.Spec.ClusterIP = co.Spec.ClusterIP
	if do.Spec.Type == core.ServiceTypeNodePort &&
		co.Spec.Type == core.ServiceTypeNodePort {
		portsMap := map[int32]int32{}
		for _, port := range co.Spec.Ports {
			portsMap[port.Port] = port.NodePort
		}
		for i, port := range do.Spec.Ports {
			// Node port should always between 1 and 65535.
			// If Node port is 0, it means user want use a
			// random port or be same as current service.
			if port.NodePort <= 0 || port.NodePort > 65535 {
				// Set desired service's node port.
				do.Spec.Ports[i].NodePort = portsMap[port.Port]
			}
		}
	}
	return nil
}
