package v1

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	corelisters "k8s.io/client-go/listers/core/v1"
)

func getPodsFor(podLister corelisters.PodLister, obj runtime.Object) ([]*corev1.Pod, error) {
	var selector labels.Selector
	var namespace string
	var err error
	switch resource := obj.(type) {
	case *appsv1.Deployment:
		namespace = resource.Namespace
		selector, err = metav1.LabelSelectorAsSelector(resource.Spec.Selector)
	case *appsv1.DaemonSet:
		namespace = resource.Namespace
		selector, err = metav1.LabelSelectorAsSelector(resource.Spec.Selector)
	case *appsv1.StatefulSet:
		namespace = resource.Namespace
		selector, err = metav1.LabelSelectorAsSelector(resource.Spec.Selector)
	default:
		return nil, fmt.Errorf("%v is no supported", obj)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %v", err)
	}
	// If a resource with a nil or empty selector creeps in, it should match nothing, not everything.
	if selector.Empty() {
		return nil, nil
	}
	return podLister.Pods(namespace).List(selector)
}
