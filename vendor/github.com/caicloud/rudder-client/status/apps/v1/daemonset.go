package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/clientset/util/event"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

var (
	dsetErrorEventCases = []event.EventCase{
		// match all FailedCreate
		{EventType: corev1.EventTypeWarning, Reason: event.FailedCreatePodReason},
		{EventType: corev1.EventTypeWarning, Reason: event.FailedPlacementPodReason},
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
	daemonset       *appsv1.DaemonSet
	updatedRevision *appsv1.ControllerRevision
}

func newDaemonSetLongRunning(daemonset *appsv1.DaemonSet) LongRunning {
	return &daemonsetLongRunning{
		daemonset: daemonset,
	}
}

func (d *daemonsetLongRunning) PredictUpdatedRevision(factory listerfactory.ListerFactory, events []*corev1.Event) (*releaseapi.ResourceStatus, error) {
	daemonset := d.daemonset

	historyList, err := getHistoriesForDaemonSet(factory.Apps().V1().ControllerRevisions(), daemonset)
	if err != nil {
		return nil, err
	}
	d.updatedRevision, err = getUpdateHistoryForDaemonSet(daemonset, historyList)
	if err != nil {
		return nil, err
	}
	if d.updatedRevision == nil {
		return nil, ErrUpdatedRevisionNotExists
	}

	return nil, nil
}

func (d *daemonsetLongRunning) IsUpdatedPod(pod *corev1.Pod) bool {
	if d.updatedRevision == nil {
		return false
	}

	hash := getLabel(d.updatedRevision, appsv1.DefaultDaemonSetUniqueLabelKey)

	generation, err := GetTemplateGeneration(d.daemonset)
	if err != nil {
		generation = nil
	}

	return IsUpdatedPodOfDaemonSet(pod, hash, generation)
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

// ControllerRevisionByRevision sorts a list of ControllerRevision by revision (desc), using their names as a tie breaker.
type ControllerRevisionByRevision []*appsv1.ControllerRevision

func (o ControllerRevisionByRevision) Len() int      { return len(o) }
func (o ControllerRevisionByRevision) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ControllerRevisionByRevision) Less(i, j int) bool {
	if o[i].Revision == o[i].Revision {
		return o[i].Name < o[j].Name
	}
	return o[i].Revision > o[i].Revision
}

func getUpdateHistoryForDaemonSet(daemonset *appsv1.DaemonSet, histories []*appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	patch, err := getPatch(daemonset)
	if err != nil {
		return nil, err
	}

	sort.Sort(ControllerRevisionByRevision(histories))

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

// GetTemplateGeneration gets the template generation associated with a v1.DaemonSet by extracting it from the
// deprecated annotation. If no annotation is found nil is returned. If the annotation is found and fails to parse
// nil is returned with an error. If the generation can be parsed from the annotation, a pointer to the parsed int64
// value is returned.
func GetTemplateGeneration(ds *appsv1.DaemonSet) (*int64, error) {
	annotation, found := ds.Annotations[appsv1.DeprecatedTemplateGeneration]
	if !found {
		return nil, nil
	}
	generation, err := strconv.ParseInt(annotation, 10, 64)
	if err != nil {
		return nil, err
	}
	return &generation, nil
}

// IsUpdatedPodOfDaemonSet checks if pod contains label value that either matches templateGeneration or hash
func IsUpdatedPodOfDaemonSet(pod *corev1.Pod, hash string, dsTemplateGeneration *int64) bool {
	// Compare with hash to see if the pod is updated, need to maintain backward compatibility of templateGeneration
	templateMatches := dsTemplateGeneration != nil &&
		pod.Labels[extensions.DaemonSetTemplateGenerationKey] == fmt.Sprint(dsTemplateGeneration)
	hashMatches := len(hash) > 0 && pod.Labels[appsv1.DefaultDaemonSetUniqueLabelKey] == hash
	return hashMatches || templateMatches
}
