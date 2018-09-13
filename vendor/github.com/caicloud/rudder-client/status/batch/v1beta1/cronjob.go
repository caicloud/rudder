package v1beta1

import (
	"fmt"
	"sort"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	statusbatchv1 "github.com/caicloud/rudder-client/status/batch/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	batchlisters "k8s.io/client-go/listers/batch/v1"
)

// SortJobByCreationTimestamp sorts job by creation time
type SortJobByCreationTimestamp []*batchv1.Job

func (x SortJobByCreationTimestamp) Len() int {
	return len(x)
}

func (x SortJobByCreationTimestamp) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x SortJobByCreationTimestamp) Less(i, j int) bool {
	itime := x[i].CreationTimestamp
	jtime := x[j].CreationTimestamp
	if itime.After(jtime.Time) {
		return true
	}
	return false
}

func JudgeCronJob(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	cronjob, ok := obj.(*batchv1beta1.CronJob)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for CronJob: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if cronjob.Spec.Suspend != nil && *cronjob.Spec.Suspend {
		return releaseapi.ResourceStatusFrom(releaseapi.ResourceSuspended), nil
	}
	if len(cronjob.Status.Active) > 0 {
		return releaseapi.ResourceStatus{
			Phase:   releaseapi.ResourceProgressing,
			Reason:  "JobRunning",
			Message: fmt.Sprintf("there are %v jobs are running", len(cronjob.Status.Active)),
		}, nil
	}
	jobList, err := getJobForCronJob(factory.Batch().V1().Jobs(), cronjob)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), err
	}

	if len(jobList) == 0 {
		return releaseapi.ResourceStatusFrom(releaseapi.ResourcePending), nil
	}

	sort.Sort(SortJobByCreationTimestamp(jobList))

	return statusbatchv1.JudgeJob(factory, jobList[0])
}

func getJobForCronJob(joblister batchlisters.JobLister, cronjob *batchv1beta1.CronJob) ([]*batchv1.Job, error) {
	selector := labels.SelectorFromSet(cronjob.Spec.JobTemplate.Labels)
	if selector.Empty() {
		return nil, nil
	}
	return joblister.Jobs(cronjob.Namespace).List(selector)
}
