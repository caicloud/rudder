package kube

import (
	"fmt"
	"strings"

	"github.com/caicloud/clientset/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type APIResources interface {
	ResourceFor(gvk schema.GroupVersionKind) (*Resource, error)
	Resources() map[schema.GroupVersionKind]*Resource
}

type Resource struct {
	metav1.APIResource
	Group        string
	Version      string
	Subresources []*Resource
}

func (r *Resource) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   r.Group,
		Version: r.Version,
		Kind:    r.APIResource.Kind,
	}
}

func (r *Resource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.Group,
		Version:  r.Version,
		Resource: r.APIResource.Name,
	}
}

type apiResources struct {
	resources map[schema.GroupVersionKind]*Resource
}

func NewAPIResourcesByConfig(config *rest.Config) (APIResources, error) {
	configCopy := *config
	config = &configCopy
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return NewAPIResources(kubeClient)
}

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

func (ar *apiResources) ResourceFor(gvk schema.GroupVersionKind) (*Resource, error) {
	resource, ok := ar.resources[gvk]
	if !ok {
		return nil, fmt.Errorf("can't find api resource for: %s", gvk)
	}
	return resource, nil
}

func (ar *apiResources) Resources() map[schema.GroupVersionKind]*Resource {
	return ar.resources
}
