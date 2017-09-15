package kube

import (
	"fmt"
	"strings"

	"github.com/caicloud/clientset/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

// APIResources contains all resources and kinds.
type APIResources interface {
	// ResourceFor gets api resource by GroupVersionKind
	ResourceFor(gvk schema.GroupVersionKind) (*Resource, error)
	// Resources gets all api resources.
	Resources() map[schema.GroupVersionKind]*Resource
}

// Resource is API resource
type Resource struct {
	// APIResource is original api resource.
	metav1.APIResource
	// Group is the gourp name of current api resource.
	Group string
	// Version is the version of current api resource.
	Version string
	// Subresources contains the subresources of current api resource.
	Subresources []*Resource
}

// GroupVersionKind gets the GroupVersionKind of the resource.
func (r *Resource) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   r.Group,
		Version: r.Version,
		Kind:    r.APIResource.Kind,
	}
}

// GroupVersionResource gets the GroupVersionResource of the resource.
func (r *Resource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.Group,
		Version:  r.Version,
		Resource: r.APIResource.Name,
	}
}

// apiResources contains all api resources.
type apiResources struct {
	resources map[schema.GroupVersionKind]*Resource
}

// NewAPIResourcesByConfig creates APIResources by kube config.
func NewAPIResourcesByConfig(config *rest.Config) (APIResources, error) {
	configCopy := *config
	config = &configCopy
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return NewAPIResources(kubeClient)
}

// NewAPIResources creates APIResources by kube client.
func NewAPIResources(client kubernetes.Interface) (APIResources, error) {
	resources, err := client.Discovery().ServerResources()
	if err != nil {
		return nil, err
	}

	apiResources := &apiResources{
		resources: make(map[schema.GroupVersionKind]*Resource),
	}
	for _, list := range resources {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			return nil, err
		}
		for _, resource := range list.APIResources {
			gvk := gv.WithKind(resource.Kind)
			res, ok := apiResources.resources[gvk]
			if !strings.Contains(resource.Name, "/") {
				// Root resource
				if ok {
					res.APIResource = resource
				} else {
					apiResources.resources[gvk] = &Resource{
						APIResource: resource,
						Group:       gv.Group,
						Version:     gv.Version,
					}
				}
			} else {
				// Subresource
				if !ok {
					res = &Resource{
						Group:   gv.Group,
						Version: gv.Version,
					}
					apiResources.resources[gvk] = res
				}
				res.Subresources = append(res.Subresources, &Resource{
					APIResource: resource,
					Group:       gv.Group,
					Version:     gv.Version,
				})
			}
		}
	}
	return apiResources, nil
}

// ResourceFor gets api resource by GroupVersionKind
func (ar *apiResources) ResourceFor(gvk schema.GroupVersionKind) (*Resource, error) {
	resource, ok := ar.resources[gvk]
	if !ok {
		return nil, fmt.Errorf("can't find api resource for: %s", gvk)
	}
	return resource, nil
}

// Resources gets all api resources.
func (ar *apiResources) Resources() map[schema.GroupVersionKind]*Resource {
	return ar.resources
}
