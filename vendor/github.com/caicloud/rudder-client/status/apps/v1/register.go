package v1

import (
	"reflect"

	"github.com/caicloud/rudder-client/status/universal"

	appsv1 "k8s.io/api/apps/v1"
)

var (
	gvkDeployment  = appsv1.SchemeGroupVersion.WithKind(reflect.TypeOf(appsv1.Deployment{}).Name())
	gvkStatefulSet = appsv1.SchemeGroupVersion.WithKind(reflect.TypeOf(appsv1.StatefulSet{}).Name())
	gvkDaemonSet   = appsv1.SchemeGroupVersion.WithKind(reflect.TypeOf(appsv1.DaemonSet{}).Name())
)

func Assist(u universal.Umpire) {
	u.Employ(gvkDeployment, JudgeDeployment)
	u.Employ(gvkStatefulSet, JudgeStatefulSet)
	u.Employ(gvkDaemonSet, JudgeDaemonSet)
}
