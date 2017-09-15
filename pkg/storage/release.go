package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	releasev1alpha1 "github.com/caicloud/clientset/kubernetes/typed/release/v1alpha1"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// All histories should have the two labels
	// LabelReleaseName is the name of release
	LabelReleaseName = "release.caicloud.io/name"
	// LabelReleaseVersion is the version of release history
	LabelReleaseVersion = "release.caicloud.io/version"
)

// ReleaseBackend is a backend for releases and release histories.
type ReleaseBackend interface {
	// ReleaseStorage returns a corresponding storage for the release.
	ReleaseStorage(release *releaseapi.Release) ReleaseStorage
}

// ReleaseStorage is a storage for a release.
type ReleaseStorage interface {
	ReleaseHolder
	ReleaseHolderExpansion
	ReleaseHistoryHolder
}

// ReleaseHolder contains a bundle of methods for manipulating release.
type ReleaseHolder interface {
	// Release returns a cached release. It may be not a latest one.
	// Don't use the release to cover running release.
	Release() (*releaseapi.Release, error)
	// Update updates the release.
	Update(release *releaseapi.Release) (*releaseapi.Release, error)
	// Patch patches the release with a modifier.
	Patch(modifier func(release *releaseapi.Release)) (*releaseapi.Release, error)
	// Rollback rollbacks running release to specified version.
	Rollback(version int32) (*releaseapi.Release, error)
	// Delete deletes the release.
	Delete() error
}

// ReleaseHolderExpansion extends the methods of release.
type ReleaseHolderExpansion interface {
	// UpdateStatus update the status of running release.
	UpdateStatus(modifier func(status *releaseapi.ReleaseStatus)) (*releaseapi.Release, error)
	// AddCondition adds a condition to running release.
	AddCondition(condition releaseapi.ReleaseCondition) (*releaseapi.Release, error)
}

// ReleaseHistoryHolder contains methods for release histories
type ReleaseHistoryHolder interface {
	// History gets specified version of release.
	History(version int32) (*releaseapi.ReleaseHistory, error)
	// Histories returns all histories of release.
	Histories() ([]releaseapi.ReleaseHistory, error)
}

// NewReleaseBackend creates a release backend.
func NewReleaseBackend(client releasev1alpha1.ReleaseV1alpha1Interface) ReleaseBackend {
	return &releaseBackend{
		client: client,
	}
}

type releaseBackend struct {
	client releasev1alpha1.ReleaseV1alpha1Interface
}

// ReleaseStorage returns a corresponding storage for the release.
func (rb *releaseBackend) ReleaseStorage(release *releaseapi.Release) ReleaseStorage {
	return &releaseStorage{
		name:                 release.Name,
		release:              release,
		releaseClient:        rb.client.Releases(release.Namespace),
		releaseHistoryClient: rb.client.ReleaseHistories(release.Namespace),
	}
}

type releaseStorage struct {
	name                 string
	release              *releaseapi.Release
	releaseClient        releasev1alpha1.ReleaseInterface
	releaseHistoryClient releasev1alpha1.ReleaseHistoryInterface
}

// Release returns a cached release. It may be not a latest one.
// Don't use the release to cover running release.
func (rs *releaseStorage) Release() (*releaseapi.Release, error) {
	return rs.release, nil
}

// Update updates the release.
func (rs *releaseStorage) Update(release *releaseapi.Release) (*releaseapi.Release, error) {
	version := int32(1)
	if release.Status.Version > 0 {
		histories, err := rs.Histories()
		if err != nil {
			return nil, err
		}
		if len(histories) <= 0 {
			version = 1
		} else {
			version = histories[0].Spec.Version + 1
		}
	}
	// Create history
	history := constructReleaseHistory(release, version)
	_, err := rs.releaseHistoryClient.Create(history)
	if err != nil {
		return nil, err
	}

	// Update release
	return rs.Patch(func(rel *releaseapi.Release) {
		rel.Status.LastUpdateTime = metav1.Now()
		rel.Status.Manifest = release.Status.Manifest
		rel.Status.Version = version
		rel.Status.Conditions = append(rel.Status.Conditions, ConditionAvailable())
	})
}

