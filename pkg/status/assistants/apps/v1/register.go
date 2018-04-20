package v1

import "github.com/caicloud/rudder/pkg/status"

// Assist adds the list of known assistant funcs to umpire
func Assist(umpire status.Umpire) {
	umpire.Employ(gvkDeployment, DeploymentAssistant)
	umpire.Employ(gvkDaemonSet, DaemonSetAssistant)
	umpire.Employ(gvkStatefulSet, StatefulSetAssistant)
}
