package storage

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Reason string

const (
	Available   Reason = "Available"
	Failure     Reason = "Failure"
	Creating    Reason = "Creating"
	Updating    Reason = "Updating"
	Rollbacking Reason = "Rollbacking"
)

// Condition returns a release condition based on given reason.
func Condition(reason Reason, msg string) releaseapi.ReleaseCondition {
	ret := releaseapi.ReleaseCondition{
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Message:            msg,
		Reason:             string(reason),
	}
	switch reason {
	case Available:
		ret.Type = releaseapi.ReleaseAvailable
	case Failure:
		ret.Type = releaseapi.ReleaseFailure
	case Creating, Updating, Rollbacking:
		ret.Type = releaseapi.ReleaseProgressing
	}
	return ret
}

// Reasons for releases
const (
	ReasonAvailable   = "Available"
	ReasonFailure     = "Failure"
	ReasonCreating    = "Creating"
	ReasonUpdating    = "Updating"
	ReasonRollbacking = "Rollbacking"
)

// ConditionAvailable returns an available condition.
func ConditionAvailable() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseAvailable,
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonAvailable,
		Message:            "",
	}
}

// ConditionFailure returns a failure condition.
func ConditionFailure(message string) releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseFailure,
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonFailure,
		Message:            message,
	}
}

// ConditionCreating returns a creating condition.
func ConditionCreating() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseProgressing,
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonCreating,
		Message:            "",
	}
}

// ConditionUpdating returns a updating condition.
func ConditionUpdating() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseProgressing,
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonUpdating,
		Message:            "",
	}
}

// ConditionRollbacking returns a rollbacking condition.
func ConditionRollbacking() releaseapi.ReleaseCondition {
	return releaseapi.ReleaseCondition{
		Type:               releaseapi.ReleaseProgressing,
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonRollbacking,
		Message:            "",
	}
}
