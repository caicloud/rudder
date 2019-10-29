package v1

import (
	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
)

func JudgeConfigmap(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error) {
	return releaseapi.ResourceStatusFrom(releaseapi.ResourceRunning), nil
}
