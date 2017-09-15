package kube

import (
	"bytes"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// Codec converts between resources and objects.
type Codec interface {
	// ResourceToObject converts a resource to a object.
	ResourceToObject(resource string) (runtime.Object, error)
	// ResourcesToObjects converts resources to a objects.
	ResourcesToObjects(resources []string) ([]runtime.Object, error)
	// ObjectToResource converts object to resource.
	ObjectToResource(obj runtime.Object) (string, error)
	// ObjectsToResources converts objects to resources.
	ObjectsToResources(objs []runtime.Object) ([]string, error)
	// AccessorForObject gets accessor from object.
	AccessorForObject(obj runtime.Object) (metav1.Object, error)
	// AccessorForResource gets accessor from resource.
	AccessorForResource(resource string) (runtime.Object, metav1.Object, error)
	// AccessorsForObjects gets accessors from objects.
	AccessorsForObjects(objs []runtime.Object) ([]metav1.Object, error)
	// AccessorsForResources gets accessors from resources.
	AccessorsForResources(resources []string) ([]runtime.Object, []metav1.Object, error)
}

// yamlCodec is a yaml codec. It converts between yaml resources and objects.
type yamlCodec struct {
	serializer *json.Serializer
}

// NewYAMLCodec creates a codec with yaml serializer.
func NewYAMLCodec(creator runtime.ObjectCreater, typer runtime.ObjectTyper) Codec {
	return &yamlCodec{
		serializer: json.NewYAMLSerializer(json.DefaultMetaFactory, creator, typer),
	}
}

// ResourceToObject converts a resource to a object.
func (c *yamlCodec) ResourceToObject(resource string) (runtime.Object, error) {
	obj, _, err := c.serializer.Decode([]byte(resource), nil, nil)
	return obj, err
}

// ResourcesToObjects converts resources to a objects.
func (c *yamlCodec) ResourcesToObjects(resources []string) ([]runtime.Object, error) {
	objects := make([]runtime.Object, len(resources))
	for i, res := range resources {
		obj, err := c.ResourceToObject(res)
		if err != nil {
			return nil, err
		}
		objects[i] = obj
	}
	return objects, nil
}

// ObjectToResource converts object to resource.
func (c *yamlCodec) ObjectToResource(obj runtime.Object) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := c.serializer.Encode(obj, buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ObjectsToResources converts objects to resources.
func (c *yamlCodec) ObjectsToResources(objs []runtime.Object) ([]string, error) {
	resources := make([]string, len(objs))
	for i, obj := range objs {
		resource, err := c.ObjectToResource(obj)
		if err != nil {
			return nil, err
		}
		resources[i] = resource
	}
	return resources, nil
}

// AccessorForObject gets accessor from object.
func (c *yamlCodec) AccessorForObject(obj runtime.Object) (metav1.Object, error) {
	accessor, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		return nil, fmt.Errorf("unrecognized object")
	}
	return accessor.GetObjectMeta(), nil
}

// AccessorForResource gets accessor from resource.
func (c *yamlCodec) AccessorForResource(resource string) (runtime.Object, metav1.Object, error) {
	obj, err := c.ResourceToObject(resource)
	if err != nil {
		return nil, nil, err
	}
	accessor, err := c.AccessorForObject(obj)
	if err != nil {
		return nil, nil, err
	}
	return obj, accessor, nil
}

// AccessorsForObjects gets accessors from objects.
func (c *yamlCodec) AccessorsForObjects(objs []runtime.Object) ([]metav1.Object, error) {
	accessors := make([]metav1.Object, len(objs))
	for i, obj := range objs {
		accessor, err := c.AccessorForObject(obj)
		if err != nil {
			return nil, err
		}
		accessors[i] = accessor
	}
	return accessors, nil
}

// AccessorsForResources gets accessors from resources.
func (c *yamlCodec) AccessorsForResources(resources []string) ([]runtime.Object, []metav1.Object, error) {
	objects := make([]runtime.Object, len(resources))
	accessors := make([]metav1.Object, len(resources))
	for i, res := range resources {
		obj, accessor, err := c.AccessorForResource(res)
		if err != nil {
			return nil, nil, err
		}
		objects[i] = obj
		accessors[i] = accessor
	}
	return objects, accessors, nil
}
