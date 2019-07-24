/*
Copyright 2019 caicloud authors. All rights reserved.
*/

// Code generated by listerfactory-gen. DO NOT EDIT.

package v1beta1

import (
	internalinterfaces "github.com/caicloud/clientset/listerfactory/internalinterfaces"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubernetes "k8s.io/client-go/kubernetes"
	v1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
)

var _ v1beta1.ReplicaSetLister = &replicaSetLister{}

var _ v1beta1.ReplicaSetNamespaceLister = &replicaSetNamespaceLister{}

// replicaSetLister implements the ReplicaSetLister interface.
type replicaSetLister struct {
	client           kubernetes.Interface
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewReplicaSetLister returns a new ReplicaSetLister.
func NewReplicaSetLister(client kubernetes.Interface) v1beta1.ReplicaSetLister {
	return NewFilteredReplicaSetLister(client, nil)
}

func NewFilteredReplicaSetLister(client kubernetes.Interface, tweakListOptions internalinterfaces.TweakListOptionsFunc) v1beta1.ReplicaSetLister {
	return &replicaSetLister{
		client:           client,
		tweakListOptions: tweakListOptions,
	}
}

// List lists all ReplicaSets in the indexer.
func (s *replicaSetLister) List(selector labels.Selector) (ret []*extensionsv1beta1.ReplicaSet, err error) {
	listopt := v1.ListOptions{
		LabelSelector: selector.String(),
	}
	if s.tweakListOptions != nil {
		s.tweakListOptions(&listopt)
	}
	list, err := s.client.ExtensionsV1beta1().ReplicaSets(v1.NamespaceAll).List(listopt)
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		ret = append(ret, &list.Items[i])
	}
	return ret, nil
}

func (s *replicaSetLister) GetPodReplicaSets(*corev1.Pod) ([]*extensionsv1beta1.ReplicaSet, error) {
	return nil, nil
}

// ReplicaSets returns an object that can list and get ReplicaSets.
func (s *replicaSetLister) ReplicaSets(namespace string) v1beta1.ReplicaSetNamespaceLister {
	return replicaSetNamespaceLister{client: s.client, tweakListOptions: s.tweakListOptions, namespace: namespace}
}

// replicaSetNamespaceLister implements the ReplicaSetNamespaceLister
// interface.
type replicaSetNamespaceLister struct {
	client           kubernetes.Interface
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// List lists all ReplicaSets in the indexer for a given namespace.
func (s replicaSetNamespaceLister) List(selector labels.Selector) (ret []*extensionsv1beta1.ReplicaSet, err error) {
	listopt := v1.ListOptions{
		LabelSelector: selector.String(),
	}
	if s.tweakListOptions != nil {
		s.tweakListOptions(&listopt)
	}
	list, err := s.client.ExtensionsV1beta1().ReplicaSets(s.namespace).List(listopt)
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		ret = append(ret, &list.Items[i])
	}
	return ret, nil
}

// Get retrieves the ReplicaSet from the indexer for a given namespace and name.
func (s replicaSetNamespaceLister) Get(name string) (*extensionsv1beta1.ReplicaSet, error) {
	return s.client.ExtensionsV1beta1().ReplicaSets(s.namespace).Get(name, v1.GetOptions{})
}