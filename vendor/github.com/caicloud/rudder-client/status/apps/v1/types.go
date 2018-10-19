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
	ErrUpdatedRevisionNotExists = fmt.Errorf("There is no updated revision found for this resource")
)

// LongRunning ...
type LongRunning interface {
	// UpdatedRevision returns the updated revision and key
	UpdatedRevision(factory listerfactory.ListerFactory) (updatedRevision interface{}, updatedRevisionKey string, err error)
	// IsUpdatedPod checks if the pod is updated
	IsUpdatedPod(pod *corev1.Pod, updateRevisionKey string) bool
	// Predict predicts resourceStatus from updatedRevision and events before judging from pods status
	Predict(updatedRevision interface{}, events []*corev1.Event) (*releaseapi.ResourceStatus, error)
	// DesiredReplics returns the desired replicas of this resource
	DesiredReplics() int32
}

type HyperPod struct {
	Pod    *corev1.Pod
	Status podstatus.PodStatus
}
