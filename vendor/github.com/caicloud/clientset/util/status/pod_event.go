package status

import (
	"sort"

	"github.com/caicloud/clientset/util/event"

	"k8s.io/api/core/v1"
)

var (
	errorEventCases = []event.EventCase{
		{
			// Liveness and Readiness probe failed
			EventType: v1.EventTypeWarning,
			Reason:    event.ContainerUnhealthy,
			MsgKeys:   []string{"probe failed"},
		},
		{
			// failed to mount volume
			EventType: v1.EventTypeWarning,
			Reason:    event.FailedMountVolume,
		},
	}
)

// JudgePodStatus judges the current status of pod from Pod.Status
// and correct it with events.
func JudgePodStatus(pod *v1.Pod, events []*v1.Event) PodStatus {
	if pod == nil {
		return PodStatus{}
	}

	status := judgePod(pod)
	// only the latest event is useful
	e := getLatestEventForPod(pod, events)
	for _, c := range errorEventCases {
		if c.Match(e) {
			status.Phase = PodError
			status.Reason = e.Reason
			status.Message = e.Message
			break
		}
	}

	switch status.Phase {
	case PodRunning, PodSucceeded:
		status.State = PodNormal
		status.Ready = true
	case PodFailed, PodError, PodUnknown:
		status.State = PodAbnormal
		status.Ready = false
	default:
		status.State = PodUncertain
	}

	return status
}

func getLatestEventForPod(pod *v1.Pod, events []*v1.Event) *v1.Event {
	if len(events) == 0 {
		return nil
	}
	ret := make([]*v1.Event, 0)

	for _, e := range events {
		if e.InvolvedObject.Kind == "Pod" &&
			e.InvolvedObject.Name == pod.Name &&
			e.InvolvedObject.Namespace == pod.Namespace &&
			e.InvolvedObject.UID == pod.UID {
			ret = append(ret, e)
		}
	}

	if len(ret) == 0 {
		return nil
	}

	sort.Sort(event.EventByLastTimestamp(ret))
	return ret[0]
}
