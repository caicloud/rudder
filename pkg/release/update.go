package release

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/release-controller/pkg/kube"
	"github.com/caicloud/release-controller/pkg/render"
	"github.com/caicloud/release-controller/pkg/storage"
	"github.com/golang/glog"
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
	})
	if err != nil {
		glog.Errorf("Failed to update resources for release %s/%s: %v", release.Namespace, release.Name, err)
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
