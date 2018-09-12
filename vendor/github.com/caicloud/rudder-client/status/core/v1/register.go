package v1

import (
	"github.com/caicloud/rudder-client/status/universal"
	corev1 "k8s.io/api/core/v1"
)

var (
	gvkService   = corev1.SchemeGroupVersion.WithKind("Service")
	gvkSecret    = corev1.SchemeGroupVersion.WithKind("Secret")
	gvkConfigMap = corev1.SchemeGroupVersion.WithKind("ConfigMap")
	gvkPVC       = corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim")
)

func Assist(u universal.Umpire) {
	u.Employ(gvkService, JudgeSVC)
	u.Employ(gvkSecret, JudgeSecret)
	u.Employ(gvkConfigMap, JudgeConfigmap)
	u.Employ(gvkPVC, JudgePVC)
}
