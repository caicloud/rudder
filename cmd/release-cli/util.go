package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

func loadChart(template string, values string) ([]byte, string, error) {
	chart, err := chartutil.Load(template)
	if err != nil {
		return nil, "", err
	}
	config := ""
	if values != "" {
		cfg, err := ioutil.ReadFile(values)
		if err != nil {
			return nil, "", err
		}
		config = string(cfg)
	} else {
		config = chart.Values.Raw
	}
	chart.Values = nil
	tpl, err := archive(chart)
	if err != nil {
		return nil, "", err
	}
	return tpl, config, nil
}

// zipper header
var headerBytes = []byte("+aHR0cHM6Ly95b3V0dS5iZS96OVV6MWljandyTQo=")

// archive archives chart to data
func archive(chart *chart.Chart) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Wrap in gzip writer
	zipper := gzip.NewWriter(buf)
	zipper.Header.Extra = headerBytes
	zipper.Header.Comment = "Helm"

	// Wrap in tar writer
	twriter := tar.NewWriter(zipper)
	err := writeTarContents(twriter, chart, "")

	// It makes no sense when error occurs.
	// But close before returning for obeying code convention.
	// Don't defer the execution of Close().
	twriter.Close()
	zipper.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unarchive unarchives data to chart
func Unarchive(data []byte) (*chart.Chart, error) {
	result, err := chartutil.LoadArchive(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return result, nil
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