// Patch patches the release with a modifier.
func (rs *releaseStorage) Patch(modifier func(release *releaseapi.Release)) (*releaseapi.Release, error) {
	oldOne, err := json.Marshal(rs.release)
	if err != nil {
		return nil, err
	}
	modifier(rs.release)
	shortenConditions(rs.release)
	newOne, err := json.Marshal(rs.release)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.CreateMergePatch(oldOne, newOne)
	if err != nil {
		return nil, err
	}
	rel, err := rs.releaseClient.Patch(rs.name, types.MergePatchType, patch)
	if err != nil {
		return nil, err
	}
	// Keep release status fresh
	rs.release = rel
	return rs.release, nil
}

// Rollback rollbacks running release to specified version.
func (rs *releaseStorage) Rollback(version int32) (*releaseapi.Release, error) {
	history, err := rs.History(version)
	if err != nil {
		return nil, err
	}
	return rs.Patch(func(release *releaseapi.Release) {
		release.Spec.Description = history.Spec.Description
		release.Spec.Template = history.Spec.Template
		release.Spec.Config = history.Spec.Config
		release.Spec.RollbackTo = nil
		release.Status.Version = history.Spec.Version
		release.Status.LastUpdateTime = metav1.Now()
		release.Status.Manifest = history.Spec.Manifest
		release.Status.Conditions = append(release.Status.Conditions, ConditionAvailable())
	})
}

// Delete deletes the release.
func (rs *releaseStorage) Delete() error {
	err := rs.releaseClient.Delete(rs.name, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = rs.releaseHistoryClient.DeleteCollection(nil, metav1.ListOptions{
		LabelSelector: labels.Set{
			LabelReleaseName: rs.name,
		}.String(),
	})
	if err != nil {
		return err
	}
	return nil
}

// History gets specified version of release.
func (rs *releaseStorage) History(version int32) (*releaseapi.ReleaseHistory, error) {
	return rs.releaseHistoryClient.Get(generateReleaseHistoryName(rs.name, version), metav1.GetOptions{})
}

// Histories returns all histories of release.
func (rs *releaseStorage) Histories() ([]releaseapi.ReleaseHistory, error) {
	histories, err := rs.releaseHistoryClient.List(metav1.ListOptions{
		LabelSelector: labels.Set{
			LabelReleaseName: rs.name,
		}.String(),
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(histories.Items, func(i, j int) bool {
		return histories.Items[i].Spec.Version > histories.Items[j].Spec.Version
	})
	return histories.Items, nil
}

// UpdateStatus update the status of running release.
func (rs *releaseStorage) UpdateStatus(modifier func(status *releaseapi.ReleaseStatus)) (*releaseapi.Release, error) {
	return rs.Patch(func(release *releaseapi.Release) {
		modifier(&release.Status)
	})
}

// AddCondition adds a condition to running release.
func (rs *releaseStorage) AddCondition(condition releaseapi.ReleaseCondition) (*releaseapi.Release, error) {
	return rs.Patch(func(release *releaseapi.Release) {
		release.Status.Conditions = append(release.Status.Conditions, condition)
	})
}

// shortenConditions limits the length of conditions.
func shortenConditions(release *releaseapi.Release) {
	const maxLength = 5
	length := len(release.Status.Conditions)
	if length > maxLength {
		release.Status.Conditions = release.Status.Conditions[length-maxLength:]
	}
}

// generateReleaseHistoryName generates the name of release history.
func generateReleaseHistoryName(name string, version int32) string {
	return fmt.Sprintf("%s-v%d", name, version)
}

// constructReleaseHistory generates a release history for a release.
func constructReleaseHistory(release *releaseapi.Release, version int32) *releaseapi.ReleaseHistory {
	// Create History
	return &releaseapi.ReleaseHistory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateReleaseHistoryName(release.Name, version),
			Namespace: release.Namespace,
			Labels: map[string]string{
				LabelReleaseName:    release.Name,
				LabelReleaseVersion: strconv.Itoa(int(version)),
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: releaseapi.SchemeGroupVersion.String(),
				Kind:       "Release",
				Name:       release.Name,
				UID:        release.UID,
			}},
		},
		Spec: releaseapi.ReleaseHistorySpec{
			Description: release.Spec.Description,
			Version:     version,
			Template:    release.Spec.Template,
			Config:      release.Spec.Config,
			Manifest:    release.Status.Manifest,
		},
	}
}
