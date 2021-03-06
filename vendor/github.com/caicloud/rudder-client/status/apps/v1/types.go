package v1

import (
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	podstatus "github.com/caicloud/clientset/util/status"

	corev1 "k8s.io/api/core/v1"
)

var (
	// ErrUpdatedRevisionNotExists ...
	ErrUpdatedRevisionNotExists = fmt.Errorf("there is no updated revision found for this resource")
)

// LongRunning ...
type LongRunning interface {
	// PredictRevision predicts longRunning resourceStatus from events
	PredictEvents(events []*corev1.Event) (*releaseapi.ResourceStatus, *corev1.Event)
	// PredictUpdatedRevision returns the updated revision and key
	PredictUpdatedRevision(factory listerfactory.ListerFactory, events []*corev1.Event) (resourceStatus *releaseapi.ResourceStatus, err error)
	// IsUpdatedPod checks if the pod is updated
	// You must call PredictUpdatedRevision before using IsUpdatedPod
	IsUpdatedPod(pod *corev1.Pod) bool
	// DesiredReplics returns the desired replicas of this resource
	DesiredReplics() int32
}

type HyperPod struct {
	Pod    *corev1.Pod
	Status podstatus.PodStatus
}
