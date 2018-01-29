package release

import (
	"reflect"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/release-controller/pkg/kube"
	"github.com/caicloud/release-controller/pkg/render"
	"github.com/caicloud/release-controller/pkg/storage"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
)

// applyRelease wouldn't delete anything. It leaves all antiquated resources to GC. So GC should take
// latest releases and delete useless resource.
func (rc *releaseContext) applyRelease(backend storage.ReleaseStorage, release *releaseapi.Release) error {
	var manifests []string
	if release.Spec.RollbackTo != nil {
		glog.V(4).Infof("Rollback release %s/%s to %v", release.Namespace, release.Name, release.Spec.RollbackTo.Version)
		// Rollback.
		rel, err := backend.Rollback(release.Spec.RollbackTo.Version)
		if err != nil {
			glog.Errorf("Failed to rollback release %s/%s: %v", release.Namespace, release.Name, err)
			return recordError(backend, err)
		}
		manifests = render.SplitManifest(rel.Status.Manifest)
	} else {
		glog.V(4).Infof("Apply release %s/%s", release.Namespace, release.Name)
		var history *releaseapi.ReleaseHistory
		if release.Status.Version > 0 {
			var err error
			history, err = backend.History(release.Status.Version)
			if err != nil && !errors.IsNotFound(err) {
				glog.Errorf("Failed to get release hisory for %s/%s: %v", release.Namespace, release.Name, err)
				return recordError(backend, err)
			}
		}
		if history != nil &&
			history.Spec.Config == release.Spec.Config &&
			reflect.DeepEqual(history.Spec.Template, release.Spec.Template) {
			glog.V(4).Infof("Release %s/%s has no updates", release.Namespace, release.Name)
			// No updates.
			manifests = render.SplitManifest(history.Spec.Manifest)
		} else {
			// Update.
			histories, err := backend.Histories()
			if err != nil {
				glog.Errorf("Failed to get histories for release %s/%s: %v", release.Namespace, release.Name, err)
				return recordError(backend, err)
			}
			if len(histories) > 0 {
				release.Status.Version = histories[0].Spec.Version + 1
			} else {
				release.Status.Version = 1
			}
			// Render
			carrier, err := rc.render.Render(&render.RenderOptions{
				Namespace: release.Namespace,
				Release:   release.Name,
				Version:   release.Status.Version,
				Template:  release.Spec.Template,
				Config:    release.Spec.Config,
			})
			if err != nil {
				// Record error status
				glog.Errorf("Failed to render release %s/%s: %v", release.Namespace, release.Name, err)
				return recordError(backend, err)
			}
			manifests = carrier.Resources()
			release.Status.Manifest = render.MergeResources(manifests)
			glog.V(4).Infof("Update release %s/%s for version %d", release.Namespace, release.Name, release.Status.Version)
			_, err = backend.Update(release)
			if err != nil {
				glog.Errorf("Failed to update release %s/%s: %v", release.Namespace, release.Name, err)
				return recordError(backend, err)
			}
		}
	}
	// Apply resources.
	if err := rc.client.Apply(release.Namespace, manifests, kube.ApplyOptions{
		OwnerReferences: referencesForRelease(release),
		// Modifier:        rc.apply,
	}); err != nil {
		glog.Infof("Failed to apply resources for release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}
	_, err := backend.FlushConditions(storage.ConditionAvailable())
	if err != nil {
		return err
	}
	glog.V(4).Infof("Applied release %s/%s for version %d", release.Namespace, release.Name, release.Status.Version)
	return nil
}
