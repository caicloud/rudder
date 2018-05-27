package v1

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkDeployment = apps.SchemeGroupVersion.WithKind("Deployment")

func DeploymentAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	dp, ok := obj.(*apps.Deployment)
	if !ok {
		return status.None, fmt.Errorf("unknown type for deployment: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if len(dp.Status.Conditions) > 0 &&
		dp.Status.Conditions[len(dp.Status.Conditions)-1].Type == apps.DeploymentReplicaFailure {
		return status.Failure, nil
	}

	desired := int32(0)
	if dp.Spec.Replicas != nil {
		desired = *dp.Spec.Replicas
	}
	current := dp.Status.Replicas
	updated := dp.Status.UpdatedReplicas
	available := dp.Status.AvailableReplicas
	unavailable := dp.Status.UnavailableReplicas
	observedGen := dp.Status.ObservedGeneration
	switch {
	case unavailable == 0 && desired == current && desired == updated && desired == available:
		return status.Available, nil
	case unavailable > 0 && desired == updated && desired != available && observedGen != 1:
		// TODO(kdada): Check wrong pods for more precise verdict
		return status.Failure, nil
	default:
		return status.Progressing, nil
	}
}
