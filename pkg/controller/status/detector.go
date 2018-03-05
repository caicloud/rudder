package status

import (
	"context"
	"sync"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"
	"github.com/caicloud/rudder/pkg/status"
	"github.com/caicloud/rudder/pkg/store"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
)

// Detector detects a release and find out some status for it.
type Detector interface {
	// Kind returns a unique id of the detector.
	Kind() string
	// Detect detects the status of release. Modify release derectly and return the same
	// instance.
	Detect(store store.IntegrationStore, release *releaseapi.Release) (map[string]releaseapi.ReleaseDetailStatus, error)
}

// ResourceDetector detects resources status of release.
type ResourceDetector struct {
	codec  kube.Codec
	umpire status.Umpire
}

// NewResourceDetector creates a NewResourceDetector.
func NewResourceDetector(
	codec kube.Codec,
	umpire status.Umpire,
) *ResourceDetector {
	return &ResourceDetector{
		codec:  codec,
		umpire: umpire,
	}
}

// Kind returns a unique id of the detector.
func (rd *ResourceDetector) Kind() string {
	// The resource detector have a kind named empty.
	return ""
}

// Detect detects the resources status of release.
func (rd *ResourceDetector) Detect(store store.IntegrationStore, release *releaseapi.Release) (map[string]releaseapi.ReleaseDetailStatus, error) {
	if release.Status.Manifest == "" {
		// No resource
		return map[string]releaseapi.ReleaseDetailStatus{}, nil
	}
	carrier, err := render.CarrierForManifest(release.Status.Manifest)
	if err != nil {
		glog.Errorf("Can't parse manifest for %s/%s: %v", release.Namespace, release.Name, err)
		return nil, err
	}
	var lock sync.Mutex
	details := make(map[string]releaseapi.ReleaseDetailStatus)
	err = carrier.Run(context.Background(), render.PositiveOrder, func(ctx context.Context, node string, resources []string) error {
		detail := releaseapi.ReleaseDetailStatus{
			Path:      node,
			Resources: make(map[string]releaseapi.ResourceCounter),
		}
		for _, resource := range resources {
			obj, accessor, err := rd.codec.AccessorForResource(resource)
			if err != nil {
				return err
			}
			gvk := obj.GetObjectKind().GroupVersionKind()
			informer, err := store.InformerFor(gvk)
			if err != nil {
				glog.Errorf("Can't get informer for %s: %v", node, err)
				return err
			}

			runningObj, err := informer.Lister().ByNamespace(release.Namespace).Get(accessor.GetName())
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
			sentence := status.Progressing
			if err == nil {
				// There is no gvk in runningObj. We set it here.
				runningObj.GetObjectKind().SetGroupVersionKind(gvk)
				sentence, err = rd.umpire.Judge(runningObj)
				if err != nil {
					// Log the error and mark as Failure
					glog.Errorf("Can't decode resource for %s: %v", node, err)
					sentence = status.Failure
				}
			}
			key := gvk.String()
			counter := detail.Resources[key]
			switch sentence {
			case status.Available:
				counter.Available++
			case status.Progressing:
				counter.Progressing++
			default:
				// All other kinds are failures
				counter.Failure++
			}
			detail.Resources[key] = counter
		}
		lock.Lock()
		defer lock.Unlock()
		details[node] = detail
		return nil
	})
	if err != nil {
		glog.Errorf("Can't run resources for %s/%s: %v", release.Namespace, release.Name, err)
		return nil, err
	}
	return details, nil
}
