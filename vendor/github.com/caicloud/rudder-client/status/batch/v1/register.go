package v1

import (
	"github.com/caicloud/rudder-client/status/universal"
	batchv1 "k8s.io/api/batch/v1"
)

var (
	gvkJob = batchv1.SchemeGroupVersion.WithKind("Job")
)

func Assist(u universal.Umpire) {
	u.Employ(gvkJob, JudgeJob)
}
