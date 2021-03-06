/*
Copyright 2019 caicloud authors. All rights reserved.
*/

// Code generated by listerfactory-gen. DO NOT EDIT.

package v1beta2

import (
	internalinterfaces "github.com/caicloud/clientset/listerfactory/internalinterfaces"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubernetes "k8s.io/client-go/kubernetes"
	v1beta2 "k8s.io/client-go/listers/apps/v1beta2"
)

var _ v1beta2.ControllerRevisionLister = &controllerRevisionLister{}

var _ v1beta2.ControllerRevisionNamespaceLister = &controllerRevisionNamespaceLister{}

// controllerRevisionLister implements the ControllerRevisionLister interface.
type controllerRevisionLister struct {
	client           kubernetes.Interface
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewControllerRevisionLister returns a new ControllerRevisionLister.
func NewControllerRevisionLister(client kubernetes.Interface) v1beta2.ControllerRevisionLister {
	return NewFilteredControllerRevisionLister(client, nil)
}

func NewFilteredControllerRevisionLister(client kubernetes.Interface, tweakListOptions internalinterfaces.TweakListOptionsFunc) v1beta2.ControllerRevisionLister {
	return &controllerRevisionLister{
		client:           client,
		tweakListOptions: tweakListOptions,
	}
}

// List lists all ControllerRevisions in the indexer.
func (s *controllerRevisionLister) List(selector labels.Selector) (ret []*appsv1beta2.ControllerRevision, err error) {
	listopt := v1.ListOptions{
		LabelSelector: selector.String(),
	}
	if s.tweakListOptions != nil {
		s.tweakListOptions(&listopt)
	}
	list, err := s.client.AppsV1beta2().ControllerRevisions(v1.NamespaceAll).List(listopt)
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		ret = append(ret, &list.Items[i])
	}
	return ret, nil
}

// ControllerRevisions returns an object that can list and get ControllerRevisions.
func (s *controllerRevisionLister) ControllerRevisions(namespace string) v1beta2.ControllerRevisionNamespaceLister {
	return controllerRevisionNamespaceLister{client: s.client, tweakListOptions: s.tweakListOptions, namespace: namespace}
}

// controllerRevisionNamespaceLister implements the ControllerRevisionNamespaceLister
// interface.
type controllerRevisionNamespaceLister struct {
	client           kubernetes.Interface
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// List lists all ControllerRevisions in the indexer for a given namespace.
func (s controllerRevisionNamespaceLister) List(selector labels.Selector) (ret []*appsv1beta2.ControllerRevision, err error) {
	listopt := v1.ListOptions{
		LabelSelector: selector.String(),
	}
	if s.tweakListOptions != nil {
		s.tweakListOptions(&listopt)
	}
	list, err := s.client.AppsV1beta2().ControllerRevisions(s.namespace).List(listopt)
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		ret = append(ret, &list.Items[i])
	}
	return ret, nil
}

// Get retrieves the ControllerRevision from the indexer for a given namespace and name.
func (s controllerRevisionNamespaceLister) Get(name string) (*appsv1beta2.ControllerRevision, error) {
	return s.client.AppsV1beta2().ControllerRevisions(s.namespace).Get(name, v1.GetOptions{})
}
