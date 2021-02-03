package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const (
	DefaultTemplateType    = "template.caicloud.io/application"
	DefaultTemplateVersion = "1.0.0"
)

var headerBytes = []byte("+aHR0cHM6Ly95b3V0dS5iZS96OVV6MWljandyTQo=")

type Raw struct {
	Data []byte
}

func (c *Raw) MarshalJSON() []byte {
	return c.Data
}

func (c *Raw) UnmarshalJSON(data []byte) {
	c.Data = data
}

type Template struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

type Metadata struct {
	Name        string   `json:"name,omitempty"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Template    Template `json:"template,omitempty"`
}

type Config struct {
	Metadata    *Metadata `json:"_metadata,omitempty"`
	Controllers *Raw      `json:"controllers,omitempty"`
}

type Values struct {
	Config       *Config
	Subordinates map[string]*Values
}

func (t *Values) MarshalJSON() ([]byte, error) {
	result := map[string]interface{}{
		"_config": t.Config,
	}
	for k, v := range t.Subordinates {
		result[k] = v
	}
	return json.Marshal(result)
}

func (t *Values) UnmarshalJSON(data []byte) error {
	result := map[string]*Raw{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return err
	}
	config, ok := result["_config"]
	if !ok {
		return fmt.Errorf("can't resolve data to config")
	}
	err = json.Unmarshal(config.Data, &t.Config)
	if err != nil {
		return err
	}
	t.Subordinates = make(map[string]*Values)
	for k, v := range result {
		// ignore all fields starts with "_"
		if strings.HasPrefix(k, "_") {
			continue
		}
		values := &Values{}
		err = json.Unmarshal(v.Data, values)
		if err != nil {
			return err
		}
		t.Subordinates[k] = values
	}
	return nil
}

func copyChart(origin *chart.Chart) *chart.Chart {
	newChart := &chart.Chart{
		Metadata: &chart.Metadata{},
	}
	*newChart.Metadata = *origin.Metadata
	newChart.Dependencies = origin.Dependencies
	newChart.Templates = origin.Templates
	newChart.Files = origin.Files
	return newChart
}

func generateChart(values *Values, basicChart *chart.Chart) *chart.Chart {
	newChart := copyChart(basicChart)
	newChart.Metadata.Name = values.Config.Metadata.Name
	newChart.Metadata.Version = values.Config.Metadata.Version
	newChart.Metadata.Description = values.Config.Metadata.Description
	newChart.Dependencies = make([]*chart.Chart, 0, len(values.Subordinates))
	for _, v := range values.Subordinates {
		subchart := generateChart(v, basicChart)
		newChart.Dependencies = append(newChart.Dependencies, subchart)
	}
	return newChart
}

// ChartPath returns basic chart path
func ChartPath() string {
	path := os.Getenv("CHART_TEMPLATE_PATH")
	if path == "" {
		path = "./templates"
	}
	return path
}

// LoadBasicChart loads chart from template
func LoadBasicChart(typ, templateVersion string) (*chart.Chart, error) {
	if typ != "" && typ != DefaultTemplateType {
		return nil, fmt.Errorf("nil typ %s %s", typ, templateVersion)
	}
	if templateVersion == "" {
		templateVersion = DefaultTemplateVersion
	}
	result, err := chartutil.LoadDir(ChartPath())
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GenerateChart(values string) (*chart.Chart, error) {
	target := &Values{}
	err := json.Unmarshal([]byte(values), target)
	if err != nil {
		return nil, err
	}
	md := target.Config.Metadata
	if md.Template.Type == "" {
		md.Template.Type = DefaultTemplateType
	}
	if md.Template.Version == "" {
		md.Template.Version = DefaultTemplateVersion
	}
	// TODO(kdada): load to a global variable.
	basicChart, err := LoadBasicChart(md.Template.Type, md.Template.Version)
	if err != nil {
		return nil, err
	}
	result := generateChart(target, basicChart)
	result.Values = &chart.Config{
		Raw: values,
	}
	return result, nil
}

// Archive archives chart to data
func Archive(chart *chart.Chart) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Wrap in gzip writer
	zipper := gzip.NewWriter(buf)
	zipper.Header.Extra = headerBytes
	zipper.Header.Comment = "Helm"

	// Wrap in tar writer
	writer := tar.NewWriter(zipper)
	err := writeTarContents(writer, chart, "")

	// It makes no sense when error occurs.
	// But close before returning for obeying code convention.
	// Don't defer the execution of Close().
	_ = writer.Close()
	_ = zipper.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeTarContents writes a chart to tar package
// Copy from: k8s.io/helm/pkg/chartutil/save.go
func writeTarContents(out *tar.Writer, c *chart.Chart, prefix string) error {
	base := filepath.Join(prefix, c.Metadata.Name)

	// Save Chart.yaml
	cdata, err := yaml.Marshal(c.Metadata)
	if err != nil {
		return err
	}
	if err := writeToTar(out, base+"/Chart.yaml", cdata); err != nil {
		return err
	}

	// Save values.yaml
	if c.Values != nil && len(c.Values.Raw) > 0 {
		if err := writeToTar(out, base+"/values.yaml", []byte(c.Values.Raw)); err != nil {
			return err
		}
	}

	// Save templates
	for _, f := range c.Templates {
		n := filepath.Join(base, f.Name)
		if err := writeToTar(out, n, f.Data); err != nil {
			return err
		}
	}

	// Save files
	for _, f := range c.Files {
		n := filepath.Join(base, f.TypeUrl)
		if err := writeToTar(out, n, f.Value); err != nil {
			return err
		}
	}

	// Save dependencies
	for _, dep := range c.Dependencies {
		if err := writeTarContents(out, dep, base+"/charts"); err != nil {
			return err
		}
	}
	return nil
}

// writeToTar writes a single file to a tar archive.
// Copy from: k8s.io/helm/pkg/chartutil/save.go
func writeToTar(out *tar.Writer, name string, body []byte) error {
	// TODO: Do we need to create dummy parent directory names if none exist?
	h := &tar.Header{
		Name: name,
		Mode: 0755,
		Size: int64(len(body)),
	}
	if err := out.WriteHeader(h); err != nil {
		return err
	}
	if _, err := out.Write(body); err != nil {
		return err
	}
	return nil
}
