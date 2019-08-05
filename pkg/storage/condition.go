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
