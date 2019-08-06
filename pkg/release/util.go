package release

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// referencesForRelease create references for a release.
func referencesForRelease(release *releaseapi.Release) []metav1.OwnerReference {
	return []metav1.OwnerReference{{
		APIVersion: releaseapi.SchemeGroupVersion.String(),
		Kind:       "Release",
		Name:       release.Name,
		UID:        release.UID,
	}}
}

// recordError records err for release.
func recordError(backend storage.ReleaseStorage, target error) error {
	// Record error status
	_, err := backend.FlushConditions(storage.Condition(storage.ReleaseReasonFailure, target.Error()))
	if err == nil {
		return target
	}
	return err
}
