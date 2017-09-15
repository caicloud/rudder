package kube

import (
	"fmt"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// Client implements CRUD methods for a group resources.
type Client interface {
	// Get gets the current object by resources.
	Get(namespace string, resources []string, options GetOptions) ([]runtime.Object, error)
	// Create creates all these resources.
	Create(namespace string, resources []string, options CreateOptions) error
	// Update updates all resources.
	Update(namespace string, originalResources, targetResources []string, options UpdateOptions) error
	// Delete deletes all resources.
	Delete(namespace string, resources []string, options DeleteOptions) error
}

// NewClient creates a client for resources.
func NewClient(pool ClientPool, codec Codec) (Client, error) {
	client := &client{
		pool:  pool,
		codec: codec,
	}
	return client, nil
}

type client struct {
	pool  ClientPool
	codec Codec
}

// Get gets the current object by resources.
func (c *client) Get(namespace string, resources []string, options GetOptions) ([]runtime.Object, error) {
	objs, err := c.codec.ResourcesToObjects(resources)
	if err != nil {
		return nil, err
	}
	result := make([]runtime.Object, 0, len(objs))
	for _, obj := range objs {
		accessor, err := c.codec.AccessorForObject(obj)
		if err != nil {
			return nil, err
		}
		client, err := c.pool.ClientFor(obj.GetObjectKind().GroupVersionKind(), namespace)
		if err != nil {
			return nil, err
		}
		object, err := client.Get(accessor.GetName(), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) && options.IgnoreNonexistence {
				// Ignore inexistent resource
				continue
			}
			return nil, err
		}
		if c.own(options.OwnerReferences, object) {
			result = append(result, object)
		}
	}
	return result, nil
}

// Create creates all these resources.
func (c *client) Create(namespace string, resources []string, options CreateOptions) error {
	objs, err := c.objectsByOrder(resources, InstallOrder)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		client, err := c.pool.ClientFor(obj.GetObjectKind().GroupVersionKind(), namespace)
		if err != nil {
			return err
		}
		accessor, err := c.codec.AccessorForObject(obj)
		if err != nil {
			return err
		}
		// Check whether the object exists.
		existence, err := client.Get(accessor.GetName(), metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		if err == nil && c.own(options.OwnerReferences, existence) {
			// Take over it if exists.
			// TODO(kdada): Ensure the two objects are same.
			continue
		}
		if options.OwnerReferences != nil {
			accessor.SetOwnerReferences(append(accessor.GetOwnerReferences(), options.OwnerReferences...))
		}
		_, err = client.Create(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

// Update updates all resources. It checks all resources and classifies into three types.
func (c *client) Update(namespace string, originalResources, targetResources []string, options UpdateOptions) error {
	originalObjects, originalAccessors, err := c.codec.AccessorsForResources(originalResources)
	if err != nil {
		return err
	}
	targetObjects, targetAccessors, err := c.codec.AccessorsForResources(targetResources)
	if err != nil {
		return err
	}
	keyFor := func(obj runtime.Object, accessor metav1.Object) string {
		gvk := obj.GetObjectKind().GroupVersionKind()
		return fmt.Sprintf("%s/%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind, accessor.GetName())
	}
	originalInfos := make(map[string]int)
	for i, accessor := range originalAccessors {
		originalInfos[keyFor(originalObjects[i], accessor)] = i
	}
	toCreate := []string{}
	toUpdate := []resources{}
	toDelete := []string{}
	for i, accessor := range targetAccessors {
		key := keyFor(targetObjects[i], accessor)
		index, ok := originalInfos[key]
		if !ok {
			toCreate = append(toCreate, targetResources[i])
		} else {
			toUpdate = append(toUpdate, resources{originalResources[index], targetResources[i]})
			delete(originalInfos, key)
		}
	}
	for _, index := range originalInfos {
		toDelete = append(toDelete, originalResources[index])
	}
	// Create
	if len(toCreate) > 0 {
		opts := CreateOptions{options.OwnerReferences}
		if err = c.Create(namespace, toCreate, opts); err != nil {
			return err
		}
	}
	// Update
	if len(toUpdate) > 0 {
		if err = c.update(namespace, toUpdate, options); err != nil {
			return err
		}
	}
	// Delete
	if len(toDelete) > 0 {
		opts := DeleteOptions{options.OwnerReferences}
		if err = c.Delete(namespace, toDelete, opts); err != nil {
			return err
		}
	}
	return nil
}

// resources is a binding for an update.
type resources struct {
	origin string
	target string
}

// update updates a list of resource updates.
func (c *client) update(namespace string, updates []resources, options UpdateOptions) error {
	// TODO(kdada): sort updates by InstallOrder
	for _, u := range updates {
		origin, err := yaml.YAMLToJSON([]byte(u.origin))
		if err != nil {
			return err
		}
		target, err := yaml.YAMLToJSON([]byte(u.target))
		if err != nil {
			return err
		}
		obj, accessor, err := c.codec.AccessorForResource(u.origin)
		if err != nil {
			return err
		}
		client, err := c.pool.ClientFor(obj.GetObjectKind().GroupVersionKind(), namespace)
		if err != nil {
			return err
		}
		current, err := client.Get(accessor.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if !c.own(options.OwnerReferences, current) {
			return fmt.Errorf("attempt to update a non-affiliated object: %s/%s", accessor.GetNamespace(), accessor.GetName())
		}
		// TODO(kdada): Replace with merge patch when obj is TPR or CRD.
		patch, err := strategicpatch.CreateTwoWayMergePatch(origin, target, obj)
		if err != nil {
			return err
		}
		_, err = client.Patch(accessor.GetName(), types.StrategicMergePatchType, patch)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes all resources.
func (c *client) Delete(namespace string, resources []string, options DeleteOptions) error {
	objs, err := c.objectsByOrder(resources, UninstallOrder)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		accessor, err := c.codec.AccessorForObject(obj)
		if err != nil {
			return err
		}
		client, err := c.pool.ClientFor(obj.GetObjectKind().GroupVersionKind(), namespace)
		if err != nil {
			return err
		}
		obj, err := client.Get(accessor.GetName(), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Object is not found. Don't need delete
				continue
			}
			return err
		}
		if c.own(options.OwnerReferences, obj) {
			deletePolicy := metav1.DeletePropagationBackground
			err = client.Delete(accessor.GetName(), &metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

// objectsByOrder converts resources and order by specified sort order.
func (c *client) objectsByOrder(resources []string, order SortOrder) ([]runtime.Object, error) {
	objs, err := c.codec.ResourcesToObjects(resources)
	if err != nil {
		return nil, err
	}
	order.Sort(objs)
	return objs, nil
}

// own checks whether obj have same reference. It always return
// true when ref is nil.
func (c *client) own(refs []metav1.OwnerReference, obj runtime.Object) bool {
	accessor, err := c.codec.AccessorForObject(obj)
	if err != nil {
		return false
	}
	if refs == nil {
		return true
	}
	references := accessor.GetOwnerReferences()
	for _, ref := range refs {
		found := false
		for _, r := range references {
			if ref.APIVersion == r.APIVersion &&
				ref.Kind == r.Kind &&
				ref.Name == r.Name &&
				ref.UID == r.UID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
