package v1

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/clientset/util/event"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

var (
	dsetErrorEventCases = []event.EventCase{
		// match all FailedCreate
		{EventType: corev1.EventTypeWarning, Reason: event.FailedCreatePodReason},
	}
)

func JudgeDaemonSet(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	daemonset, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for daemonset: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if daemonset == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("daemonset can not be nil")
	}

	lr, err := newLongRunning(factory, daemonset)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), err
	}
	return lr.Judge()

}

type daemonsetLongRunning struct {
	daemonset *appsv1.DaemonSet
}

func newDaemonSetLongRunning(daemonset *appsv1.DaemonSet) LongRunning {
	return &daemonsetLongRunning{daemonset}
}

func (d *daemonsetLongRunning) PredictUpdatedRevision(factory listerfactory.ListerFactory, events []*corev1.Event) (*releaseapi.ResourceStatus, string, error) {
	daemonset := d.daemonset

	historyList, err := getHistoriesForDaemonSet(factory.Apps().V1().ControllerRevisions(), daemonset)
	if err != nil {
		return nil, "", err
	}
	history, err := getUpdateHistoryForDaemonSet(daemonset, historyList)
	if err != nil {
		return nil, "", err
	}
	if history == nil {
		return nil, "", ErrUpdatedRevisionNotExists
	}

	return nil, getLabel(history, appsv1.DefaultDaemonSetUniqueLabelKey), nil
}

func (d *daemonsetLongRunning) IsUpdatedPod(pod *corev1.Pod, updatedRevisionKey string) bool {
	return getLabel(pod, appsv1.DefaultDaemonSetUniqueLabelKey) == updatedRevisionKey
}

func (d *daemonsetLongRunning) PredictEvents(events []*corev1.Event) *releaseapi.ResourceStatus {
	lastEvent := getLatestEventFor(d.daemonset.GroupVersionKind().Kind, d.daemonset, events)
	for _, c := range dsetErrorEventCases {
		if c.Match(lastEvent) {
			return &releaseapi.ResourceStatus{
				Phase:   releaseapi.ResourceFailed,
				Reason:  lastEvent.Reason,
				Message: lastEvent.Message,
			}
		}
	}
	return nil
}

func (d *daemonsetLongRunning) PredictRevision(updatedRevision interface{}, events []*corev1.Event) *releaseapi.ResourceStatus {
	return nil
}

func (d *daemonsetLongRunning) DesiredReplics() int32 {
	// daemonset has no desired replicas, its value should always be 0
	return d.daemonset.Status.DesiredNumberScheduled
}

func getHistoriesForDaemonSet(historyLister appslisters.ControllerRevisionLister, daemonset *appsv1.DaemonSet) ([]*appsv1.ControllerRevision, error) {
	selector, err := metav1.LabelSelectorAsSelector(daemonset.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %v", err)
	}
	// If a daemonset with a nil or empty selector creeps in, it should match nothing, not everything.
	if selector.Empty() {
		return nil, nil
	}

	return historyLister.ControllerRevisions(daemonset.Namespace).List(selector)
}

func getUpdateHistoryForDaemonSet(daemonset *appsv1.DaemonSet, histories []*appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	patch, err := getPatch(daemonset)
	if err != nil {
		return nil, err
	}

	for _, history := range histories {
		if bytes.Equal(patch, history.Data.Raw) {
			return history, nil
		}
	}
	return nil, nil
}

// getPatch returns a strategic merge patch that can be applied to restore a Daemonset to a
// previous version. If the returned error is nil the patch is valid. The current state that we save is just the
// PodSpecTemplate. We can modify this later to encompass more state (or less) and remain compatible with previously
// recorded patches.
func getPatch(ds *appsv1.DaemonSet) ([]byte, error) {
	dsBytes, err := json.Marshal(ds)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	err = json.Unmarshal(dsBytes, &raw)
	if err != nil {
		return nil, err
	}
	objCopy := make(map[string]interface{})
	specCopy := make(map[string]interface{})

	// Create a patch of the DaemonSet that replaces spec.template
	spec := raw["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	specCopy["template"] = template
	template["$patch"] = "replace"
	objCopy["spec"] = specCopy
	patch, err := json.Marshal(objCopy)
	return patch, err
}
