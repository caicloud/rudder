package v1beta1

import (
	"reflect"

	"github.com/caicloud/rudder-client/status/universal"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
)

var (
	gvkCronJob = batchv1beta1.SchemeGroupVersion.WithKind(reflect.TypeOf(batchv1beta1.CronJob{}).Name())
)

func Assist(u universal.Umpire) {
	u.Employ(gvkCronJob, JudgeCronJob)
}
