package status

import (
	"sort"

	"k8s.io/api/core/v1"
)

var (
	errorEventCases = []eventCase{
		// Liveness and Readiness probe failed
		{v1.EventTypeWarning, EventUnhealthy, []string{"probe failed"}},
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
	event := getLatestEventForPod(pod, events)
	for _, c := range errorEventCases {
		if c.match(event) {
			status.Phase = PodError
			status.Reason = event.Reason
			status.Message = event.Message
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

	for _, event := range events {
		if event.InvolvedObject.Kind == "Pod" &&
			event.InvolvedObject.Name == pod.Name &&
			event.InvolvedObject.Namespace == pod.Namespace &&
			event.InvolvedObject.UID == pod.UID {
			ret = append(ret, event)
		}
	}

	if len(ret) == 0 {
		return nil
	}

	sort.Sort(eventByLastTimestamp(ret))
	return ret[0]
}
