package release

import (
	"reflect"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/release-controller/pkg/storage"
	"k8s.io/apimachinery/pkg/api/errors"
)

// judgeNothing judge whether the release should do nothing.
// Conditions:
//  1. newOne has no change from the oldOne.
//  2. newOne has no change from the history.
func (rc *releaseContext) judgeNothing(backend storage.ReleaseStorage, oldOne *releaseapi.Release, newOne *releaseapi.Release) (bool, error) {
	if oldOne != nil {
		if reflect.DeepEqual(oldOne.Spec, newOne.Spec) {
			return true, nil
		}
	}
	if newOne.Status.Version > 0 {
		if newOne.Spec.RollbackTo != nil {
			return false, nil
		}
		history, err := backend.History(newOne.Status.Version)
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if newOne.Spec.Config == history.Spec.Config && reflect.DeepEqual(newOne.Spec.Template, history.Spec.Template) {
			return true, nil
		}
	}
	return false, nil
}

// judgeCreation judge whether the release should create resources.
// Conditions:
//  1. oldOne is nil and no history.
func (rc *releaseContext) judgeCreation(backend storage.ReleaseStorage, oldOne *releaseapi.Release, newOne *releaseapi.Release) (bool, error) {
	if newOne.Status.Version <= 0 {
		histories, err := backend.Histories()
		if err != nil {
			return false, err
		}
		if len(histories) == 0 {
			return true, nil
		}
	}
	return false, nil
}

// judgeRollback judge whether the release should rollback resources.
// Conditions:
//  1. newOne has RollbackTo and oldOne does not have.
//  2. The two releases have different RollbackTo.Version.
func (rc *releaseContext) judgeRollback(backend storage.ReleaseStorage, oldOne *releaseapi.Release, newOne *releaseapi.Release) (bool, error) {
	if oldOne == nil {
		if newOne.Spec.RollbackTo != nil {
			return true, nil
		}
	} else if newOne.Spec.RollbackTo != nil {
		if oldOne.Spec.RollbackTo == nil || newOne.Spec.RollbackTo.Version != oldOne.Spec.RollbackTo.Version {
			return true, nil
		}
	}
	return false, nil
}

// judgeUpdate judge whether the release should update resources.
// Conditions:
//  1. newOne has changes from history.
func (rc *releaseContext) judgeUpdate(backend storage.ReleaseStorage, oldOne *releaseapi.Release, newOne *releaseapi.Release) (bool, error) {
	if newOne.Status.Version > 0 {
		history, err := backend.History(newOne.Status.Version)
		if err != nil {
			return false, err
		}
		if newOne.Spec.Config != history.Spec.Config || !reflect.DeepEqual(newOne.Spec.Template, history.Spec.Template) {
			return true, nil
		}
	}
	return false, nil
}
