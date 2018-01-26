package apply

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Applier is used to update objects.
// Some fields of a kind be immutable, Appliers fix it.
type Applier func(current, desired runtime.Object) error

var appliers = map[schema.GroupVersionKind]Applier{}

// RegisterApplier registers an applier for specific gvk.
func RegisterApplier(gvk schema.GroupVersionKind, applier Applier) {
	appliers[gvk] = applier
}

// Apply fixes objects.
func Apply(gvk schema.GroupVersionKind, current, desired runtime.Object) error {
	if applier, ok := appliers[gvk]; ok {
		return applier(current, desired)
	}
	return nil
}
