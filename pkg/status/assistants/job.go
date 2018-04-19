package assistants

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	batch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkJob = batch.SchemeGroupVersion.WithKind("Job")

func JobAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	job, ok := obj.(*batch.Job)
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
