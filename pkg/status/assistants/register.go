package assistants

import "github.com/caicloud/rudder/pkg/status"

// Register registers all assistant for an umpire.
func Register(umpire status.Umpire) {
	umpire.Employ(gvkService, ServiceAssistant)
	umpire.Employ(gvkDeployment, DeploymentAssistant)
	umpire.Employ(gvkStatefulSet, StatefulSetAssistant)
	umpire.Employ(gvkDaemonSet, DaemonSetAssistant)
	umpire.Employ(gvkJob, JobAssistant)
	umpire.Employ(gvkCronJob, CronJobAssistant)
	umpire.Employ(gvkPVC, PVCAssistant)
	umpire.Employ(gvkConfigMap, ConfigMapAssistant)
	umpire.Employ(gvkSecret, SecretAssistant)
}
