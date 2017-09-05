/*
Copyright 2017 caicloud authors. All rights reserved.
*/

package kubernetes

import (
	configv1alpha1 "github.com/caicloud/clientset/kubernetes/typed/config/v1alpha1"
	releasev1alpha1 "github.com/caicloud/clientset/kubernetes/typed/release/v1alpha1"
	glog "github.com/golang/glog"
	kubernetes "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	kubernetes.Interface
	ConfigV1alpha1() configv1alpha1.ConfigV1alpha1Interface
	// Deprecated: please explicitly pick a version if possible.
	Config() configv1alpha1.ConfigV1alpha1Interface
	ReleaseV1alpha1() releasev1alpha1.ReleaseV1alpha1Interface
	// Deprecated: please explicitly pick a version if possible.
	Release() releasev1alpha1.ReleaseV1alpha1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*kubernetes.Clientset
	*configv1alpha1.ConfigV1alpha1Client
	*releasev1alpha1.ReleaseV1alpha1Client
}

// ConfigV1alpha1 retrieves the ConfigV1alpha1Client
func (c *Clientset) ConfigV1alpha1() configv1alpha1.ConfigV1alpha1Interface {
	if c == nil {
		return nil
	}
	return c.ConfigV1alpha1Client
}

// Deprecated: Config retrieves the default version of ConfigClient.
// Please explicitly pick a version.
func (c *Clientset) Config() configv1alpha1.ConfigV1alpha1Interface {
	if c == nil {
		return nil
	}
	return c.ConfigV1alpha1Client
}

// ReleaseV1alpha1 retrieves the ReleaseV1alpha1Client
func (c *Clientset) ReleaseV1alpha1() releasev1alpha1.ReleaseV1alpha1Interface {
	if c == nil {
		return nil
	}
	return c.ReleaseV1alpha1Client
}

// Deprecated: Release retrieves the default version of ReleaseClient.
// Please explicitly pick a version.
func (c *Clientset) Release() releasev1alpha1.ReleaseV1alpha1Interface {
	if c == nil {
		return nil
	}
	return c.ReleaseV1alpha1Client
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.ConfigV1alpha1Client, err = configv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.ReleaseV1alpha1Client, err = releasev1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.Clientset, err = kubernetes.NewForConfig(&configShallowCopy)
	if err != nil {
		glog.Errorf("failed to create the client-go Clientset: %v", err)
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.ConfigV1alpha1Client = configv1alpha1.NewForConfigOrDie(c)
	cs.ReleaseV1alpha1Client = releasev1alpha1.NewForConfigOrDie(c)

	cs.Clientset = kubernetes.NewForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.ConfigV1alpha1Client = configv1alpha1.New(c)
	cs.ReleaseV1alpha1Client = releasev1alpha1.New(c)

	cs.Clientset = kubernetes.New(c)
	return &cs
}
