package assistants

import (
	"fmt"

	"github.com/caicloud/release-controller/pkg/status"
	"github.com/caicloud/release-controller/pkg/store"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var gvkStatefulSet = appsv1beta1.SchemeGroupVersion.WithKind("StatefulSet")

func StatefulSetAssistant(store store.IntegrationStore, obj runtime.Object) (status.Sentence, error) {
	ss, ok := obj.(*appsv1beta1.StatefulSet)
	if !ok {
		return status.None, fmt.Errorf("unknown type for stateful set: %s", obj.GetObjectKind().GroupVersionKind().String())
	}

	desired := int32(0)
	if ss.Spec.Replicas != nil {
		desired = *ss.Spec.Replicas
	}
	current := ss.Status.Replicas
	switch {
	case desired == current:
		// TODO(kdada): Check wrong pods for more precise verdict
		return status.Available, nil
	default:
		return status.Progressing, nil
	}
}
