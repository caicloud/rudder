package v1

import (
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func JudgeReplicaSet(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	resource, ok := obj.(*appsv1.ReplicaSet)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for replicaset: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if factory == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("receive nil ListerFactory")
	}
	if resource == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("replicaset can not be nil")
	}

	for _, c := range resource.Status.Conditions {
		if c.Type == appsv1.ReplicaSetReplicaFailure &&
			c.Status == corev1.ConditionTrue {
			return releaseapi.ResourceStatus{
				Phase:   releaseapi.ResourceFailed,
				Reason:  c.Reason,
				Message: c.Message,
			}, nil
		}
	}

	desiredReplicas := int32(0)
	// use AvailableReplicas rather than status.replicas
	currentReplicas := resource.Status.AvailableReplicas
	if resource.Spec.Replicas != nil {
		desiredReplicas = *resource.Spec.Replicas
	}

	if desiredReplicas == currentReplicas {
		if desiredReplicas == 0 {
			return releaseapi.ResourceStatusFrom(releaseapi.ResourceSuspended), nil
		}
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceRunning), nil
	}
	return releaseapi.ResourceStatusFrom(releaseapi.ResourceProcessing), nil

}
