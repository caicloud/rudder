package kube

import (
	"fmt"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
)

func judgeIPSpecDecreasing(target, existence runtime.Object) (bool, error) {
	// last operation not complete
	_, decreasing := getRuntimeObjectAnnotationValue(existence, AnnoKeySpecifiedIPsDecreasing)
	if decreasing {
		return true, nil
	}
	// ip list diff
	targetIPsRaw, _ := getRuntimeObjectAnnotationValue(target, AnnoKeySpecifiedIPs)
	existenceIPsRaw, _ := getRuntimeObjectAnnotationValue(existence, AnnoKeySpecifiedIPs)
	// string same
	if targetIPsRaw == existenceIPsRaw {
		return false, nil
	}
	// parse
	targetIPs, e := parseSpecifiedIPSetsFromString(targetIPsRaw)
	if e != nil { // do not set invalid ip setting
		return false, e
	}
	existenceIPs, _ := parseSpecifiedIPSetsFromString(existenceIPsRaw)
	if len(existenceIPs) == 0 { // no prev ip setting
		return false, nil
	}
	// check
	m := ipSets2AddressMap(targetIPs)
	for i := range existenceIPs {
		for _, ip := range existenceIPs[i].IPs {
			if _, ok := m[ip]; !ok {
				return true, nil
			}
		}
	}
	return false, nil
}

// applyIpSpecDecreasing deals with release ip spec decreasing extra operations
// 1. update deployment/statefulset annotations
// 2. delete pods not in new list
func (c *client) applyIPSpecDecreasing(client *ResourceClient, namespace string, obj, existence runtime.Object) error {
	// parse and check ips
	targetAnnoValue, _ := getRuntimeObjectAnnotationValue(obj, AnnoKeySpecifiedIPs)
	ipSets, err := parseSpecifiedIPSetsFromString(targetAnnoValue)
	if err != nil {
		return err
	}
	// set deployment/statefulset anno
	tempObj := existence.DeepCopyObject()
	setRuntimeObjectAnnotationValue(tempObj, AnnoKeySpecifiedIPs, targetAnnoValue)
	setRuntimeObjectAnnotationValue(tempObj, AnnoKeySpecifiedIPsDecreasing, "") // rm on final update
	// do deployment/statefulset update
	if _, err = client.Update(tempObj); err != nil {
		return err
	}
	// delete pods on deleted ips
	podClient, err := c.pool.ClientFor(corev1.SchemeGroupVersion.WithKind("Pod"), namespace)
	if err != nil {
		return err
	}
	releaseName, _ := getRuntimeObjectLabelValue(obj, LabelsKeyRelease)
	if len(releaseName) == 0 {
		return fmt.Errorf("get release name empty")
	}
	delList, err := deleteReleasePodsNotInList(podClient, releaseName, ipSets2AddressMap(ipSets))
	for _, del := range delList {
		glog.Infof("applyIpSpecDecreasing[namespace:%v][release:%s][pod:%s][ip:%s] deleted",
			namespace, releaseName, del[0], del[1])
	}
	return err
}

// deleteReleasePodsNotInList deletes release pods not in ipMap, return deleted pods name and ip
func deleteReleasePodsNotInList(podClient *ResourceClient,
	releaseName string, ipMap map[string]struct{}) (delList [][2]string, err error) {
	// list pod of release
	labelReq, err := labels.NewRequirement(LabelsKeyRelease, selection.Equals, []string{releaseName})
	if err != nil {
		return
	}
	labelSelector := labels.NewSelector().Add(*labelReq)
	list, err := podClient.List(metav1.ListOptions{LabelSelector: labelSelector.String()})
	if err != nil {
		return
	}
	podList := list.(*corev1.PodList)
	if podList == nil {
		return
	}
	// do delete
	for i := range podList.Items {
		pod := &podList.Items[i]
		podIP := pod.Status.PodIP
		if podIP == "" {
			continue
		}
		if _, ok := ipMap[podIP]; ok {
			continue
		}
		// del not in list
		if err = podClient.Delete(pod.Name, nil); err != nil && !errors.IsNotFound(err) {
			return
		}
		delList = append(delList, [2]string{pod.Name, podIP})
	}
	return
}
