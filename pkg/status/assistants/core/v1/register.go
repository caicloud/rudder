package v1

import "github.com/caicloud/rudder/pkg/status"

// Assist adds the list of known assistant funcs to umpire
func Assist(umpire status.Umpire) {
	umpire.Employ(gvkConfigMap, ConfigMapAssistant)
	umpire.Employ(gvkPVC, PVCAssistant)
	umpire.Employ(gvkSecret, SecretAssistant)
	umpire.Employ(gvkService, ServiceAssistant)
}
