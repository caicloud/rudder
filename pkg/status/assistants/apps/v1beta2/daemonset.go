package v1beta2

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkDaemonSet = apps.SchemeGroupVersion.WithKind("DaemonSet")

func DaemonSetAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	ds, ok := obj.(*apps.DaemonSet)
	if !ok {
		return status.None, fmt.Errorf("unknown type for daemon set: %s", obj.GetObjectKind().GroupVersionKind().String())
	}

	desired := ds.Status.DesiredNumberScheduled
	current := ds.Status.CurrentNumberScheduled
	updated := ds.Status.UpdatedNumberScheduled
	available := ds.Status.NumberAvailable
	unavailable := ds.Status.NumberUnavailable
	switch {
	case unavailable == 0 && desired == current && desired == updated && desired == available:
		return status.Available, nil
	case unavailable > 0 && desired == current && desired != available:
		// TODO(kdada): Check wrong pods for more precise verdict
		return status.Failure, nil
	default:
		return status.Progressing, nil
	}
}
