package v1

import (
	"reflect"

	"github.com/caicloud/rudder-client/status/universal"

	batchv1 "k8s.io/api/batch/v1"
)

var (
	gvkJob = batchv1.SchemeGroupVersion.WithKind(reflect.TypeOf(batchv1.Job{}).Name())
)

func Assist(u universal.Umpire) {
	u.Employ(gvkJob, JudgeJob)
}
