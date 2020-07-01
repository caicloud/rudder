package gc

import (
	"fmt"
	"reflect"
	"testing"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newRelease(name string, version int32) *releaseapi.Release {
	return &releaseapi.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: releaseapi.ReleaseStatus{
			Version: version,
		},
	}
}

func TestIsRetainHistory(t *testing.T) {
	gc := &GarbageCollector{
		historyLimit: 5,
	}
	testCases := []struct {
		rls        string
		curVersion int32
		rlsHistory string
		want       bool
		err        error
	}{
		{"hello", 3, "hello-v1", true, nil},
		{"hello", 10, "hello-v1", false, nil},
		{"hello", 3, "hello-v0", false, fmt.Errorf("cur rlshistory hello-v0 version 0 is  invalid")},
		{"hello", 3, "hello-v-9", false, fmt.Errorf("cur rlshistory hello-v-9 version -9 is  invalid")},
		{"hello", 10, "hello-v5", false, nil},
		{"hello", 10, "hello-v6", true, nil},
	}

	for _, ca := range testCases {
		get, err := gc.ifRetainHistory(newRelease(ca.rls, ca.curVersion), ca.rlsHistory)
		if reflect.DeepEqual(err, ca.want) || get != ca.want {
			t.Errorf("limit %v got %v but condition is %v err %v", gc.historyLimit, get, ca, err)
		}
	}
}
