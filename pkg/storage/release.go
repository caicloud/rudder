package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	releasev1alpha1 "github.com/caicloud/clientset/kubernetes/typed/release/v1alpha1"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// All histories should have the two labels
	// LabelReleaseName is the name of release
	LabelReleaseName = "release.caicloud.io/name"
	// LabelReleaseVersion is the version of release history
	LabelReleaseVersion = "release.caicloud.io/version"
)

var (
	gvkRelease        = releaseapi.SchemeGroupVersion.WithKind("Release")
	gvkReleaseHistory = releaseapi.SchemeGroupVersion.WithKind("ReleaseHistory")
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
	// FlushConditions flushes conditions to running release.
	FlushConditions(condition ...releaseapi.ReleaseCondition) (*releaseapi.Release, error)
}

// ReleaseHistoryHolder contains methods for release histories
type ReleaseHistoryHolder interface {
	// History gets specified version of release.
	History(version int32) (*releaseapi.ReleaseHistory, error)
	// Histories returns all histories of release.
	Histories() ([]releaseapi.ReleaseHistory, error)
}

// NewReleaseBackendWithCacheLayer creates a release backend.
func NewReleaseBackendWithCacheLayer(client releasev1alpha1.ReleaseV1alpha1Interface, layers kube.CacheLayers) ReleaseBackend {
	return &releaseBackend{
		client: client,
		layers: layers,
	}
}

// NewReleaseBackend creates a release backend.
func NewReleaseBackend(client releasev1alpha1.ReleaseV1alpha1Interface) ReleaseBackend {
	return &releaseBackend{
		client: client,
	}
}

type releaseBackend struct {
	client releasev1alpha1.ReleaseV1alpha1Interface
	layers kube.CacheLayers
}

// ReleaseStorage returns a corresponding storage for the release.
func (rb *releaseBackend) ReleaseStorage(release *releaseapi.Release) ReleaseStorage {
	return &releaseStorage{
		name:                 release.Name,
		release:              release.DeepCopy(),
		releaseClient:        rb.client.Releases(release.Namespace),
		releaseHistoryClient: rb.client.ReleaseHistories(release.Namespace),
		layers:               rb.layers,
	}
}

type releaseStorage struct {
	name                 string
	release              *releaseapi.Release
	releaseClient        releasev1alpha1.ReleaseInterface
	releaseHistoryClient releasev1alpha1.ReleaseHistoryInterface
	layers               kube.CacheLayers
}

const (
	actionCreated = "Created"
	actionUpdated = "Updated"
	actionDeleted = "Deleted"
)

func (rs *releaseStorage) withLayer(gvk schema.GroupVersionKind, obj runtime.Object, action string) error {
	if rs.layers != nil {
		layer, err := rs.layers.LayerFor(gvk)
		if err != nil {
			return err
		}
		switch action {
		case actionCreated:
			layer.Created(obj)
		case actionUpdated:
			layer.Updated(obj)
		case actionDeleted:
			layer.Deleted(obj)
		}
	}
	return nil
}

// Release returns a cached release. It may be not a latest one.
// Don't use the release to cover running release.
func (rs *releaseStorage) Release() (*releaseapi.Release, error) {
	if rs.layers != nil {
		layer, err := rs.layers.LayerFor(gvkRelease)
		if err != nil {
			return nil, err
		}
		obj, err := layer.ByNamespace(rs.release.Namespace).Get(rs.release.Name)
		if err != nil {
			return nil, err
		}
		return obj.(*releaseapi.Release), nil
	}
	return rs.release, nil
}

// own checks if the history is belong to current release.
func (rs *releaseStorage) own(history *releaseapi.ReleaseHistory) bool {
	if len(history.OwnerReferences) != 1 {
		return false
	}
	or := history.OwnerReferences[0]
	return or.APIVersion == releaseapi.SchemeGroupVersion.String() &&
		or.Kind == gvkRelease.Kind &&
		or.Name == rs.release.Name &&
		or.UID == rs.release.UID
}

// Update updates the release.
func (rs *releaseStorage) Update(release *releaseapi.Release) (*releaseapi.Release, error) {
	history, err := rs.History(release.Status.Version)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		// Create history
		history = constructReleaseHistory(release, release.Status.Version)
		_, err := rs.releaseHistoryClient.Create(history)
		if err != nil {
			return nil, err
		}
		if err := rs.withLayer(gvkReleaseHistory, history, actionCreated); err != nil {
			return nil, err
		}
	}
	// Update release
	return rs.Patch(func(rel *releaseapi.Release) {
		rel.Status.LastUpdateTime = metav1.Now()
		rel.Status.Manifest = release.Status.Manifest
		rel.Status.Version = release.Status.Version
		rel.Status.Conditions = []releaseapi.ReleaseCondition{ConditionUpdating()}
	})
}

