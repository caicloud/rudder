/*
Copyright 2019 caicloud authors. All rights reserved.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// CanaryReleases returns a CanaryReleaseInformer.
	CanaryReleases() CanaryReleaseInformer
	// Releases returns a ReleaseInformer.
	Releases() ReleaseInformer
	// ReleaseHistories returns a ReleaseHistoryInformer.
	ReleaseHistories() ReleaseHistoryInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// CanaryReleases returns a CanaryReleaseInformer.
func (v *version) CanaryReleases() CanaryReleaseInformer {
	return &canaryReleaseInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Releases returns a ReleaseInformer.
func (v *version) Releases() ReleaseInformer {
	return &releaseInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ReleaseHistories returns a ReleaseHistoryInformer.
func (v *version) ReleaseHistories() ReleaseHistoryInformer {
	return &releaseHistoryInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
