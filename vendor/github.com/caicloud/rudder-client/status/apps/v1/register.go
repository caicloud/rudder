package v1

import (
	"github.com/caicloud/rudder-client/status/universal"
	appsv1 "k8s.io/api/apps/v1"
)

var (
	gvkDeployment  = appsv1.SchemeGroupVersion.WithKind("Deployment")
	gvkStatefulSet = appsv1.SchemeGroupVersion.WithKind("StatefulSet")
	gvkDaemonSet   = appsv1.SchemeGroupVersion.WithKind("DaemonSet")
)

func Assist(u universal.Umpire) {
	u.Employ(gvkDeployment, JudgeDeployment)
	u.Employ(gvkStatefulSet, JudgeStatefulSet)
	u.Employ(gvkDaemonSet, JudgeDaemonSet)
}
