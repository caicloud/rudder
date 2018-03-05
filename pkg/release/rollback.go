package release

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/golang/glog"
)

func (rc *releaseContext) rollbackRelease(backend storage.ReleaseStorage, release *releaseapi.Release) error {
	glog.V(4).Infof("Rollback release: %s/%s", release.Namespace, release.Name)
	// Prepare
	_, err := backend.AddCondition(storage.ConditionRollbacking())
	if err != nil {
		glog.Errorf("Failed to add condition for release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}

	originalManifests := render.SplitManifest(release.Status.Manifest)
	// Get history
	h, err := backend.History(release.Spec.RollbackTo.Version)
	if err != nil {
		glog.Errorf("Failed to get history for release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}

	manifests := render.SplitManifest(h.Spec.Manifest)
	err = rc.client.Update(release.Namespace, originalManifests, manifests, kube.UpdateOptions{
		OwnerReferences: referencesForRelease(release),
		Modifier:        rc.fix,
		Filter:          rc.ignore,
	})
	if err != nil {
		glog.Errorf("Failed to rollback resources for release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}
	// Record success status
	_, err = backend.Rollback(release.Spec.RollbackTo.Version)
	if err != nil {
		glog.Errorf("Failed to rollback release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}
	glog.V(4).Infof("Rollbacked release: %s/%s", release.Namespace, release.Name)
	return nil
}