// Patch patches the release with a modifier.
func (rs *releaseStorage) Patch(modifier func(release *releaseapi.Release)) (*releaseapi.Release, error) {
	oldOne, err := json.Marshal(rs.release)
	if err != nil {
		return nil, err
	}
	target := rs.release.DeepCopy()
	modifier(target)
	newOne, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.CreateMergePatch(oldOne, newOne)
	if err != nil {
		return nil, err
	}
	if len(patch) == 2 && string(patch) == "{}" {
		return rs.release, nil
	}
	rel, err := rs.releaseClient.Patch(rs.name, types.MergePatchType, patch)
	if err != nil {
		return nil, err
	}
	if err := rs.withLayer(gvkRelease, rel, actionUpdated); err != nil {
		return nil, err
	}

	// Keep release status fresh
	rs.release = rel
	return rs.release, nil
}

// Rollback rollbacks running release to specified version.
func (rs *releaseStorage) Rollback(version int32) (*releaseapi.Release, error) {
	history, err := rs.History(version)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	if err != nil {
		// Record condition.
		return rs.FlushConditions(ConditionFailure(err.Error()))
	}
	// FIX: use temporary render to avoid concurrent issue
	// need render again instead of using history's manifest directly because of the history's manifest
	// remained suspend status when be generated.
	carrier, err := render.NewRender().Render(&render.RenderOptions{
		Namespace: rs.release.Namespace,
		Release:   rs.release.Name,
		Version:   history.Spec.Version,
		Template:  history.Spec.Template,
		Config:    history.Spec.Config,
		Suspend:   rs.release.Spec.Suspend,
	})
	if err != nil {
		return nil, err
	}
	manifests := carrier.Resources()
	return rs.Patch(func(release *releaseapi.Release) {
		release.Spec.Description = history.Spec.Description
		release.Spec.Template = history.Spec.Template
		release.Spec.Config = history.Spec.Config
		release.Spec.RollbackTo = nil
		release.Status.Version = history.Spec.Version
		release.Status.LastUpdateTime = metav1.Now()
		release.Status.Manifest = render.MergeResources(manifests)
		release.Status.Conditions = []releaseapi.ReleaseCondition{ConditionRollbacking()}
	})
}

// Delete deletes the release.
func (rs *releaseStorage) Delete() error {
	err := rs.releaseClient.Delete(rs.name, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err := rs.withLayer(gvkRelease, rs.release, actionDeleted); err != nil {
		return err
	}

	// Don't need to record deleted histories.
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
func (rs *releaseStorage) History(version int32) (history *releaseapi.ReleaseHistory, err error) {
	if rs.layers != nil {
		layer, err := rs.layers.LayerFor(gvkReleaseHistory)
		if err != nil {
			return nil, err
		}
		obj, err := layer.ByNamespace(rs.release.Namespace).Get(generateReleaseHistoryName(rs.name, version))
		if err != nil {
			return nil, err
		}
		history = obj.(*releaseapi.ReleaseHistory)
	} else {
		history, err = rs.releaseHistoryClient.Get(generateReleaseHistoryName(rs.name, version), metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}
	if !rs.own(history) {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group: gvkReleaseHistory.Group,
		}, history.Name)
	}
	return history, nil
}

// Histories returns all histories of release.
func (rs *releaseStorage) Histories() (results []releaseapi.ReleaseHistory, err error) {
	if rs.layers != nil {
		layer, err := rs.layers.LayerFor(gvkReleaseHistory)
		if err != nil {
			return nil, err
		}
		list, err := layer.ByNamespace(rs.release.Namespace).List(labels.Set{
			LabelReleaseName: rs.name,
		}.AsSelector())
		if err != nil {
			return nil, err
		}
		results = make([]releaseapi.ReleaseHistory, 0, len(list))
		for _, obj := range list {
			results = append(results, *obj.(*releaseapi.ReleaseHistory))
		}
	} else {
		histories, err := rs.releaseHistoryClient.List(metav1.ListOptions{
			LabelSelector: labels.Set{
				LabelReleaseName: rs.name,
			}.String(),
		})
		if err != nil {
			return nil, err
		}
		results = histories.Items
	}
	count := 0
	for _, history := range results {
		if rs.own(&history) {
			results[count] = history
			count++
		}
	}
	results = results[:count]
	sort.Slice(results, func(i, j int) bool {
		return results[i].Spec.Version > results[j].Spec.Version
	})
	return results, nil
}

// UpdateStatus update the status of running release.
func (rs *releaseStorage) UpdateStatus(modifier func(status *releaseapi.ReleaseStatus)) (*releaseapi.Release, error) {
	return rs.Patch(func(release *releaseapi.Release) {
		modifier(&release.Status)
	})
}

// AddCondition adds a condition to running release. Deprecated, Use FlushConditions as instead.
func (rs *releaseStorage) AddCondition(condition releaseapi.ReleaseCondition) (*releaseapi.Release, error) {
	return rs.Patch(func(release *releaseapi.Release) {
		release.Status.Conditions = append(release.Status.Conditions, condition)
	})
}

// FlushConditions flushes conditions to running release.
func (rs *releaseStorage) FlushConditions(conditions ...releaseapi.ReleaseCondition) (*releaseapi.Release, error) {
	return rs.Patch(func(release *releaseapi.Release) {
		release.Status.Conditions = conditions
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
			Annotations: release.Annotations,
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
