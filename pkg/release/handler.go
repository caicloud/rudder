package release

import (
	"context"
	"fmt"
	"reflect"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
)

type ReleaseAction string

const (
	ReleaseNothing  ReleaseAction = "ReleaseNothing"
	ReleaseCreate   ReleaseAction = "ReleaseCreate"
	ReleaseUpdate   ReleaseAction = "ReleaseUpdate"
	ReleaseRollback ReleaseAction = "ReleaseRollback"
	ReleaseDelete   ReleaseAction = "ReleaseDelete"

	maxRetries = 3
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

	// Retry queue.
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	// target only has single read and single write thread. So don't need a lock here.
	// target never is nil.
	var target *releaseapi.Release
	go func() {
		for {
			_, shutdown := queue.Get()
			if shutdown {
				return
			}
			// In the past, call handleRelease to judge and select an handler for release.
			// Now just apply the release.
			err := rc.applyRelease(backend, target)
			if err == nil {
				// Everything is ok. Save target.
				glog.V(4).Infof("Successfully handled release: %s/%s", target.Namespace, target.Name)
				queue.Forget(target.Name)
			} else if queue.NumRequeues(target.Name) < maxRetries {
				// Something is wrong. Retry it with rate limit.
				glog.Errorf("Can't apply release %s/%s retry: %v, err: %v",
					target.Namespace, target.Name, queue.NumRequeues(target.Name), err)
				queue.AddRateLimited(target.Name)
			} else {
				// exceed max retry times
				glog.Errorf("Dropping release %s/%s from the queue, err: %v", target.Namespace, target.Name, err)
				queue.Forget(target.Name)
			}
			queue.Done(target.Name)
		}
	}()
FOR:
	for {
		select {
		case _ = <-ctx.Done():
			break FOR
		case rel := <-getter.Get():
			if !(target != nil && rel.Spec.RollbackTo == nil &&
				target.Spec.Config == rel.Spec.Config &&
				reflect.DeepEqual(target.Spec.Template, rel.Spec.Template)) {
				// Config was changed. Add it to queue.
				queue.Forget(target.Name)
			}
			target = rel
			queue.Add(target.Name)
		}
	}
	queue.ShutDown()
	// Delete release resources.
	// if err := rc.deleteRelease(backend, target); err != nil {
	//     glog.Errorf("Can't delete release %s/%s: %v", target.Namespace, target.Name, err)
	// }

	glog.V(2).Infof("Stopped handler: %s", getter.Key())
}

func (rc *releaseContext) handleRelease(backend storage.ReleaseStorage, origin, target *releaseapi.Release) error {
	// create/rollback/update resources
	action, err := rc.judge(backend, origin, target)
	if err != nil {
		return err
	}
	switch action {
	case ReleaseCreate:
		// Create
		return rc.createRelease(backend, target)
	case ReleaseRollback:
		// Rollback
		return rc.rollbackRelease(backend, target)
	case ReleaseUpdate:
		// Update
		return rc.updateRelease(backend, target)
	}
	return nil
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
