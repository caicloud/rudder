package kube

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/imdario/mergo"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	k8spodutil "k8s.io/kubernetes/pkg/api/pod"
	k8sbatchv1 "k8s.io/kubernetes/pkg/apis/batch/v1"
	k8score "k8s.io/kubernetes/pkg/apis/core"

	"github.com/caicloud/rudder/pkg/kube/apply"
)

// CacheLayers Contains layers for all kinds.
type CacheLayers interface {
	// LayerFor get a layer for concrete kind.
	LayerFor(gvk schema.GroupVersionKind) (CacheLayer, error)
}

// CacheLayer is a resource cache store.
type CacheLayer interface {
	// Created records an object is created.
	Created(obj runtime.Object)
	// Updated records an object is updated.
	Updated(obj runtime.Object)
	// Deleted records an object is deleted.
	Deleted(obj runtime.Object)
	// GenericLister is a cached lister to get objects.
	cache.GenericLister
}

// Client implements CRUD methods for a group resources.
type Client interface {
	// Get gets the current object by resources.
	Get(namespace string, resources []string, options GetOptions) ([]runtime.Object, error)
	// Apply creates/updates all these resources.
	Apply(namespace string, resources []string, options ApplyOptions) error
	// Create creates all these resources.
	Create(namespace string, resources []string, options CreateOptions) error
	// Update updates all resources.
	Update(namespace string, originalResources, targetResources []string, options UpdateOptions) error
	// Delete deletes all resources.
	Delete(namespace string, resources []string, options DeleteOptions) error
}

// NewClientWithCacheLayer creates a client for resources with cache layers. If layers is not
// nil, the client gets/lists objects by layers preferentially.
func NewClientWithCacheLayer(pool ClientPool, codec Codec, layers CacheLayers) (Client, error) {
	client := &client{
		pool:   pool,
		codec:  codec,
		layers: layers,
	}
	return client, nil
}

// NewClient creates a client for resources.
func NewClient(pool ClientPool, codec Codec) (Client, error) {
	return NewClientWithCacheLayer(pool, codec, nil)
}

type client struct {
	pool   ClientPool
	codec  Codec
	layers CacheLayers
}

func (c *client) getObject(gvk schema.GroupVersionKind, namespace, name string) (runtime.Object, error) {
	if c.layers != nil {
		// Get object from cache.
		layer, err := c.layers.LayerFor(gvk)
		if err != nil {
			return nil, err
		}
		return layer.ByNamespace(namespace).Get(name)
	}
	// Get object by client.
	client, err := c.pool.ClientFor(gvk, namespace)
	if err != nil {
		return nil, err
	}
	return client.Get(name, metav1.GetOptions{})
}

// Get gets the current object by resources.
func (c *client) Get(namespace string, resources []string, options GetOptions) ([]runtime.Object, error) {
	objs, accessors, err := c.codec.AccessorsForResources(resources)
	if err != nil {
		return nil, err
	}
	result := make([]runtime.Object, 0, len(objs))
	for i, obj := range objs {
		accessor, err := c.codec.AccessorForObject(obj)
		if err != nil {
			return nil, err
		}
		gvk := obj.GetObjectKind().GroupVersionKind()
		object, err := c.getObject(gvk, namespace, accessor.GetName())
		if err != nil {
			if errors.IsNotFound(err) && options.IgnoreNonexistence {
				// Ignore inexistent resource
				continue
			}
			return nil, err
		}
		if !c.own(options.OwnerReferences, object) {
			// The object is not belong to current owner.
			accessor := accessors[i]
			return nil, fmt.Errorf("%s/%s(%s) exists but not belong to current owner",
				namespace, accessor.GetName(), gvk.Kind)
		}
		result = append(result, object)
	}
	return result, nil
}

