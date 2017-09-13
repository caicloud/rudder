package release

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/release-controller/pkg/kube"
	"github.com/caicloud/release-controller/pkg/render"
	"github.com/caicloud/release-controller/pkg/storage"
	"github.com/golang/glog"
)

func (rc *releaseContext) deleteRelease(backend storage.ReleaseStorage, release *releaseapi.Release) error {
	glog.V(4).Infof("Delete release: %s/%s", release.Namespace, release.Name)
	// Delete resources
	manifests := render.SplitManifest(release.Status.Manifest)
	err := rc.client.Delete(release.Namespace, manifests, kube.DeleteOptions{
		OwnerReferences: referencesForRelease(release),
	})
	if err != nil {
		glog.Errorf("Failed to delete release: %v", err)
		return err
	}
	// Delete release
	err = backend.Delete()
	if err != nil {
		glog.Errorf("Failed to delete release: %v", err)
		return err
	}
	glog.V(4).Infof("Deleted release: %s/%s", release.Namespace, release.Name)
	return nil
}
