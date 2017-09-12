package storage

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	ReasonAvailable   = "Available"
	ReasonFailure     = "Failure"
	ReasonCreating    = "Creating"
	ReasonUpdating    = "Updating"
	ReasonRollbacking = "Rollbacking"
)

func ConditionAvailable() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseAvailable,
		Status:             apiv1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonAvailable,
		Message:            "",
	}
}

func ConditionFailure(message string) releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseFailure,
		Status:             apiv1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonFailure,
		Message:            message,
	}
}

func ConditionCreating() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseProgressing,
		Status:             apiv1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonCreating,
		Message:            "",
	}
}

func ConditionUpdating() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseProgressing,
		Status:             apiv1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonUpdating,
		Message:            "",
	}
}

func ConditionRollbacking() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseProgressing,
		Status:             apiv1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonRollbacking,
		Message:            "",
	}
}
