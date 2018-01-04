package release

import (
	"context"
	"fmt"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/release-controller/pkg/kube"
	"github.com/caicloud/release-controller/pkg/render"
	"github.com/caicloud/release-controller/pkg/storage"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ReleaseAction string

const (
	ReleaseNothing  ReleaseAction = "ReleaseNothing"
	ReleaseCreate   ReleaseAction = "ReleaseCreate"
	ReleaseUpdate   ReleaseAction = "ReleaseUpdate"
	ReleaseRollback ReleaseAction = "ReleaseRollback"
	ReleaseDelete   ReleaseAction = "ReleaseDelete"
)

type releaseContext struct {
	render  render.Render
	client  kube.Client
	ignored []schema.GroupVersionKind
}

func NewReleaseHandler(render render.Render, client kube.Client, ignored []schema.GroupVersionKind) Handler {
	return (&releaseContext{
		render:  render,
		client:  client,
		ignored: ignored,
	}).handle
}

// Handle handles a release and its updates
func (rc *releaseContext) handle(ctx context.Context, backend storage.ReleaseStorage, getter Getter) {
	glog.V(2).Infof("Start handler: %s", getter.Key())

	var release *releaseapi.Release
FOR:
	for {
		select {
		case _ = <-ctx.Done():
			break FOR
		case target := <-getter.Get():
			// create/rollback/update resources
			action, err := rc.judge(backend, release, target)
			if err != nil {
				// TODO(kdada): Re-enqueue the obj and handle it at appropriate time
				glog.V(4).Infof("Can't judge an action for release: %+v", target)
				glog.Errorf("Can't judge an action for release: %v", err)
				continue
			}
			release = target
			switch action {
			case ReleaseCreate:
				// Create
				err = rc.createRelease(backend, release)
			case ReleaseRollback:
				// Rollback
				err = rc.rollbackRelease(backend, release)
			case ReleaseUpdate:
				// Update
				err = rc.updateRelease(backend, release)
			}
			if err != nil {
				// TODO(kdada): Re-enqueue the obj and handle it at appropriate time
				glog.Errorf("Can't do action for %s/%s: %v", release.Namespace, release.Name, err)
				continue
			}
		}
	}
	if release == nil {
		glog.Errorf("No available release to clean")
	} else {
		// Delete
		if err := rc.deleteRelease(backend, release); err != nil {
			// TODO(kdada): Re-enqueue the obj and handle it at appropriate time
			glog.Errorf("Can't delete release %s/%s: %v", release.Namespace, release.Name, err)
		}
	}
	glog.V(2).Infof("Stopped handler: %s", getter.Key())
}

func (rc *releaseContext) judge(backend storage.ReleaseStorage, oldOne *releaseapi.Release, newOne *releaseapi.Release) (ReleaseAction, error) {
	checked, err := rc.judgeNothing(backend, oldOne, newOne)
	if err != nil {
		return "", err
	}
	if checked {
		return ReleaseNothing, nil
	}
	checked, err = rc.judgeCreation(backend, oldOne, newOne)
	if err != nil {
		return "", err
	}
	if checked {
		return ReleaseCreate, nil
	}
	checked, err = rc.judgeRollback(backend, oldOne, newOne)
	if err != nil {
		return "", err
	}
	if checked {
		return ReleaseRollback, nil
	}
	checked, err = rc.judgeUpdate(backend, oldOne, newOne)
	if err != nil {
		return "", err
	}
	if checked {
		return ReleaseUpdate, nil
	}
	return "", fmt.Errorf("unknown release action: %s/%s", newOne.Namespace, newOne.Name)
}
