package v1beta1

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkDaemonSet = extensions.SchemeGroupVersion.WithKind("DaemonSet")

func DaemonSetAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	ds, ok := obj.(*extensions.DaemonSet)
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
