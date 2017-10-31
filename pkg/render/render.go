package render

import (
	"bytes"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/hooks"
	chartapi "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/releaseutil"
	"k8s.io/helm/pkg/timeconv"
)

// RenderOptions is used to render template.
type RenderOptions struct {
	// Namespace for resources.
	Namespace string
	// Release is the name of release.
	Release string
	// Version is the version of release.
	Version int32
	// Template is a binary data of template.
	Template []byte
	// Config is a json config to render template.
	Config string
}

// Render renders template and config to resources.
type Render interface {
	// Render renders template and return a resources carrier.
	Render(options *RenderOptions) (Carrier, error)
}

// NewRender creates a template render.
func NewRender() Render {
	return &render{
		engine: engine.New(),
	}
}

type render struct {
	engine *engine.Engine
}

// Render renders release and return a resources carrier.
func (r *render) Render(options *RenderOptions) (Carrier, error) {
	chart, err := chartutil.LoadArchive(bytes.NewReader(options.Template))
	if err != nil {
		return nil, err
	}
	releaseOpts := chartutil.ReleaseOptions{
		Name:      options.Release,
		Time:      timeconv.Now(),
		Namespace: options.Namespace,
		Revision:  int(options.Version),
		IsInstall: true,
	}
	values, err := chartutil.ToRenderValues(chart, &chartapi.Config{Raw: options.Config}, releaseOpts)
	if err != nil {
		return nil, err
	}
	resources, err := r.renderResources(chart, values)
	if err != nil {
		return nil, err
	}
	return treeCarrierFor(resources)
}

// renderResources renders chart to a list of resources
func (r *render) renderResources(chart *chartapi.Chart, values chartutil.Values) (map[string][]string, error) {
	files, err := r.engine.Render(chart, values)
	if err != nil {
		return nil, err
	}
	// result is a file-resources map
	result := make(map[string][]string)

	// Remove unused files:
	// * Files which name has suffix "NOTES.txt"
	// * Files which name starts with "_"
	// * Hooks
	// These files are defined by helm, we don't need them.
	for k, v := range files {
		base := path.Base(k)
		if strings.HasPrefix(base, "_") || strings.HasSuffix(base, "NOTES.txt") {
			continue
		}

		// Parse files
		resources := releaseutil.SplitManifests(v)
		if len(resources) <= 0 {
			continue
		}
		validRes := make([]string, 0, len(resources))
		for _, res := range resources {
			ok, err := r.isHook(res)
			if err != nil {
				return nil, err
			}
			if ok {
				continue
			}
			// Add resources to result map
			validRes = append(validRes, res)
		}
		result[k] = validRes
	}
	return result, nil
}

// isHook checks whether the resource is a helm hook
func (r *render) isHook(resource string) (bool, error) {
	head := &releaseutil.SimpleHead{}
	err := yaml.Unmarshal([]byte(resource), &head)
	if err != nil {
		return false, err
	}
	if head.Metadata == nil || head.Metadata.Annotations == nil {
		return false, nil
	}
	_, ok := head.Metadata.Annotations[hooks.HookAnno]
	return ok, nil
}
