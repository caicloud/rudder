package apply

import (
	"testing"
	"reflect"
)

func TestServiceMergeAnnotations(t *testing.T) {
	var testSets = []struct {
		current map[string]string
		desired map[string]string
		expected map[string]string
	} {
		{nil, nil, nil},
		{
			map[string]string{"current":"true"},
			nil,
			map[string]string{"current": "true"},
		},
		{
			nil,
			map[string]string{"desired":"true"},
			map[string]string{"desired": "true"},
		},
		{
			map[string]string{"current":"true", "desired":"false"},
			map[string]string{"desired":"true"},
			map[string]string{"current":"true","desired":"true"},
		},
	}

	for _, test := range testSets {
		out := mergeAnnotations(test.current, test.desired)
		if !reflect.DeepEqual(out, test.expected) {
			t.Errorf("got %v, want %v", out, test.expected)
		}
	}
}
