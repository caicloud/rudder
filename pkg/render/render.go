package render

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/hooks"
	chartapi "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/releaseutil"
	"k8s.io/helm/pkg/timeconv"
)

// Options is used to render template.
type Options struct {
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
	// Suspend is a flag of release.
	Suspend *bool
}

// Render renders template and config to resources.
type Render interface {
	// Render renders template and return a resources carrier.
	Render(options *Options) (Carrier, error)
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
func (r *render) Render(options *Options) (Carrier, error) {
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

	config, err := r.renderConfig(options)
	if err != nil {
		glog.Errorf("render release: %s 's config error: %v", options.Release, err)
		return nil, err
	}

	values, err := chartutil.ToRenderValues(chart, &chartapi.Config{Raw: config}, releaseOpts)
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

func (r *render) renderConfig(options *Options) (string, error) {
	if options.Suspend == nil || (options.Suspend != nil && !*options.Suspend) {
		return options.Config, nil
	}

	var err error
	confBytes := []byte(options.Config)
	count := 0
	cb := func(value []byte, dataType jsonparser.ValueType, offset int, _ error) {
		defer func() { count++ }()
		var typ string
		typ, err = jsonparser.GetString(value, "type")
		if err != nil {
			return
		}
		switch typ {
		case "Deployment", "StatefulSet":
			confBytes, err = jsonparser.Set(confBytes, []byte("0"), "_config", "controllers", fmt.Sprintf("[%d]", count), "controller", "replica")
			if err != nil {
				glog.Error(err)
			}
		case "CronJob":
			confBytes, err = jsonparser.Set(confBytes, []byte("true"), "_config", "controllers", fmt.Sprintf("[%d]", count), "controller", "suspend")
			if err != nil {
				glog.Error(err)
			}
		case "Job", "DaemonSet":
			glog.Warningf("controller type is: %s, release suspend flag can not work on it", typ)
		default:
			err = fmt.Errorf("illegal controller type: %s", typ)
		}
	}
	_, err = jsonparser.ArrayEach(confBytes, cb, "_config", "controllers")
	if err != nil {
		glog.Errorf("render release: %s suspend flag error: %v", options.Release, err)
		glog.Errorf("release: %s 's config: %s", options.Release, options.Config)
		return "", err
	}
	return string(confBytes), nil
}
