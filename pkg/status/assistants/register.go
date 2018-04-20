package assistants

import (
	"github.com/caicloud/rudder/pkg/status"
	appsv1 "github.com/caicloud/rudder/pkg/status/assistants/apps/v1"
	appsv1beta1 "github.com/caicloud/rudder/pkg/status/assistants/apps/v1beta1"
	appsv1beta2 "github.com/caicloud/rudder/pkg/status/assistants/apps/v1beta2"
	batchv1 "github.com/caicloud/rudder/pkg/status/assistants/batch/v1"
	batchv1beta1 "github.com/caicloud/rudder/pkg/status/assistants/batch/v1beta1"
	batchv2alpha1 "github.com/caicloud/rudder/pkg/status/assistants/batch/v2alpha1"
	corev1 "github.com/caicloud/rudder/pkg/status/assistants/core/v1"
	extensionsv1beta1 "github.com/caicloud/rudder/pkg/status/assistants/extensions/v1beta1"
)

func Assist(umpire status.Umpire) {
	corev1.Assist(umpire)
	appsv1.Assist(umpire)
	appsv1beta1.Assist(umpire)
	appsv1beta2.Assist(umpire)
	batchv1.Assist(umpire)
	batchv1beta1.Assist(umpire)
	batchv2alpha1.Assist(umpire)
	extensionsv1beta1.Assist(umpire)
}
