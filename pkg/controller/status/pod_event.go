package status

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (sc *StatusController) enqueuePod(obj interface{}) {
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		obj = tombstone.Obj
	}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}

	// FIXME:
	releaseName := ""
	if pod.Annotations != nil && pod.Annotations["helm.sh/release"] != "" {
		releaseName = pod.Annotations["helm.sh/release"]
	} else if pod.Labels != nil && pod.Labels["controller.caicloud.io/release"] != "" {
		releaseName = pod.Labels["controller.caicloud.io/release"]
	}
	if releaseName == "" {
		return
	}
	sc.workqueue.Enqueue(cache.ExplicitKey(pod.Namespace + "/" + releaseName))
}

func (sc *StatusController) enqueueEvent(obj interface{}) {
	event, ok := obj.(*corev1.Event)
	if !ok {
		return
	}
	if event.InvolvedObject.Kind != "Pod" {
		return
	}
	factory := sc.store.SharedInformerFactory()
	pod, err := factory.Core().V1().Pods().Lister().Pods(event.InvolvedObject.Namespace).Get(event.InvolvedObject.Name)
	if err != nil {
		return
	}
	sc.enqueuePod(pod)

}
