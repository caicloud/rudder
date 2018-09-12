package v1

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	podstatus "github.com/caicloud/clientset/util/status"

	"k8s.io/api/core/v1"
)

// JudgeLongRunning ...
//
// [Note] DaemonSet has no desired replicas in spec, so desiredReplicas should always be 0
// we only use the desiredReplicas to judge whether the long running workload is Suspended
func JudgeLongRunning(desiredReplicas int32, oldPods, updatePods []*v1.Pod, events []*v1.Event) releaseapi.ResourceStatus {
	// get updated and old replicas
	updateReplicas := len(updatePods)
	oldReplicas := len(oldPods)

	realReplicas := updateReplicas + oldReplicas

	if realReplicas == 0 {
		if desiredReplicas == 0 {
			return releaseapi.ResourceStatus{
				Phase:  releaseapi.ResourceSuspended,
				Reason: "DesiredZeroReplicas",
			}
		}
		return releaseapi.ResourceStatus{
			Phase:  releaseapi.ResourceProcessing,
			Reason: "ZeroReplicas",
		}
	}

	// realReplicas > 0
	running := true

	// check the updated pods
	// if one of updated pods if in Abnormal, the Resource is Failed
	for _, pod := range updatePods {
		podStatus := podstatus.JudgePodStatus(pod, events)
		if podStatus.State == podstatus.PodAbnormal {
			return releaseapi.ResourceStatus{
				Phase:   releaseapi.ResourceFailed,
				Reason:  podStatus.Reason,
				Message: podStatus.Message,
			}
		}
		if podStatus.Phase != podstatus.PodRunning {
			running = false
		}
	}

	// check old pods
	// if one of old pods if in Abnormal, the Resource is Failed
	// otherwise it is in Updating
	if oldReplicas > 0 {
		for _, pod := range oldPods {
			podStatus := podstatus.JudgePodStatus(pod, events)
			if podStatus.State == podstatus.PodAbnormal {
				return releaseapi.ResourceStatus{
					Phase:   releaseapi.ResourceFailed,
					Reason:  podStatus.Reason,
					Message: podStatus.Message,
				}
			}
		}
		return releaseapi.ResourceStatus{
			Phase: releaseapi.ResourceUpdating,
		}
	}

	// no old pods and all updated pods is Running
	if running {
		return releaseapi.ResourceStatus{
			Phase: releaseapi.ResourceRunning,
		}
	}

	// Progressing
	return releaseapi.ResourceStatus{
		Phase: releaseapi.ResourceProcessing,
	}
}
