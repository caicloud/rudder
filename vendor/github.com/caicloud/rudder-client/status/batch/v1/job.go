package v1

import (
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func JudgeJob(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for Job: %s", obj.GetObjectKind().GroupVersionKind().String())
	}

	desired := int32(1)
	if job.Spec.Completions != nil {
		desired = *job.Spec.Completions
	}
	succeeded := job.Status.Succeeded
	// job.Status
	if desired == succeeded {
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceSucceeded), nil
	}

	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return releaseapi.ResourceStatus{
				Phase:   releaseapi.ResourceFailed,
				Reason:  c.Reason,
				Message: c.Message,
			}, nil
		}
	}

	return releaseapi.ResourceStatusFrom(releaseapi.ResourceProgressing), nil
}