// Apply creates/updates all these resources.
func (c *client) Apply(namespace string, resources []string, options ApplyOptions) error {
	objs, err := c.objectsByOrder(resources, InstallOrder)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		gvk := obj.GetObjectKind().GroupVersionKind()
		accessor, err := c.codec.AccessorForObject(obj)
		if err != nil {
			return err
		}
		if options.OwnerReferences != nil &&
			// options.Checker is used to check if the object is belong to current owner.
			// If not, add owner references to obj.
			(options.Checker == nil || !options.Checker(obj)) {
			accessor.SetOwnerReferences(append(accessor.GetOwnerReferences(), options.OwnerReferences...))
		}
		client, err := c.pool.ClientFor(gvk, namespace)
		if err != nil {
			return err
		}
		// Check whether the object exists.
		existence, err := c.getObject(gvk, namespace, accessor.GetName())
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		if err != nil {
			// Create
			result, err := client.Create(obj)
			if err != nil {
				return err
			}
			if c.layers != nil {
				// Record the result into cache.
				layer, err := c.layers.LayerFor(gvk)
				if err != nil {
					return err
				}
				layer.Created(result)
			}
		} else {
			// Update
			if c.own(options.OwnerReferences, existence) ||
				(options.Checker != nil && options.Checker(obj)) {
				// Job Cannot be update, so we must re-create Job
				if gvk.Kind == "Job" {
					err := c.applyJob(client, gvk, obj, existence)
					if err != nil {
						return err
					}
					continue
				}
				// Deployment/StatefulSet ip list decrease
				if gvk.Kind == "Deployment" || gvk.Kind == "StatefulSet" {
					isIPDecreasing, err := judgeIPSpecDecreasing(obj, existence)
					if err != nil {
						return err
					}
					if isIPDecreasing {
						err = c.applyIPSpecDecreasing(client, namespace, obj, existence)
						if err != nil {
							return err
						}
					}
				}
				if err := apply.Apply(gvk, existence, obj); err != nil {
					return err
				}
				result, err := client.Update(obj)
				if err != nil {
					return err
				}
				if c.layers != nil {
					// Record the result into cache.
					layer, err := c.layers.LayerFor(gvk)
					if err != nil {
						return err
					}
					layer.Updated(result)
				}
			} else {
				glog.Errorf("%+v, %v", existence, err)
				// Conflict
				return fmt.Errorf("%s/%s(%s) is not belong to current owner %v",
					namespace, accessor.GetName(),
					gvk.Kind, options.OwnerReferences)
			}
		}
	}
	return nil
}

