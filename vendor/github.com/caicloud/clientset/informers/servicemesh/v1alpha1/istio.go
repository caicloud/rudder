/*
Copyright 2019 caicloud authors. All rights reserved.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	kubernetes "github.com/caicloud/clientset/kubernetes"
	v1alpha1 "github.com/caicloud/clientset/listers/servicemesh/v1alpha1"
	servicemeshv1alpha1 "github.com/caicloud/clientset/pkg/apis/servicemesh/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	clientgokubernetes "k8s.io/client-go/kubernetes"
	cache "k8s.io/client-go/tools/cache"
)

// IstioInformer provides access to a shared informer and lister for
// Istios.
type IstioInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.IstioLister
}

type istioInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewIstioInformer constructs a new informer for Istio type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewIstioInformer(client kubernetes.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredIstioInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredIstioInformer constructs a new informer for Istio type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredIstioInformer(client kubernetes.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServicemeshV1alpha1().Istios().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServicemeshV1alpha1().Istios().Watch(options)
			},
		},
		&servicemeshv1alpha1.Istio{},
		resyncPeriod,
		indexers,
	)
}

func (f *istioInformer) defaultInformer(client clientgokubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredIstioInformer(client.(kubernetes.Interface), resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *istioInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&servicemeshv1alpha1.Istio{}, f.defaultInformer)
}

func (f *istioInformer) Lister() v1alpha1.IstioLister {
	return v1alpha1.NewIstioLister(f.Informer().GetIndexer())
}