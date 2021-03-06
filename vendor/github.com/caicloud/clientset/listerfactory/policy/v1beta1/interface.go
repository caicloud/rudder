/*
Copyright 2019 caicloud authors. All rights reserved.
*/

// Code generated by listerfactory-gen. DO NOT EDIT.

package v1beta1

import (
	internalinterfaces "github.com/caicloud/clientset/listerfactory/internalinterfaces"
	informers "k8s.io/client-go/informers"
	kubernetes "k8s.io/client-go/kubernetes"
	v1beta1 "k8s.io/client-go/listers/policy/v1beta1"
)

// Interface provides access to all the listers in this group version.
type Interface interface { // PodDisruptionBudgets returns a PodDisruptionBudgetLister
	PodDisruptionBudgets() v1beta1.PodDisruptionBudgetLister
	// PodSecurityPolicies returns a PodSecurityPolicyLister
	PodSecurityPolicies() v1beta1.PodSecurityPolicyLister
}

type version struct {
	client           kubernetes.Interface
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

type infromerVersion struct {
	factory informers.SharedInformerFactory
}

// New returns a new Interface.
func New(client kubernetes.Interface, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{client: client, tweakListOptions: tweakListOptions}
}

// NewFrom returns a new Interface.
func NewFrom(factory informers.SharedInformerFactory) Interface {
	return &infromerVersion{factory: factory}
}

// PodDisruptionBudgets returns a PodDisruptionBudgetLister.
func (v *version) PodDisruptionBudgets() v1beta1.PodDisruptionBudgetLister {
	return &podDisruptionBudgetLister{client: v.client, tweakListOptions: v.tweakListOptions}
}

// PodDisruptionBudgets returns a PodDisruptionBudgetLister.
func (v *infromerVersion) PodDisruptionBudgets() v1beta1.PodDisruptionBudgetLister {
	return v.factory.Policy().V1beta1().PodDisruptionBudgets().Lister()
}

// PodSecurityPolicies returns a PodSecurityPolicyLister.
func (v *version) PodSecurityPolicies() v1beta1.PodSecurityPolicyLister {
	return &podSecurityPolicyLister{client: v.client, tweakListOptions: v.tweakListOptions}
}

// PodSecurityPolicies returns a PodSecurityPolicyLister.
func (v *infromerVersion) PodSecurityPolicies() v1beta1.PodSecurityPolicyLister {
	return v.factory.Policy().V1beta1().PodSecurityPolicies().Lister()
}
