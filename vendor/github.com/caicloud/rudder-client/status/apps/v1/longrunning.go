package v1

import (
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	podstatus "github.com/caicloud/clientset/util/status"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type longRunning struct {
	factory  listerfactory.ListerFactory
	delegate LongRunning
	obj      runtime.Object
	events   []*corev1.Event
}

func newLongRunning(factory listerfactory.ListerFactory, obj runtime.Object) (*longRunning, error) {
	var delegate LongRunning
	var namespace string
	switch resource := obj.(type) {
	case *appsv1.Deployment:
		namespace = resource.Namespace
		delegate = newDeploymetLongRunning(resource)
	case *appsv1.DaemonSet:
		namespace = resource.Namespace
		delegate = newDaemonSetLongRunning(resource)
	case *appsv1.StatefulSet:
		namespace = resource.Namespace
		delegate = newStatefulSetLongRunning(resource)
	default:
		return nil, fmt.Errorf("unsupported type for %v", resource)
	}

	events, err := factory.Core().V1().Events().Events(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	return &longRunning{
		factory:  factory,
		delegate: delegate,
		obj:      obj,
		events:   events,
	}, nil
}

func (d *longRunning) Judge() (resStatus releaseapi.ResourceStatus, retErr error) {
	// predicts resource status from events
	// remain it util we get PodStatistics
	predictEventsIssue := d.delegate.PredictEvents(d.events)

	// predicts resource status from updated revision and get updated revision key
	predictRevisionIssue, updatedRevisionKey, err := d.delegate.PredictUpdatedRevision(d.factory, d.events)
	if err != nil && err != ErrUpdatedRevisionNotExists {
		return releaseapi.ResourceStatusFrom(""), err
	}

	if err == ErrUpdatedRevisionNotExists && predictEventsIssue == nil {
		// only if there is no predicted events error and no updated revison,
		// then we can return noUpdatedRevisionStatus
		return noUpdatedRevisionStatus, nil
	}

	// we should get pod statistics before returning predict revision status
	// separate pods inte updated and old
	updated, old, err := d.Pods(updatedRevisionKey)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), err
	}

	defer func() {
		// add PodStatistics to resourceStatus when err == nil
		if retErr == nil {
			resStatus.PodStatistics = getPodStatistics(updated, old)
		}
	}()

	if predictEventsIssue != nil {
		return *predictEventsIssue, nil
	}

	if predictRevisionIssue != nil {
		return *predictRevisionIssue, nil
	}

	// judge status from pods
	return d.judge(d.delegate.DesiredReplics(), updated, old), nil
}

func (d *longRunning) Pods(updatedRevisionKey string) (updated, old []HyperPod, err error) {
	// get pods
	podList, err := getPodsFor(d.factory.Core().V1().Pods(), d.obj)
	if err != nil {
		return nil, nil, err
	}

	for _, pod := range podList {
		// judge pod status
		status := podstatus.JudgePodStatus(pod, d.events)
		if d.delegate.IsUpdatedPod(pod, updatedRevisionKey) {
			updated = append(updated, HyperPod{pod, status})
		} else {
			old = append(old, HyperPod{pod, status})
		}
	}
	return updated, old, nil
}

func (d *longRunning) judge(desiredReplicas int32, updatePods, oldPods []HyperPod) releaseapi.ResourceStatus {
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
		// scaling up
		return releaseapi.ResourceStatus{
			Phase:  releaseapi.ResourceProgressing,
			Reason: "ZeroReplicas",
		}
	}

	// realReplicas > 0
	running := true
	// check the updated pods
	// if one of updated pods if in Abnormal, the Resource is Failed
	for _, pod := range updatePods {
		podStatus := pod.Status
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
			podStatus := pod.Status
			if podStatus.State == podstatus.PodAbnormal {
				return releaseapi.ResourceStatus{
					Phase:   releaseapi.ResourceFailed,
					Reason:  podStatus.Reason,
					Message: podStatus.Message,
				}
			}
		}
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceUpdating)
	}

	// The running status prerequisite:
	// 1. no old pods
	// 2. all updated pods are Running
	// 3. the number of running pods == desired replicas
	if running && realReplicas == int(desiredReplicas) {
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceRunning)
	}

	// Progressing
	return releaseapi.ResourceStatusFrom(releaseapi.ResourceProgressing)
}
