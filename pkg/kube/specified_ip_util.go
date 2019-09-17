package kube

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"

	"github.com/caicloud/go-common/interfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// consts
const (
	AnnoKeySpecifiedIPs           = "ipam.caicloud.io/specified-ips"
	AnnoKeySpecifiedIPsDecreasing = "controller.caicloud.io/specified-ips-decreasing"
	LabelsKeyRelease              = "controller.caicloud.io/release"
)

// parse

func parseSpecifiedIPSetsFromString(s string) ([]WorkloadSubnetSpecifiedIPs, error) {
	var sets []WorkloadSubnetSpecifiedIPs
	if len(s) == 0 {
		return sets, nil
	}
	e := json.Unmarshal([]byte(s), &sets)
	if e != nil {
		return nil, e
	}
	e = validateSpecifiedIPsSets(sets)
	if e != nil {
		return nil, e
	}
	formatSpecifiedIPsSets(sets)
	return sets, nil
}

// kube meta util

func getMetaObjectFromRuntimeObject(obj runtime.Object) metav1.Object {
	ao, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		return nil
	}
	if interfaces.IsNil(ao) {
		return nil
	}
	return ao.GetObjectMeta()
}

func getRuntimeObjectAnnotationValue(obj runtime.Object, key string) (string, bool) {
	mo := getMetaObjectFromRuntimeObject(obj)
	if interfaces.IsNil(mo) {
		return "", false
	}
	m := mo.GetAnnotations()
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

func setRuntimeObjectAnnotationValue(obj runtime.Object, key, value string) (updated bool) {
	mo := getMetaObjectFromRuntimeObject(obj)
	if interfaces.IsNil(mo) {
		return
	}
	m := mo.GetAnnotations()
	if m == nil {
		m = make(map[string]string, 1)
	}
	prev, ok := m[key]
	updated = !ok || prev != value
	m[key] = value
	return
}

func getRuntimeObjectLabelValue(obj runtime.Object, key string) (string, bool) {
	mo := getMetaObjectFromRuntimeObject(obj)
	if interfaces.IsNil(mo) {
		return "", false
	}
	m := mo.GetLabels()
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

// specified ips calculate

func ipSets2AddressMap(sets []WorkloadSubnetSpecifiedIPs) map[string]struct{} {
	mSize := 0
	for i := range sets {
		mSize += len(sets[i].IPs)
	}
	m := make(map[string]struct{}, mSize)
	for i := range sets {
		for _, ip := range sets[i].IPs {
			m[ip] = struct{}{}
		}
	}
	return m
}

// SAME WITH APP-ADMIN: TODO: format in one place

// ip setting objects

// WorkloadSubnetSpecifiedIPs represents specified ips of one subnet
type WorkloadSubnetSpecifiedIPs struct {
	// Network, the name of network
	Network string `json:"network"`
	// Subnet, id of subnet
	Subnet string   `json:"subnet"`
	IPs    []string `json:"ips"`
}

// validateSpecifiedIPsSets will check is sets legal
func validateSpecifiedIPsSets(sets []WorkloadSubnetSpecifiedIPs) error {
	if len(sets) == 0 {
		return nil
	}
	ipCount := 0
	// set check
	keyMap := make(map[string]struct{}, len(sets))
	for _, set := range sets {
		// sub object check
		if set.Network == "" || len(set.IPs) == 0 {
			return fmt.Errorf("network and ips must be set")
		}
		// network+subnet duplicate
		key := set.Network + "/" + set.Subnet
		if _, exist := keyMap[key]; exist {
			return fmt.Errorf("duplicate network and subnet %v", key)
		}
		keyMap[key] = struct{}{}
		ipCount += len(set.IPs)
	}
	// ip check
	ipMap := make(map[string]struct{}, ipCount)
	for _, set := range sets {
		// ip
		for _, ip := range set.IPs {
			// format
			if net.ParseIP(ip) == nil {
				return fmt.Errorf("bad ip address %v", ip)
			}
			// duplicate
			if _, exist := ipMap[ip]; exist {
				return fmt.Errorf("duplicate ip %v", ip)
			}
			ipMap[ip] = struct{}{}
		}
	}
	return nil
}

// formatSpecifiedIPsSets will format sets
func formatSpecifiedIPsSets(sets []WorkloadSubnetSpecifiedIPs) {
	if len(sets) == 0 {
		return
	}
	// if validated, net+subnet will be unique
	sort.Slice(sets, func(i, j int) bool {
		if sets[i].Network != sets[j].Network {
			return sets[i].Network < sets[j].Network
		}
		return sets[i].Subnet < sets[j].Subnet
	})
	// format unit
	for i := range sets {
		sort.Strings(sets[i].IPs)
	}
}
