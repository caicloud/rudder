package release

import (
	"context"
	"reflect"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/storage"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
)

type Action string

const (
	ReleaseNothing  Action = "ReleaseNothing"
	ReleaseCreate   Action = "ReleaseCreate"
	ReleaseUpdate   Action = "ReleaseUpdate"
	ReleaseRollback Action = "ReleaseRollback"
	ReleaseDelete   Action = "ReleaseDelete"
)

type releaseContext struct {
	client  kube.Client
	ignored []schema.GroupVersionKind
}

func NewReleaseHandler(client kube.Client, ignored []schema.GroupVersionKind) Handler {
	return (&releaseContext{
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
			if err := rc.applyRelease(backend, target); err != nil {
				if queue.NumRequeues(target.Name) < 3 {
					// Something is wrong. Retry it with rate limit.
					queue.AddRateLimited(target.Name)
					glog.Errorf("Can't apply release %s/%s: %v, retry", target.Namespace, target.Name, err)
				} else {
					glog.Warningf("Dropping release %s/%s", target.Namespace, target.Name)
				}
			} else {
				glog.V(4).Infof("Successfully handled release: %s/%s", target.Namespace, target.Name)
				// Everything is ok. Save target.
				queue.Forget(target.Name)
			}
			queue.Done(target.Name)
		}
	}()
FOR:
	for {
		select {
		case <-ctx.Done():
			break FOR
		case rel := <-getter.Get():
			if !(target != nil && rel.Spec.RollbackTo == nil &&
				target.Spec.Config == rel.Spec.Config &&
				reflect.DeepEqual(target.Spec.Suspend, rel.Spec.Suspend) &&
				reflect.DeepEqual(target.Spec.Template, rel.Spec.Template)) {
				// Config was changed. Add it to queue.
				target = rel
				queue.Forget(target.Name)
				queue.Add(target.Name)
			}
		}
	}
	queue.ShutDown()

	glog.V(2).Infof("Stopped handler: %s", getter.Key())
}
