package v2alpha1

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	batch "k8s.io/api/batch/v1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkCronJob = batchv2alpha1.SchemeGroupVersion.WithKind("CronJob")
var gvkJob = batch.SchemeGroupVersion.WithKind("Job")

func CronJobAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	cj, ok := obj.(*batchv2alpha1.CronJob)
	if !ok {
		return status.None, fmt.Errorf("unknown type for cron job: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	processing := 0
	for _, obj := range cj.Status.Active {
		informer, err := store.InformerFor(gvkJob)
		if err != nil {
			return status.None, err
		}
		obj, err := informer.Lister().ByNamespace(obj.Namespace).Get(obj.Name)
		if err != nil {
			return status.None, err
		}
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
			continue
		case failed > 0:
			processing++
		default:
			return status.Failure, nil
		}
	}
	if processing > 0 {
		return status.Progressing, nil
	}
	return status.Available, nil
}
