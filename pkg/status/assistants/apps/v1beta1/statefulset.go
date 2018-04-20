package v1beta1

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	apps "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

var gvkStatefulSet = apps.SchemeGroupVersion.WithKind("StatefulSet")

func StatefulSetAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	ss, ok := obj.(*apps.StatefulSet)
	if !ok {
		return status.None, fmt.Errorf("unknown type for stateful set: %s", obj.GetObjectKind().GroupVersionKind().String())
	}

	desired := int32(0)
	if ss.Spec.Replicas != nil {
		desired = *ss.Spec.Replicas
	}
	replicas := ss.Status.Replicas
	current := ss.Status.CurrentReplicas
	ready := ss.Status.ReadyReplicas
	switch {
	case desired == replicas && desired == current && desired == ready:
		// TODO(kdada): Check wrong pods for more precise verdict
		return status.Available, nil
	default:
		return status.Progressing, nil
	}
}
