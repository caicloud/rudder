package v2alpha1

import "github.com/caicloud/rudder/pkg/status"

// Assist adds the list of known assistant funcs to umpire
func Assist(umpire status.Umpire) {
	umpire.Employ(gvkCronJob, CronJobAssistant)
}
