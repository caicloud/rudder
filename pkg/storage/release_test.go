package storage

import (
	"testing"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
)

func Test_conditionsEqual(t *testing.T) {
	type args struct {
		conditions1 []releaseapi.ReleaseCondition
		conditions2 []releaseapi.ReleaseCondition
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			args: args{
				conditions1: nil,
				conditions2: nil,
			},
			want: true,
		},
		{
			args: args{
				conditions1: nil,
				conditions2: []releaseapi.ReleaseCondition{},
			},
			want: true,
		},
		{
			args: args{
				conditions1: nil,
				conditions2: []releaseapi.ReleaseCondition{
					{
						Type:   releaseapi.ReleaseAvailable,
						Status: "True",
						Reason: "Available",
					},
				},
			},
			want: false,
		},
		{
			args: args{
				conditions1: []releaseapi.ReleaseCondition{
					{
						Type:   releaseapi.ReleaseAvailable,
						Status: "False",
						Reason: "Available",
					},
				},
				conditions2: []releaseapi.ReleaseCondition{
					{
						Type:   releaseapi.ReleaseAvailable,
						Status: "True",
						Reason: "Available",
					},
				},
			},
			want: false,
		},
		{
			args: args{
				conditions1: []releaseapi.ReleaseCondition{
					{
						Type:   releaseapi.ReleaseAvailable,
						Status: "True",
						Reason: "Available",
					},
				},
				conditions2: []releaseapi.ReleaseCondition{
					{
						Type:   releaseapi.ReleaseAvailable,
						Status: "True",
						Reason: "Available",
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conditionsEqual(tt.args.conditions1, tt.args.conditions2); got != tt.want {
				t.Errorf("conditionsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
