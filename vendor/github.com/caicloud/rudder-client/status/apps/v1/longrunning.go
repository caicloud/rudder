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
	// get updatedRevision
	updatedRevision, updatedRevisionKey, err := d.delegate.UpdatedRevision(d.factory)
	if err != nil && err != ErrUpdatedRevisionNotExists {
		return releaseapi.ResourceStatusFrom(""), err
	}

	if err == ErrUpdatedRevisionNotExists {
		return releaseapi.ResourceStatus{
			Phase:   releaseapi.ResourceProgressing,
			Reason:  "NoUpdateRevision",
			Message: err.Error(),
		}, nil
	}

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

	// predict resourceStatus from updatedRevision or events
	predict, err := d.delegate.Predict(updatedRevision, d.events)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), err
	}
	if predict != nil {
		return *predict, nil
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

// [Note] DaemonSet has no desired replicas in spec, so desiredReplicas should always be 0
// we only use the desiredReplicas to judge whether the long running workload is Suspended
func (d *longRunning) judge(desiredReplicas int32, oldPods, updatePods []HyperPod) releaseapi.ResourceStatus {
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
		for _, pod := range updatePods {
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

	// no old pods and all updated pods is Running
	if running {
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceRunning)
	}

	// Progressing
	return releaseapi.ResourceStatusFrom(releaseapi.ResourceProgressing)
}
