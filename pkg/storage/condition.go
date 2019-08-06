package storage

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type releaseConditionReason string

const (
	ReleaseReasonAvailable   releaseConditionReason = "Available"
	ReleaseReasonFailure     releaseConditionReason = "Failure"
	ReleaseReasonCreating    releaseConditionReason = "Creating"
	ReleaseReasonUpdating    releaseConditionReason = "Updating"
	ReleaseReasonRollbacking releaseConditionReason = "Rollbacking"
)

// Condition returns a release condition based on given release condition reason.
func Condition(r releaseConditionReason, msg string) releaseapi.ReleaseCondition {
	ret := releaseapi.ReleaseCondition{
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Message:            msg,
		Reason:             string(r),
	}
	switch r {
	case ReleaseReasonAvailable:
		ret.Type = releaseapi.ReleaseAvailable
	case ReleaseReasonFailure:
		ret.Type = releaseapi.ReleaseFailure
	case ReleaseReasonCreating, ReleaseReasonUpdating, ReleaseReasonRollbacking:
		ret.Type = releaseapi.ReleaseProgressing
	}
	return ret
}
