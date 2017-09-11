package kube

import (
	"bytes"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

type Codec interface {
	ResourceToObject(resource string) (runtime.Object, error)
	ResourcesToObjects(resources []string) ([]runtime.Object, error)
	ObjectToResource(obj runtime.Object) (string, error)
	ObjectsToResources(objs []runtime.Object) ([]string, error)
	AccessorForObject(obj runtime.Object) (metav1.Object, error)
	AccessorForResource(resource string) (runtime.Object, metav1.Object, error)
	AccessorsForObjects(objs []runtime.Object) ([]metav1.Object, error)
	AccessorsForResources(resources []string) ([]runtime.Object, []metav1.Object, error)
}

type yamlCodec struct {
	serializer *json.Serializer
}

func NewYAMLCodec(creator runtime.ObjectCreater, typer runtime.ObjectTyper) Codec {
	return &yamlCodec{
		serializer: json.NewYAMLSerializer(json.DefaultMetaFactory, creator, typer),
	}
}

func (c *yamlCodec) ResourceToObject(resource string) (runtime.Object, error) {
	obj, _, err := c.serializer.Decode([]byte(resource), nil, nil)
	return obj, err
}

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

func (c *yamlCodec) ObjectToResource(obj runtime.Object) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := c.serializer.Encode(obj, buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

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

func (c *yamlCodec) AccessorForObject(obj runtime.Object) (metav1.Object, error) {
	accessor, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		return nil, fmt.Errorf("unrecognized object")
	}
	return accessor.GetObjectMeta(), nil
}

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
