package v1

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

func JudgeDaemonSet(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	resource, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for daemonset: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if factory == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("receive nil ListerFactory")
	}
	if resource == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("daemonset can not be nil")
	}
	historyList, err := getHistoriesForDaemonSet(factory.Apps().V1().ControllerRevisions(), resource)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), nil
	}
	history, err := getUpdateHistoryForDaemonSet(resource, historyList)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), nil
	}
	if history == nil {
		return releaseapi.ResourceStatus{
			Phase:  releaseapi.ResourceProcessing,
			Reason: "NoHistory",
		}, nil
	}

	podList, err := getPodsFor(factory.Core().V1().Pods(), resource)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), nil
	}
	oldPods := make([]*corev1.Pod, 0)
	updatePods := make([]*corev1.Pod, 0)
	for _, pod := range podList {
		if pod.Labels[appsv1.DefaultDaemonSetUniqueLabelKey] == history.Labels[appsv1.DefaultDaemonSetUniqueLabelKey] {
			updatePods = append(updatePods, pod)
			continue
		}
		oldPods = append(oldPods, pod)
	}

	events, err := factory.Core().V1().Events().Events(resource.Namespace).List(labels.Everything())
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), nil
	}

	// daemonset has no desired replicas, its value should always be 0
	return JudgeLongRunning(0, oldPods, updatePods, events), nil
}

func getHistoriesForDaemonSet(historyLister appslisters.ControllerRevisionLister, daemonset *appsv1.DaemonSet) ([]*appsv1.ControllerRevision, error) {
	selector, err := metav1.LabelSelectorAsSelector(daemonset.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %v", err)
	}
	// If a deployment with a nil or empty selector creeps in, it should match nothing, not everything.
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
