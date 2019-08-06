package release

import (
	"fmt"
	"reflect"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// applyRelease wouldn't delete anything. It leaves all antiquated resources to GC. So GC should take
// latest releases and delete useless resource.
func (rc *releaseContext) applyRelease(backend storage.ReleaseStorage, release *releaseapi.Release) error {
	// Deep copy release. Avoid modifying original release.
	release = release.DeepCopy()

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

		// current max_version correct_version changed     nextVersion
		// 0       0            1               true/false  1
		// 0       2            2               true        3
		// 0       2            2               false       2
		// 1       2            1               false       1
		// 1       2            1               true        3
		// 2       2            2               true        3
		// 2       2            2               false       2
		// 3       2            2               true        3
		// 3       2            2               false       2

		// calculate version
		var correctedVersion, nextVersion int32

		var err error
		// find the history with version
		// get the history with biggest version
		histories, err := backend.Histories()
		if err != nil {
			glog.Errorf("Failed to get histories for release %s/%s: %v", release.Namespace, release.Name, err)
			return recordError(backend, err)
		}

		if len(histories) == 0 {
			correctedVersion = 1
			nextVersion = 1
		} else {
			var currentHistory *releaseapi.ReleaseHistory
			latestHistory := &histories[0]
			latestVersion := latestHistory.Spec.Version
			changed := false

			// Assuming that the latest history is current history
			currentHistory = latestHistory
			// try to find out the real history
			for _, h := range histories {
				if h.Spec.Version == release.Status.Version {
					currentHistory = &h
					break
				}
			}

			if release.Spec.Config != currentHistory.Spec.Config ||
				!reflect.DeepEqual(release.Spec.Template, currentHistory.Spec.Template) {
				changed = true
			}

			correctedVersion = currentHistory.Spec.Version

			if !changed {
				// nothing changed, nextVersion is correctedVersion
				nextVersion = correctedVersion
			} else {
				// if somthing changed, the nextVersion always be latestVersion + 1
				nextVersion = latestVersion + 1
			}

		}

		release.Status.Version = nextVersion

		// check the manifests

		// FIX: use temporary render to avoid concurrent issue
		carrier, err := render.NewRender().Render(&render.RenderOptions{
			Namespace: release.Namespace,
			Release:   release.Name,
			Version:   release.Status.Version,
			Template:  release.Spec.Template,
			Config:    release.Spec.Config,
			Suspend:   release.Spec.Suspend,
		})
		if err != nil {
			// Record error status
			glog.Errorf("Failed to render release %s/%s: %v", release.Namespace, release.Name, err)
			return recordError(backend, err)
		}

		manifests = carrier.Resources()
		release.Status.Manifest = render.MergeResources(manifests)

		glog.V(4).Infof("Update manifest of release %s/%s for version %d", release.Namespace, release.Name, release.Status.Version)
		_, err = backend.Update(release)
		if err != nil {
			glog.Errorf("Failed to update release %s/%s: %v", release.Namespace, release.Name, err)
			return recordError(backend, err)
		}

	}
	// Apply resources.
	if err := rc.client.Apply(release.Namespace, manifests, kube.ApplyOptions{
		OwnerReferences: referencesForRelease(release),
		Checker:         rc.ignore,
	}); err != nil {
		glog.Infof("Failed to apply resources for release %s/%s: %v", release.Namespace, release.Name, err)
		return recordError(backend, err)
	}
	_, err := backend.FlushConditions(storage.Condition(storage.ReleaseReasonAvailable, ""))
	if err != nil {
		return err
	}
	glog.V(4).Infof("Applied release %s/%s for version %d", release.Namespace, release.Name, release.Status.Version)
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

// fix modifies new object.
func (rc *releaseContext) fix(origin, new, current runtime.Object) error {
	switch origin.GetObjectKind().GroupVersionKind() {
	case core.SchemeGroupVersion.WithKind("Service"):
		// Fix service when convert NodePort to ClusterIP.
		o, ok := origin.(*core.Service)
		if !ok {
			return fmt.Errorf("can't convert origin object %v to service", origin.GetObjectKind())
		}
		n, ok := new.(*core.Service)
		if !ok {
			return fmt.Errorf("can't convert new object %v to service", new.GetObjectKind())
		}
		c, ok := current.(*core.Service)
		if !ok {
			return fmt.Errorf("can't convert current object %v to service", current.GetObjectKind())
		}
		if o.Spec.Type == core.ServiceTypeNodePort && n.Spec.Type == core.ServiceTypeClusterIP {
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