func (c *client) applyJob(client *ResourceClient, gvk schema.GroupVersionKind, obj, existence runtime.Object) error {
	desiredJob := obj.(*batchv1.Job)
	currentJob := existence.(*batchv1.Job)

	equal, err := jobEqual(desiredJob, currentJob)
	if err != nil {
		return err
	}
	if equal {
		return nil
	}

	// Job Cannot be update, so we must re-create Job
	deletePolicy := metav1.DeletePropagationBackground
	err = client.Delete(desiredJob.Name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if c.layers != nil {
		// Record the result into cache.
		layer, err := c.layers.LayerFor(gvk)
		if err != nil {
			return err
		}
		layer.Deleted(obj)
	}
	result, err := client.Create(obj)
	if err != nil {
		return err
	}
	if c.layers != nil {
		// Record the result into cache.
		layer, err := c.layers.LayerFor(gvk)
		if err != nil {
			return err
		}
		layer.Created(result)
	}
	return nil
}

func jobEqual(desired, current *batchv1.Job) (bool, error) {
	err := setDefaultJob(desired)
	if err != nil {
		return false, err
	}

	currentSpec := current.Spec
	destSpec := currentSpec.DeepCopy()

	// apply desired spec to dest
	err = mergo.Merge(destSpec, desired.Spec, mergo.WithOverride)
	if err != nil {
		glog.Errorf("merge job spec error: %v", err)
		return false, err
	}

	currentSpecBytes, _ := json.Marshal(currentSpec)
	destSpecBytes, _ := json.Marshal(destSpec)

	if !reflect.DeepEqual(currentSpecBytes, destSpecBytes) {
		// something changed in spec
		glog.V(5).Infof("job spec changed, in cluster is %v, desired is %v", string(currentSpecBytes), string(destSpecBytes))
		return false, err
	}
	return true, nil

}

// TODO: the feature in utilfeature.DefaultFeatureGate must be the same as apiserver
func setDefaultJob(job *batchv1.Job) error {
	// set default in job spec and pod template
	k8sbatchv1.SetObjectDefaults_Job(job)

	in := job.Spec.Template.Spec
	out := k8score.Pod{}

	err := legacyscheme.Scheme.Convert(&in, &out.Spec, nil)
	if err != nil {
		return err
	}
	// drop disabled alpha fields in podSpec
	k8spodutil.DropDisabledPodFields(&out, nil)
	return legacyscheme.Scheme.Convert(&out.Spec, &job.Spec.Template.Spec, nil)
}

// Create creates all these resources.
func (c *client) Create(namespace string, resources []string, options CreateOptions) error {
	objs, err := c.objectsByOrder(resources, InstallOrder)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		accessor, err := c.codec.AccessorForObject(obj)
		if err != nil {
			return err
		}
		gvk := obj.GetObjectKind().GroupVersionKind()
		// Check whether the object exists.
		existence, err := c.getObject(gvk, namespace, accessor.GetName())
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
		client, err := c.pool.ClientFor(gvk, namespace)
		if err != nil {
			return err
		}
		result, err := client.Create(obj)
		if err != nil {
			return err
		}
		if c.layers != nil {
			// Record the result into cache.
			layer, err := c.layers.LayerFor(gvk)
			if err != nil {
				return err
			}
			layer.Created(result)
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
		opts := DeleteOptions{
			OwnerReferences: options.OwnerReferences,
			Filter:          options.Filter,
		}
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
		origin, accessor, err := c.codec.AccessorForResource(u.origin)
		if err != nil {
			return err
		}
		target, err := c.codec.ResourceToObject(u.target)
		if err != nil {
			return err
		}
		gvk := target.GetObjectKind().GroupVersionKind()
		current, err := c.getObject(gvk, namespace, accessor.GetName())
		if err != nil {
			return err
		}
		if !c.own(options.OwnerReferences, current) {
			return fmt.Errorf("attempt to update a non-affiliated object: %s/%s", namespace, accessor.GetName())
		}
		if options.Modifier != nil {
			if err = options.Modifier(origin, target, current); err != nil {
				return err
			}
		}
		old, err := json.Marshal(origin)
		if err != nil {
			return err
		}
		new, err := json.Marshal(target)
		if err != nil {
			return err
		}
		// TODO(kdada): Replace with merge patch when obj is TPR or CRD.
		patch, err := strategicpatch.CreateTwoWayMergePatch([]byte(old), []byte(new), current)
		if err != nil {
			return err
		}
		// Ignore empty patch.
		if len(patch) == 2 && string(patch) == "{}" {
			continue
		}
		client, err := c.pool.ClientFor(gvk, namespace)
		if err != nil {
			return err
		}
		result, err := client.Patch(accessor.GetName(), types.StrategicMergePatchType, patch)
		if err != nil {
			return err
		}
		if c.layers != nil {
			// Record the deleted obj into cache.
			layer, err := c.layers.LayerFor(gvk)
			if err != nil {
				return err
			}
			layer.Updated(result)
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
		if options.Filter != nil && options.Filter(obj) {
			continue
		}
		accessor, err := c.codec.AccessorForObject(obj)
		if err != nil {
			return err
		}
		gvk := obj.GetObjectKind().GroupVersionKind()
		obj, err := c.getObject(gvk, namespace, accessor.GetName())
		if err != nil {
			if errors.IsNotFound(err) {
				// Object is not found. Don't need delete
				continue
			}
			return err
		}

		if c.own(options.OwnerReferences, obj) {
			gvk := obj.GetObjectKind().GroupVersionKind()
			client, err := c.pool.ClientFor(gvk, namespace)
			if err != nil {
				return err
			}
			deletePolicy := metav1.DeletePropagationBackground
			err = client.Delete(accessor.GetName(), &metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
			if c.layers != nil {
				// Record the deleted obj into cache.
				layer, err := c.layers.LayerFor(gvk)
				if err != nil {
					return err
				}
				layer.Deleted(obj)
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
