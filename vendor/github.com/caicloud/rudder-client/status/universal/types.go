package universal

import (
	"github.com/caicloud/clientset/listerfactory"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Assistant handles a kind of object. It will generates the resourceStatus for the object.
type Assistant func(factory listerfactory.ListerFactory, obj runtime.Object) (releaseapi.ResourceStatus, error)

// Umpire can employs many assistant to handle many kinds of objects.
type Umpire interface {
	// Employ employs an assistant for specified object kind.
	Employ(gvk schema.GroupVersionKind, assistant Assistant)
	// Judge judges the object and generates a sentence.
	Judge(obj runtime.Object) (releaseapi.ResourceStatus, error)
}
