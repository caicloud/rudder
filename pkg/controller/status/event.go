package status

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func (sc *Controller) enqueueAccessor(obj interface{}) {
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

func (sc *Controller) enqueueEvent(obj interface{}) {
	event, ok := obj.(*corev1.Event)
	if !ok {
		return
	}
	factory := sc.store.SharedInformerFactory()
	var accessor metav1.ObjectMetaAccessor
	var err error
	switch event.InvolvedObject.Kind {
	case "Pod":
		accessor, err = factory.Core().V1().Pods().Lister().Pods(event.InvolvedObject.Namespace).Get(event.InvolvedObject.Name)
		if err != nil {
			return
		}
	case "ReplicaSet":
		accessor, err = factory.Apps().V1().ReplicaSets().Lister().ReplicaSets(event.InvolvedObject.Namespace).Get(event.InvolvedObject.Name)
		if err != nil {
			return
		}
	case "StatefulSet":
		accessor, err = factory.Apps().V1().StatefulSets().Lister().StatefulSets(event.InvolvedObject.Namespace).Get(event.InvolvedObject.Name)
		if err != nil {
			return
		}
	case "DaemonSet":
		accessor, err = factory.Apps().V1().DaemonSets().Lister().DaemonSets(event.InvolvedObject.Namespace).Get(event.InvolvedObject.Name)
		if err != nil {
			return
		}
	default:
		return
	}

	sc.enqueueAccessor(accessor)
}
