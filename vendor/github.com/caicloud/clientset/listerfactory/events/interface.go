/*
Copyright 2019 caicloud authors. All rights reserved.
*/

// Code generated by listerfactory-gen. DO NOT EDIT.

package events

import (
	v1beta1 "github.com/caicloud/clientset/listerfactory/events/v1beta1"
	internalinterfaces "github.com/caicloud/clientset/listerfactory/internalinterfaces"
	informers "k8s.io/client-go/informers"
	kubernetes "k8s.io/client-go/kubernetes"
)

// Interface provides access to each of this group's versions.
type Interface interface {
	// V1beta1 provides access to listers for resources in V1beta1.
	V1beta1() v1beta1.Interface
}

type group struct {
	client           kubernetes.Interface
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

type informerGroup struct {
	factory informers.SharedInformerFactory
}

// New returns a new Interface.
func New(client kubernetes.Interface, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &group{client: client, tweakListOptions: tweakListOptions}
}

// NewFrom returns a new Interface
func NewFrom(factory informers.SharedInformerFactory) Interface {
	return &informerGroup{factory: factory}
}

// V1beta1 returns a new v1beta1.Interface.
func (g *group) V1beta1() v1beta1.Interface {
	return v1beta1.New(g.client, g.tweakListOptions)
}

func (g *informerGroup) V1beta1() v1beta1.Interface {
	return v1beta1.NewFrom(g.factory)
}
