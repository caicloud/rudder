package storage

import (
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type releaseConditionType string

const (
	Available   releaseConditionType = "Available"
	Failure     releaseConditionType = "Failure"
	Creating    releaseConditionType = "Creating"
	Updating    releaseConditionType = "Updating"
	Rollbacking releaseConditionType = "Rollbacking"
)

// Condition returns a release condition based on given release status.
func Condition(typ releaseConditionType, msg string) releaseapi.ReleaseCondition {
	ret := releaseapi.ReleaseCondition{
		Status:             core.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Message:            msg,
		Reason:             string(typ),
	}
	switch typ {
	case Available:
		ret.Type = releaseapi.ReleaseAvailable
	case Failure:
		ret.Type = releaseapi.ReleaseFailure
	case Creating, Updating, Rollbacking:
		ret.Type = releaseapi.ReleaseProgressing
	}
	return ret
}
