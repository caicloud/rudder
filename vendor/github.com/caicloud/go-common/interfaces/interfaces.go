package interfaces

import (
	"reflect"
)

// IsNil returns true if the given interface itself or its value is nil.
func IsNil(obj interface{}) bool {
	return obj == nil || reflect.ValueOf(obj).IsNil()
}
