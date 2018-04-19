package status

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/api/core/v1"
)

func TestJudgePodStatus(t *testing.T) {
	type args struct {
		pod *v1.Pod
	}
	tests := []struct {
		name string
		pod  *v1.Pod
		want PodStatus
	}{
		{
			"unschedulable",
			&v1.Pod{
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					Conditions: []v1.PodCondition{
						{Type: v1.PodScheduled, Status: v1.ConditionFalse, Reason: v1.PodReasonUnschedulable},
					},
				},
			},
			PodStatus{Ready: false, Phase: PodError, Reason: v1.PodReasonUnschedulable},
		},
		{
			"pending",
			&v1.Pod{
				Status: v1.PodStatus{
					Phase:      v1.PodPending,
					Conditions: []v1.PodCondition{},
					Reason:     "pod is pending",
				},
			},
			PodStatus{Ready: false, Phase: v1.PodPending, Reason: "pod is pending"},
		},
		{
			"ready",
			&v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Name: "ready"}},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
					},
				},
			},
			PodStatus{Ready: true, ReadyContainers: 1, TotalContainers: 1, Phase: v1.PodRunning, Reason: string(v1.PodRunning)},
		},
		{
			"terminating",
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{}},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Name: "ready"}},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
					},
				},
			},
			PodStatus{Ready: false, ReadyContainers: 1, TotalContainers: 1, Phase: PodTerminating, Reason: "Terminating"},
		},
		{
			"initializing",
			&v1.Pod{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{{Name: "init1"}, {Name: "init2"}},
					Containers:     []v1.Container{{Name: "ready"}},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					InitContainerStatuses: []v1.ContainerStatus{
						{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}}},
						{State: v1.ContainerState{Waiting: &v1.ContainerStateWaiting{Reason: "Waiting", Message: "Waiting"}}},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
					},
				},
			},
			PodStatus{Ready: false, InitContainers: 2, ReadyContainers: 0, TotalContainers: 1, Phase: PodInitializing, Reason: "Init:Waiting", Message: "Waiting"},
		},
		{
			"initializing2",
			&v1.Pod{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{{Name: "init1"}, {Name: "init2"}},
					Containers:     []v1.Container{{Name: "ready"}},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					InitContainerStatuses: []v1.ContainerStatus{
						{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}}},
						{State: v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}}},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
					},
				},
			},
			PodStatus{Ready: false, InitContainers: 2, ReadyContainers: 0, TotalContainers: 1, Phase: PodInitializing, Reason: "Init:1/2", Message: string(PodInitializing)},
		},
		{
			"initializing error",
			&v1.Pod{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{{Name: "init1"}, {Name: "init2"}},
					Containers:     []v1.Container{{Name: "ready"}},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					InitContainerStatuses: []v1.ContainerStatus{
						{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}}},
						{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 1, Signal: 9, Reason: "Exit with 1"}}},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
					},
				},
			},
			PodStatus{Ready: false, InitContainers: 2, ReadyContainers: 0, TotalContainers: 1, Phase: PodError, Reason: "Init:Exit with 1"},
		},
		{
			"running error",
			&v1.Pod{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{{Name: "init1"}},
					Containers:     []v1.Container{{Name: "ready"}, {Name: "error"}},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					InitContainerStatuses: []v1.ContainerStatus{
						{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}}},
					},
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
						{Ready: false, State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 2, Signal: 9, Reason: "Exit with 2"}}},
					},
				},
			},
			PodStatus{Ready: false, InitContainers: 1, ReadyContainers: 1, TotalContainers: 2, Phase: PodError, Reason: "Exit with 2"},
		},
		{
			"CrashLoopBackOff",
			&v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Name: "error"}},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Ready: false,
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason:  "CrashLoopBackOff",
									Message: "Back-off 5m0s restarting failed container=c0 pod=mysql-mysql-v1-0-3888019538-vs2qd_qaz(e8ba3f78-204f-11e8-b3ff-525400c2714a)",
								},
							},
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									ExitCode: 137,
									Reason:   "OOMKilled",
								},
							},
							RestartCount: 6,
						},
					},
				},
			},
			PodStatus{
				Ready: false, ReadyContainers: 0, TotalContainers: 1, RestartCount: 6,
				Phase:   PodError,
				Reason:  "OOMKilled",
				Message: "Back-off 5m0s restarting failed container=c0 pod=mysql-mysql-v1-0-3888019538-vs2qd_qaz(e8ba3f78-204f-11e8-b3ff-525400c2714a)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JudgePodStatus(tt.pod); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JudgePodStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
