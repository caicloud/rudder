package v1

import (
	"reflect"

	"github.com/caicloud/rudder-client/status/universal"

	corev1 "k8s.io/api/core/v1"
)

var (
	gvkService   = corev1.SchemeGroupVersion.WithKind(reflect.TypeOf(corev1.Service{}).Name())
	gvkSecret    = corev1.SchemeGroupVersion.WithKind(reflect.TypeOf(corev1.Secret{}).Name())
	gvkConfigMap = corev1.SchemeGroupVersion.WithKind(reflect.TypeOf(corev1.ConfigMap{}).Name())
	gvkPVC       = corev1.SchemeGroupVersion.WithKind(reflect.TypeOf(corev1.PersistentVolumeClaim{}).Name())
)

func Assist(u universal.Umpire) {
	u.Employ(gvkService, JudgeSVC)
	u.Employ(gvkSecret, JudgeSecret)
	u.Employ(gvkConfigMap, JudgeConfigmap)
	u.Employ(gvkPVC, JudgePVC)
}
