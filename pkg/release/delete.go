package release

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
)

func (rc *releaseContext) deleteRelease(backend storage.ReleaseStorage, release *releaseapi.Release) error {
	glog.V(4).Infof("Delete release: %s/%s", release.Namespace, release.Name)
	// Delete resources
	manifests := render.SplitManifest(release.Status.Manifest)
	err := rc.client.Delete(release.Namespace, manifests, kube.DeleteOptions{
		OwnerReferences: referencesForRelease(release),
		Filter:          rc.ignore,
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

// ignore checks if an object should be ignored.
func (rc *releaseContext) ignore(obj runtime.Object) bool {
	for _, i := range rc.ignored {
		if i == obj.GetObjectKind().GroupVersionKind() {
			return true
		}
	}
	return false
}
