package store

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/caicloud/release-controller/pkg/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// DefaultTTL is for objects in cache layer.
// If underlying store has no changes, cached objects have 10s to live.
const DefaultTTL = time.Second * 10

// IntegrationStore is a store of generic informers.
type IntegrationStore interface {
	// InformerFor gets a generic informer for specified GroupVersionKind.
	InformerFor(gvk schema.GroupVersionKind) (informers.GenericInformer, error)
	// LayerFor get a layer for concrete kind.
	LayerFor(gvk schema.GroupVersionKind) (kube.CacheLayer, error)
}

type integrationStore struct {
	lock      sync.RWMutex
	resources kube.APIResources
	factory   informers.SharedInformerFactory
	informers map[schema.GroupVersionKind]*cacheInformer
	ttl       time.Duration
	stopCh    <-chan struct{}
}

// NewIntegrationStoreWithTTL creates a IntegrationStore with a default cache TTL.
func NewIntegrationStoreWithTTL(resources kube.APIResources, factory informers.SharedInformerFactory, ttl time.Duration, stopCh <-chan struct{}) IntegrationStore {
	return &integrationStore{
		resources: resources,
		factory:   factory,
		informers: make(map[schema.GroupVersionKind]*cacheInformer),
		ttl:       ttl,
		stopCh:    stopCh,
	}
}

// NewIntegrationStore creates a IntegrationStore.
func NewIntegrationStore(resources kube.APIResources, factory informers.SharedInformerFactory, stopCh <-chan struct{}) IntegrationStore {
	return NewIntegrationStoreWithTTL(resources, factory, DefaultTTL, stopCh)
}

func (is *integrationStore) informerFor(gvk schema.GroupVersionKind) (*cacheInformer, error) {
	is.lock.Lock()
	defer is.lock.Unlock()
	informer, ok := is.informers[gvk]
	if !ok {
		resource, err := is.resources.ResourceFor(gvk)
		if err != nil {
			return nil, err
		}
		gi, err := is.factory.ForResource(resource.GroupVersionResource())
		if err != nil {
			return nil, err
		}
		informer = newCacheInformer(gi, resource.GroupVersionResource(), is.ttl)
		is.informers[gvk] = informer
		is.factory.Start(is.stopCh)
		if !cache.WaitForCacheSync(is.stopCh, gi.Informer().HasSynced) {
			return nil, fmt.Errorf("can't sync informer for: %s", gvk)
		}
	}
	return informer, nil
}

// InformerFor gets a generic informer for specified GroupVersionKind.
// The method will block until target informer is synced.
func (is *integrationStore) InformerFor(gvk schema.GroupVersionKind) (informers.GenericInformer, error) {
	return is.informerFor(gvk)
}

// LayerFor get a layer for concrete kind. Before get a layer,
func (is *integrationStore) LayerFor(gvk schema.GroupVersionKind) (kube.CacheLayer, error) {
	informer, err := is.informerFor(gvk)
	if err != nil {
		return nil, err
	}
	return informer.layer, nil
}

type cacheInformer struct {
	informer informers.GenericInformer
	layer    *cacheLayer
}

func newCacheInformer(informer informers.GenericInformer, gvr schema.GroupVersionResource, ttl time.Duration) *cacheInformer {
	return &cacheInformer{
		informer,
		newCacheLayer(informer.Lister(), gvr.GroupResource(), ttl),
	}
}

func (i *cacheInformer) Informer() cache.SharedIndexInformer {
	return i.informer.Informer()
}

func (i *cacheInformer) Lister() cache.GenericLister {
	return i.layer
}

type cacheObject struct {
	namespace         string
	name              string
	uuid              types.UID
	creationTimestamp metav1.Time
	resourceVersion   int
	existing          bool
	object            runtime.Object
	deadline          time.Time
}

func (c *cacheObject) GetObjectKind() schema.ObjectKind {
	return c.object.GetObjectKind()
}

func (c *cacheObject) Expired() bool {
	return time.Now().After(c.deadline)
}

// cacheLayer caches objects by a temporary store.
type cacheLayer struct {
	lock        sync.RWMutex
	gr          schema.GroupResource
	indexer     cache.Indexer
	indexLister cache.GenericLister
	lister      cache.GenericLister
	ttl         time.Duration
}

func newCacheLayer(lister cache.GenericLister, gr schema.GroupResource, ttl time.Duration) *cacheLayer {
	// indexer is for cache object.
	indexer := cache.NewIndexer(func(obj interface{}) (string, error) {
		if co, ok := obj.(*cacheObject); ok {
			if len(co.namespace) > 0 {
				return co.namespace + "/" + co.name, nil
			}
			return co.name, nil
		}
		// It should not come here.
		panic("Invalid cache object")
	}, cache.Indexers{cache.NamespaceIndex: func(obj interface{}) ([]string, error) {
		switch o := obj.(type) {
		case *cacheObject:
			return []string{o.namespace}, nil
		case metav1.ObjectMetaAccessor:
			return []string{o.GetObjectMeta().GetNamespace()}, nil
		}
		// It should not come here.
		panic("Invalid cache object")
	}})
	return &cacheLayer{
		indexer:     indexer,
		indexLister: cache.NewGenericLister(indexer, gr),
		gr:          gr,
		lister:      lister,
		ttl:         ttl,
	}
}

func (c *cacheLayer) get(object runtime.Object) (*cacheObject, bool) {
	accessor, ok := object.(metav1.ObjectMetaAccessor)
	if !ok {
		return nil, false
	}
	meta := accessor.GetObjectMeta()
	var co runtime.Object
	var err error
	if meta.GetNamespace() == "" {
		co, err = c.indexLister.Get(meta.GetName())
	} else {
		co, err = c.indexLister.ByNamespace(meta.GetNamespace()).Get(meta.GetName())
	}
	if err != nil {
		return nil, false
	}
	return co.(*cacheObject), true
}

