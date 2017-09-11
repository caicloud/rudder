package kube

import (
	"math"
	"sort"

	"k8s.io/apimachinery/pkg/runtime"
)

type SortOrder map[string]int8

// InstallOrder is the order in which manifests should be installed (by Kind).
//
// Those occurring earlier in the list get installed before those occurring later in the list.
var InstallOrder = orderBy([]string{
	"Namespace",
	"ResourceQuota",
	"LimitRange",
	"Secret",
	"ConfigMap",
	"PersistentVolume",
	"PersistentVolumeClaim",
	"ServiceAccount",
	"ClusterRole",
	"ClusterRoleBinding",
	"Role",
	"RoleBinding",
	"Service",
	"DaemonSet",
	"Pod",
	"ReplicationController",
	"ReplicaSet",
	"Deployment",
	"StatefulSet",
	"Job",
	"CronJob",
	"Ingress",
})

// UninstallOrder is the order in which manifests should be uninstalled (by Kind).
//
// Those occurring earlier in the list get uninstalled before those occurring later in the list.
var UninstallOrder = orderBy([]string{
	"Ingress",
	"Service",
	"CronJob",
	"Job",
	"StatefulSet",
	"Deployment",
	"ReplicaSet",
	"ReplicationController",
	"Pod",
	"DaemonSet",
	"RoleBinding",
	"Role",
	"ClusterRoleBinding",
	"ClusterRole",
	"ServiceAccount",
	"PersistentVolumeClaim",
	"PersistentVolume",
	"ConfigMap",
	"Secret",
	"LimitRange",
	"ResourceQuota",
	"Namespace",
})

// orderBy create a SortOrder for kinds.
func orderBy(kinds []string) SortOrder {
	order := make(SortOrder, len(kinds))
	for i, k := range kinds {
		order[k] = int8(i)
	}
	return order
}

// Sort sorts a list of objects by kind order.
func (so SortOrder) Sort(objects []runtime.Object) {
	sort.Slice(objects, func(i, j int) bool {
		vi, ok := so[objects[i].GetObjectKind().GroupVersionKind().Kind]
		if !ok {
			vi = math.MaxInt8
		}
		vj, ok := so[objects[j].GetObjectKind().GroupVersionKind().Kind]
		if !ok {
			vj = math.MaxInt8
		}
		return vi < vj
	})
}
