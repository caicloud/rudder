package node

import (
	"net"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetNodeHostIP(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name        string
		node        *v1.Node
		labels      []string
		annotations []string
		want        net.IP
		wantErr     bool
	}{
		{
			"",
			&v1.Node{
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeExternalIP,
							Address: "error",
						},
						{
							Type:    v1.NodeExternalIP,
							Address: "172.16.0.1",
						},
					},
				},
			},
			nil,
			nil,
			net.ParseIP("172.16.0.1"),
			false,
		},
		{
			"",
			&v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"annotations/nodeIP": "45.199.12.10",
					},
					Labels: map[string]string{
						"labels/nodeIP": "10.0.1.10",
					},
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeExternalIP,
							Address: "192.168.1.1",
						},
						{
							Type:    v1.NodeInternalIP,
							Address: "172.16.0.1",
						},
					},
				},
			},
			[]string{"labels/nodeIP"},
			[]string{"annotations/nodeIP"},
			net.ParseIP("10.0.1.10"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNodeHostIP(tt.node, tt.labels, tt.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodeHostIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodeHostIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