func (c *cacheLayer) add(obj runtime.Object, existing bool) {
	meta := obj.(metav1.ObjectMetaAccessor).GetObjectMeta()
	version, _ := strconv.Atoi(meta.GetResourceVersion())
	co := &cacheObject{
		namespace:         meta.GetNamespace(),
		name:              meta.GetName(),
		uuid:              meta.GetUID(),
		creationTimestamp: meta.GetCreationTimestamp(),
		resourceVersion:   version,
		existing:          existing,
		object:            obj,
		deadline:          time.Now().Add(c.ttl),
	}
	c.indexer.Add(co)
}

func (c *cacheLayer) remove(obj runtime.Object) {
	if co, ok := c.get(obj); ok {
		c.indexer.Delete(co)
	}
}

// Created records an object is created.
func (c *cacheLayer) Created(obj runtime.Object) {
	c.add(obj, true)
}

// Updated records an object is updated.
func (c *cacheLayer) Updated(obj runtime.Object) {
	c.add(obj, true)
}

// Deleted records an object is deleted.
func (c *cacheLayer) Deleted(obj runtime.Object) {
	c.add(obj, false)
}

// List will return all objects across namespaces
func (c *cacheLayer) List(selector labels.Selector) (ret []runtime.Object, err error) {
	return c.listBySelector(selector, c.lister, c.indexLister)
}

// Get will attempt to retrieve assuming that name==key
func (c *cacheLayer) Get(name string) (runtime.Object, error) {
	return c.getByKey(name, c.lister, c.indexLister)
}

// listBySelector lists obejects by selector. All valid objects in cache lister would be added into results.
func (c *cacheLayer) listBySelector(selector labels.Selector, lister, cacheLister cache.GenericNamespaceLister) ([]runtime.Object, error) {
	currentObjs, err := lister.List(selector)
	if err != nil {
		return currentObjs, err
	}
	cacheObjs, e := cacheLister.List(selector)
	if e != nil {
		return currentObjs, err
	}
	if len(cacheObjs) == 0 {
		return currentObjs, err
	}
	// Get all cached objects.
	cacheObjsMap := map[string]*cacheObject{}
	for _, obj := range cacheObjs {
		co := obj.(*cacheObject)
		cacheObjsMap[keyForNamespaceAndName(co.namespace, co.name)] = co
	}
	count := 0
	for _, obj := range currentObjs {
		meta := obj.(metav1.ObjectMetaAccessor).GetObjectMeta()
		key := keyForNamespaceAndName(meta.GetNamespace(), meta.GetName())
		co, ok := cacheObjsMap[key]
		if ok {
			// Select latest one.
			if result := c.selectObject(obj, co); result != nil {
				currentObjs[count] = result
				count++
			}
			delete(cacheObjsMap, key)
		}
	}
	currentObjs = currentObjs[:count]
	for _, co := range cacheObjsMap {
		if obj := c.selectObject(nil, co); obj != nil {
			currentObjs = append(currentObjs, obj)
		}
	}
	return currentObjs, nil
}

// getByKey gets an object from listers by key.
func (c *cacheLayer) getByKey(key string, lister, cacheLister cache.GenericNamespaceLister) (runtime.Object, error) {
	obj, err := lister.Get(key)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	cacheObj, e := cacheLister.Get(key)
	if e != nil {
		return obj, err
	}
	obj = c.selectObject(obj, cacheObj)
	if obj == nil {
		return nil, errors.NewNotFound(c.gr, key)
	}
	return obj, nil
}

// selectObject select object between current and co. co must be *cacheObject.
func (c *cacheLayer) selectObject(current, cacheObj runtime.Object) runtime.Object {
	if cacheObj == nil {
		return current
	}
	co := cacheObj.(*cacheObject)
	if co.Expired() {
		return current
	}
	objCreationTimestamp := metav1.Time{}
	resourceVersion := 0
	if current != nil {
		meta := current.(metav1.ObjectMetaAccessor).GetObjectMeta()
		objCreationTimestamp = meta.GetCreationTimestamp()
		if meta.GetResourceVersion() != "" {
			resourceVersion, _ = strconv.Atoi(meta.GetResourceVersion())
		}
	}
	if !(objCreationTimestamp.Before(co.creationTimestamp)) || resourceVersion >= co.resourceVersion {
		// object is newer than co.
		c.remove(co.object)
		return current
	}
	if co.existing {
		return co.object
	}
	return nil
}

// ByNamespace will give you a GenericNamespaceLister for one namespace
func (c *cacheLayer) ByNamespace(namespace string) cache.GenericNamespaceLister {
	return &namespacedLister{c, c.indexLister.ByNamespace(namespace), c.lister.ByNamespace(namespace), namespace}
}

type namespacedLister struct {
	layer       *cacheLayer
	indexLister cache.GenericNamespaceLister
	lister      cache.GenericNamespaceLister
	namespace   string
}

// List will return all objects across namespaces
func (l *namespacedLister) List(selector labels.Selector) (ret []runtime.Object, err error) {
	return l.layer.listBySelector(selector, l.lister, l.indexLister)
}

// Get will attempt to retrieve assuming that name==key
func (l *namespacedLister) Get(name string) (runtime.Object, error) {
	return l.layer.getByKey(name, l.lister, l.indexLister)
}

func keyForNamespaceAndName(namespace, name string) string {
	return namespace + "/" + name
}
