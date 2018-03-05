package release

import (
	"fmt"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (rc *releaseContext) updateRelease(backend storage.ReleaseStorage, release *releaseapi.Release) error {
	glog.V(4).Infof("Update release: %s/%s", release.Namespace, release.Name)
	// Prepare
	_, err := backend.AddCondition(storage.ConditionUpdating())
	if err != nil {
		glog.Errorf("Failed to add condition for release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}

	originalManifests := render.SplitManifest(release.Status.Manifest)
	histories, err := backend.Histories()
	if err != nil {
		glog.Errorf("Failed to get histories for release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}
	nextVersion := int32(1)
	if len(histories) > 0 {
		nextVersion = histories[0].Spec.Version + 1
	}

	// Render
	carrier, err := rc.render.Render(&render.RenderOptions{
		Namespace: release.Namespace,
		Release:   release.Name,
		Version:   nextVersion,
		Template:  release.Spec.Template,
		Config:    release.Spec.Config,
	})
	if err != nil {
		glog.Errorf("Failed to render release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}
	resources := carrier.Resources()
	err = rc.client.Update(release.Namespace, originalManifests, resources, kube.UpdateOptions{
		OwnerReferences: referencesForRelease(release),
		Modifier:        rc.fix,
		Filter:          rc.ignore,
	})
	if err != nil {
		glog.Errorf("Failed to update resources for release %s/%s: %v", release.Namespace, release.Name, err)
		glog.Infof("Recover resources for release %s/%s", release.Namespace, release.Name)
		// Clear Resources
		if err := rc.client.Update(release.Namespace, resources, originalManifests, kube.UpdateOptions{
			OwnerReferences: referencesForRelease(release),
			Modifier:        rc.fix,
			// Don't need to ignore some resources here.
		}); err != nil {
			glog.Infof("Failed to recover resources for release %s/%s: %v", release.Namespace, release.Name, err)
		}
		return recordError(backend, err)
	}
	// Record success status
	release.Status.Manifest = render.MergeResources(resources)
	_, err = backend.Update(release)
	if err != nil {
		glog.Errorf("Failed to update release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}
	glog.V(4).Infof("Updated release: %s/%s", release.Namespace, release.Name)
	return nil
}

// fix modifies new object.
func (rc *releaseContext) fix(origin, new, current runtime.Object) error {
	switch origin.GetObjectKind().GroupVersionKind() {
	case apiv1.SchemeGroupVersion.WithKind("Service"):
		// Fix service when convert NodePort to ClusterIP.
		o, ok := origin.(*apiv1.Service)
		if !ok {
			return fmt.Errorf("can't convert origin object %v to service", origin.GetObjectKind())
		}
		n, ok := new.(*apiv1.Service)
		if !ok {
			return fmt.Errorf("can't convert new object %v to service", new.GetObjectKind())
		}
		c, ok := current.(*apiv1.Service)
		if !ok {
			return fmt.Errorf("can't convert current object %v to service", current.GetObjectKind())
		}
		if o.Spec.Type == apiv1.ServiceTypeNodePort && n.Spec.Type == apiv1.ServiceTypeClusterIP {
			// Set NodePort for origin object
			for i := 0; i < len(o.Spec.Ports) && i < len(c.Spec.Ports); i++ {
				o.Spec.Ports[i].NodePort = c.Spec.Ports[i].NodePort
			}
			// Clear NodePort for new object
			ports := n.Spec.Ports
			for i := range ports {
				ports[i].NodePort = 0
			}
		}
	}
	return nil
}
