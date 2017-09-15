package store

import (
	"fmt"
	"sync"

	"github.com/caicloud/release-controller/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// IntegrationStore is a store of generic informers.
type IntegrationStore interface {
	// InformerFor gets a generic informer for specified GroupVersionKind.
	InformerFor(gvk schema.GroupVersionKind) (informers.GenericInformer, error)
}

type integrationStore struct {
	lock      sync.RWMutex
	resources kube.APIResources
	factory   informers.SharedInformerFactory
	informers map[schema.GroupVersionKind]informers.GenericInformer
	stopCh    <-chan struct{}
}

// NewIntegrationStore creates a IntegrationStore.
func NewIntegrationStore(resources kube.APIResources, factory informers.SharedInformerFactory, stopCh <-chan struct{}) IntegrationStore {
	return &integrationStore{
		resources: resources,
		factory:   factory,
		informers: make(map[schema.GroupVersionKind]informers.GenericInformer),
		stopCh:    stopCh,
	}
}

// InformerFor gets a generic informer for specified GroupVersionKind.
// The method will block until target informer is synced.
func (is *integrationStore) InformerFor(gvk schema.GroupVersionKind) (informers.GenericInformer, error) {
	is.lock.Lock()
	defer is.lock.Unlock()
	gi, ok := is.informers[gvk]
	if !ok {
		resource, err := is.resources.ResourceFor(gvk)
		if err != nil {
			return nil, err
		}
		gi, err = is.factory.ForResource(resource.GroupVersionResource())
		if err != nil {
			return nil, err
		}
		is.informers[gvk] = gi
		is.factory.Start(is.stopCh)
		if !cache.WaitForCacheSync(is.stopCh, gi.Informer().HasSynced) {
			return nil, fmt.Errorf("can't sync informer for: %s", gvk)
		}
	}
	return gi, nil
}
