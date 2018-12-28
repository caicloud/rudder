package v1

import (
	"fmt"
	"sort"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

func JudgeDeployment(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for deployment: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if deployment == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("deployment can not be nil")
	}

	lr, err := newLongRunning(factory, deployment)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), err
	}
	return lr.Judge()
}

type deploymentLongRunning struct {
	deployment *appsv1.Deployment
}

func newDeploymetLongRunning(deployment *appsv1.Deployment) LongRunning {
	return &deploymentLongRunning{deployment}
}

func (d *deploymentLongRunning) PredictUpdatedRevision(factory listerfactory.ListerFactory, events []*corev1.Event) (*releaseapi.ResourceStatus, string, error) {
	deployment := d.deployment

	rsList, err := getReplicaSetsforDeployment(factory.Apps().V1().ReplicaSets(), deployment)
	if err != nil {
		return nil, "", err
	}

	rs := getUpdatedReplicaSetForDeployment(deployment, rsList)
	if rs == nil {
		return nil, "", ErrUpdatedRevisionNotExists
	}

	// a replica set when one of its pods fails to be created
	// due to insufficient quota, limit ranges, pod security policy, node selectors, etc. or deleted
	// due to kubelet being down or finalizers are failing.
	for _, c := range rs.Status.Conditions {
		if c.Type == appsv1.ReplicaSetReplicaFailure &&
			c.Status == corev1.ConditionTrue {
			return &releaseapi.ResourceStatus{
				Phase:   releaseapi.ResourceFailed,
				Reason:  c.Reason,
				Message: c.Message,
			}, string(rs.UID), nil
		}
	}
	return nil, string(rs.UID), nil
}

func (d *deploymentLongRunning) IsUpdatedPod(pod *corev1.Pod, updatedRevisionKey string) bool {
	latest := false
	for _, owner := range pod.OwnerReferences {
		if string(owner.UID) == updatedRevisionKey {
			latest = true
		}
	}
	return latest
}

func (d *deploymentLongRunning) PredictEvents(events []*corev1.Event) *releaseapi.ResourceStatus {
	return nil
}

func (d *deploymentLongRunning) DesiredReplics() int32 {
	return *d.deployment.Spec.Replicas
}

func getReplicaSetsforDeployment(rslister appslisters.ReplicaSetLister, deployment *appsv1.Deployment) ([]*appsv1.ReplicaSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %v", err)
	}
	// If a deployment with a nil or empty selector creeps in, it should match nothing, not everything.
	if selector.Empty() {
		return nil, nil
	}

	rsList, err := rslister.ReplicaSets(deployment.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	ret := make([]*appsv1.ReplicaSet, 0)

LOOP:
	for _, rs := range rsList {
		for _, owner := range rs.OwnerReferences {
			if owner.UID == deployment.UID &&
				owner.Name == deployment.Name {
				ret = append(ret, rs)
				continue LOOP
			}
		}
	}

	return ret, nil
}

// Note(li-ang): Deployment will stop adding pod-template-hash labels/selector to ReplicaSets and Pods it adopts. Resources created by Deployments are not affected (will still have pod-template-hash labels/selector).  ([#61615](https://github.com/kubernetes/kubernetes/pull/61615), [@janetkuo](https://github.com/janetkuo))

// ReplicaSetsByCreationTimestamp sorts a list of ReplicaSet by creation timestamp, using their names as a tie breaker.
type ReplicaSetsByCreationTimestamp []*appsv1.ReplicaSet

func (o ReplicaSetsByCreationTimestamp) Len() int      { return len(o) }
func (o ReplicaSetsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ReplicaSetsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// getUpdatedReplicaSetForDeployment returns the updated RS this given deployment targets (the one with the same pod template).
func getUpdatedReplicaSetForDeployment(deployment *appsv1.Deployment, rsList []*appsv1.ReplicaSet) *appsv1.ReplicaSet {
	sort.Sort(ReplicaSetsByCreationTimestamp(rsList))
	for i := range rsList {
		if EqualIgnoreHash(&rsList[i].Spec.Template, &deployment.Spec.Template) {
			// In rare cases, such as after cluster upgrades, Deployment may end up with
			// having more than one new ReplicaSets that have the same template as its template,
			// see https://github.com/kubernetes/kubernetes/issues/40415
			// We deterministically choose the oldest new ReplicaSet.
			return rsList[i]
		}
	}
	// new ReplicaSet does not exist.
	return nil
}

// EqualIgnoreHash returns true if two given podTemplateSpec are equal, ignoring the diff in value of Labels[pod-template-hash]
// We ignore pod-template-hash because:
// 1. The hash result would be different upon podTemplateSpec API changes
//    (e.g. the addition of a new field will cause the hash code to change)
// 2. The deployment template won't have hash labels
func EqualIgnoreHash(template1, template2 *corev1.PodTemplateSpec) bool {
	t1Copy := template1.DeepCopy()
	t2Copy := template2.DeepCopy()
	// Remove hash labels from template.Labels before comparing
	delete(t1Copy.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	delete(t2Copy.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	return apiequality.Semantic.DeepEqual(t1Copy, t2Copy)
}
