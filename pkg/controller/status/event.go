package status

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func (sc *StatusController) enqueueAccessor(obj interface{}) {
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		obj = tombstone.Obj
	}

	// pod, ok := obj.(*corev1.Pod)
	accessor, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		return
	}

	// FIXME:
	releaseName := ""
	annotations := accessor.GetObjectMeta().GetAnnotations()
	labels := accessor.GetObjectMeta().GetLabels()
	namepsace := accessor.GetObjectMeta().GetNamespace()
	if len(annotations) > 0 && annotations["helm.sh/release"] != "" {
		releaseName = annotations["helm.sh/release"]
	} else if len(labels) > 0 && labels["controller.caicloud.io/release"] != "" {
		releaseName = labels["controller.caicloud.io/release"]
	}
	if releaseName == "" {
		return
	}
	sc.workqueue.Enqueue(cache.ExplicitKey(namepsace + "/" + releaseName))
}

func (sc *StatusController) enqueueEvent(obj interface{}) {
	event, ok := obj.(*corev1.Event)
	if !ok {
		return
	}
	if event.InvolvedObject.Kind != "Pod" &&
		event.InvolvedObject.Kind != "ReplicaSet" {
		return
	}
	factory := sc.store.SharedInformerFactory()
	if event.InvolvedObject.Kind == "Pod" {
		pod, err := factory.Core().V1().Pods().Lister().Pods(event.InvolvedObject.Namespace).Get(event.InvolvedObject.Name)
		if err != nil {
			return
		}
		sc.enqueueAccessor(pod)
	} else if event.InvolvedObject.Kind == "ReplicaSet" {
		rs, err := factory.Apps().V1().ReplicaSets().Lister().ReplicaSets(event.InvolvedObject.Namespace).Get(event.InvolvedObject.Name)
		if err != nil {
			return
		}
		sc.enqueueAccessor(rs)
	}

}
