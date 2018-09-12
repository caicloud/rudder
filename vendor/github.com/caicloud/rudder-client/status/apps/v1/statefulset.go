package v1

import (
	"fmt"

	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func JudgeStatefulSet(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	statefulset, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("unknown type for statefulset: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	if factory == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("receive nil ListerFactory")
	}
	if statefulset == nil {
		return releaseapi.ResourceStatusFrom(""), fmt.Errorf("statefulset can not be nil")
	}

	podList, err := getPodsFor(factory.Core().V1().Pods(), statefulset)
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), err
	}
	oldPods := make([]*corev1.Pod, 0)
	updatePods := make([]*corev1.Pod, 0)
	for _, pod := range podList {
		if pod.Labels[appsv1.StatefulSetRevisionLabel] == statefulset.Status.UpdateRevision {
			updatePods = append(updatePods, pod)
			continue
		}
		oldPods = append(oldPods, pod)
	}
	events, err := factory.Core().V1().Events().Events(statefulset.Namespace).List(labels.Everything())
	if err != nil {
		return releaseapi.ResourceStatusFrom(""), nil
	}

	return JudgeLongRunning(*statefulset.Spec.Replicas, oldPods, updatePods, events), nil
}
