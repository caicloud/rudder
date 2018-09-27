package v1

import (
	"fmt"
	"sort"

	"github.com/caicloud/clientset/listerfactory"
	listerfactorycorev1 "github.com/caicloud/clientset/listerfactory/core/v1"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/clientset/util/event"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	ssetErrorEventCases = []event.EventCase{
		// Liveness and Readiness probe failed
		{corev1.EventTypeWarning, event.FailedCreatePodReason, []string{"exceeded quota"}},
	}
)

func JudgeStatefulSet(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	statefulset, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for statefulset: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if factory == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("receive nil ListerFactory")
	}
	if statefulset == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("statefulset can not be nil")
	}

	podList, err := getPodsFor(factory.Core().V1().Pods(), statefulset)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), err
	}
	oldPods := make([]*corev1.Pod, 0)
	updatePods := make([]*corev1.Pod, 0)
	for _, pod := range podList {
		if pod.Labels[appsv1.StatefulSetRevisionLabel] == statefulset.Status.UpdateRevision {
			updatePods = append(updatePods, pod)
			continue
		}
		oldPods = append(oldPods, pod)
	}

	events, err := listerfactorycorev1.NewEventLister(factory.Client()).Events(statefulset.Namespace).List(labels.Everything())
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), nil
	}
	lastEvent := getLatestEventForStatefulSet(statefulset, events)
	for _, c := range ssetErrorEventCases {
		if c.Match(lastEvent) {
			return releaseapi.ResourceStatus{
				Phase:   releaseapi.ResourceFailed,
				Reason:  lastEvent.Reason,
				Message: lastEvent.Message,
			}, nil
			break
		}
	}

	return JudgeLongRunning(*statefulset.Spec.Replicas, oldPods, updatePods, events), nil
}

func getLatestEventForStatefulSet(sset *appsv1.StatefulSet, events []*corev1.Event) *corev1.Event {
	if len(events) == 0 {
		return nil
	}
	ret := make([]*corev1.Event, 0)
	for _, e := range events {
		if e.InvolvedObject.Kind == "StatefulSet" &&
			e.InvolvedObject.Name == sset.Name &&
			e.InvolvedObject.Namespace == sset.Namespace &&
			e.InvolvedObject.UID == sset.UID {
			ret = append(ret, e)
		}
	}
	if len(ret) == 0 {
		return nil
	}
	sort.Sort(event.EventByLastTimestamp(ret))
	return ret[0]
}
