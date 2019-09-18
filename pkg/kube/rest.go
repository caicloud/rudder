package kube

import (
	"fmt"
	"sync"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

// ClientPool is a pool of clients.
type ClientPool interface {
	// ClientFor gets a client for specified kind of an object. If APIResource of the kind is
	// non-namespaced, ignore the namespace. If the resource is namespaced and namespace is empty,
	// It uses 'Default' as the namespace.
	ClientFor(gvk schema.GroupVersionKind, namespace string) (*ResourceClient, error)
}

// clientPool describes a client pool for object scheme.
type clientPool struct {
	sync.Mutex
	scheme    *runtime.Scheme
	codec     runtime.ParameterCodec
	factory   serializer.CodecFactory
	config    *rest.Config
	resources APIResources
	clients   map[schema.GroupVersionKind]*ResourceClient
}

// NewClientPool create a client pool for objects. The config only contains basic configurations
// to connect target api server.
func NewClientPool(scheme *runtime.Scheme, config *rest.Config, resources APIResources) (ClientPool, error) {
	pool := &clientPool{
		scheme:    scheme,
		codec:     runtime.NewParameterCodec(scheme),
		factory:   serializer.NewCodecFactory(scheme),
		config:    config,
		resources: resources,
		clients:   make(map[schema.GroupVersionKind]*ResourceClient),
	}
	return pool, nil
}

// ClientFor gets a client for specified kind of an object. If APIResource of the kind is
// non-namespaced, ignore the namespace. If the resource is namespaced and namespace is empty,
// It uses 'Default' as the namespace.
func (cp *clientPool) ClientFor(gvk schema.GroupVersionKind, namespace string) (*ResourceClient, error) {
	resource, err := cp.resources.ResourceFor(gvk)
	if err != nil {
		return nil, err
	}
	if !resource.Namespaced {
		namespace = ""
	} else if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	cp.Lock()
	defer cp.Unlock()
	client, ok := cp.clients[gvk]
	if !ok {
		conf := *cp.config
		if gvk.Group == core.SchemeGroupVersion.Group {
			conf.APIPath = "/api"
		} else {
			conf.APIPath = "/apis"
		}
		gv := gvk.GroupVersion()
		conf.GroupVersion = &gv
		conf.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: cp.factory}
		cl, err := rest.RESTClientFor(&conf)
		if err != nil {
			return nil, err
		}
		client = &ResourceClient{
			cl:             cl,
			resource:       &resource.APIResource,
			parameterCodec: cp.codec,
		}
		cp.clients[gvk] = client
	}
	if namespace != "" {
		// Copy client for namespace
		copy := *client
		copy.ns = namespace
		return &copy, nil
	}
	return client, nil

}

// ResourceClient contains a bundle of methods to manipulate certain kind of object.
type ResourceClient struct {
	cl             *rest.RESTClient
	resource       *metav1.APIResource
	ns             string
	parameterCodec runtime.ParameterCodec
}

// List returns a list of objects for this resource.
func (rc *ResourceClient) List(opts metav1.ListOptions) (runtime.Object, error) {
	return rc.cl.Get().
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		VersionedParams(&opts, rc.parameterCodec).
		Do().
		Get()
}

// Get gets the resource with the specified name.
func (rc *ResourceClient) Get(name string, opts metav1.GetOptions) (runtime.Object, error) {
	return rc.cl.Get().
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		VersionedParams(&opts, rc.parameterCodec).
		Name(name).
		Do().
		Get()
}

// Delete deletes the resource with the specified name.
func (rc *ResourceClient) Delete(name string, opts *metav1.DeleteOptions) error {
	return rc.cl.Delete().
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		Name(name).
		Body(opts).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (rc *ResourceClient) DeleteCollection(deleteOptions *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return rc.cl.Delete().
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		VersionedParams(&listOptions, rc.parameterCodec).
		Body(deleteOptions).
		Do().
		Error()
}

// Create creates the provided resource.
func (rc *ResourceClient) Create(obj runtime.Object) (runtime.Object, error) {
	return rc.cl.Post().
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		Body(obj).
		Do().
		Get()
}

// Update updates the provided resource.
func (rc *ResourceClient) Update(obj runtime.Object) (runtime.Object, error) {
	accessor, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		return obj, fmt.Errorf("unrecognized object")
	}
	name := accessor.GetObjectMeta().GetName()

	if len(name) == 0 {
		return obj, fmt.Errorf("object missing name")
	}
	return rc.cl.Put().
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		Name(name).
		Body(obj).
		Do().
		Get()
}

// Watch returns a watch.Interface that watches the resource.
func (rc *ResourceClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return rc.cl.Get().
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		VersionedParams(&opts, rc.parameterCodec).
		Watch()
}

// Patch patches the provided resource.
func (rc *ResourceClient) Patch(name string, pt types.PatchType, data []byte) (runtime.Object, error) {
	return rc.cl.Patch(pt).
		NamespaceIfScoped(rc.ns, rc.resource.Namespaced).
		Resource(rc.resource.Name).
		Name(name).
		Body(data).
		Do().
		Get()
}
