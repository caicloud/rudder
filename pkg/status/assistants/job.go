package assistants

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	"k8s.io/apimachinery/pkg/runtime"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
)

var gvkJob = batchv1.SchemeGroupVersion.WithKind("Job")

func JobAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return status.None, fmt.Errorf("unknown type for job: %s", obj.GetObjectKind().GroupVersionKind().String())
	}

	desired := int32(1)
	if job.Spec.Completions != nil {
		desired = *job.Spec.Completions
	}
	succeeded := job.Status.Succeeded
	failed := job.Status.Failed
	switch {
	case desired == succeeded:
		return status.Available, nil
	case failed > 0:
		// TODO(kdada): Check wrong pods for more precise verdict
		return status.Failure, nil
	default:
		return status.Progressing, nil
	}
}
