/*
Copyright 2020 caicloud authors. All rights reserved.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ReleaseHistoryLister helps list ReleaseHistories.
type ReleaseHistoryLister interface {
	// List lists all ReleaseHistories in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ReleaseHistory, err error)
	// ReleaseHistories returns an object that can list and get ReleaseHistories.
	ReleaseHistories(namespace string) ReleaseHistoryNamespaceLister
	ReleaseHistoryListerExpansion
}

// releaseHistoryLister implements the ReleaseHistoryLister interface.
type releaseHistoryLister struct {
	indexer cache.Indexer
}

// NewReleaseHistoryLister returns a new ReleaseHistoryLister.
func NewReleaseHistoryLister(indexer cache.Indexer) ReleaseHistoryLister {
	return &releaseHistoryLister{indexer: indexer}
}

// List lists all ReleaseHistories in the indexer.
func (s *releaseHistoryLister) List(selector labels.Selector) (ret []*v1alpha1.ReleaseHistory, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ReleaseHistory))
	})
	return ret, err
}

// ReleaseHistories returns an object that can list and get ReleaseHistories.
func (s *releaseHistoryLister) ReleaseHistories(namespace string) ReleaseHistoryNamespaceLister {
	return releaseHistoryNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ReleaseHistoryNamespaceLister helps list and get ReleaseHistories.
type ReleaseHistoryNamespaceLister interface {
	// List lists all ReleaseHistories in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.ReleaseHistory, err error)
	// Get retrieves the ReleaseHistory from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.ReleaseHistory, error)
	ReleaseHistoryNamespaceListerExpansion
}

// releaseHistoryNamespaceLister implements the ReleaseHistoryNamespaceLister
// interface.
type releaseHistoryNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ReleaseHistories in the indexer for a given namespace.
func (s releaseHistoryNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ReleaseHistory, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ReleaseHistory))
	})
	return ret, err
}

// Get retrieves the ReleaseHistory from the indexer for a given namespace and name.
func (s releaseHistoryNamespaceLister) Get(name string) (*v1alpha1.ReleaseHistory, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("releasehistory"), name)
	}
	return obj.(*v1alpha1.ReleaseHistory), nil
}
