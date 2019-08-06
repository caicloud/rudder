package main

import (
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/caicloud/rudder/pkg/render"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func init() {
	root.AddCommand(lint)
	fs := lint.Flags()

	fs.StringVarP(&lintOptions.Values, "values", "c", "", "Chart values file path. Override values.yaml in template")
	fs.StringVarP(&lintOptions.Standard, "standard", "s", "", "Standard Chart values file path")
	fs.StringVarP(&lintOptions.Template, "template", "t", "", "Chart template file path. Can be a tgz package or a chart directory")
	fs.BoolVarP(&lintOptions.Detail, "detail", "d", false, "Show details")
}

var lintOptions = struct {
	Values   string
	Template string
	Detail   bool
	Standard string
}{}

var lint = &cobra.Command{
	Use:   "lint",
	Short: "Lint checks if a chart is right",
	Run:   runLint,
}

func runLint(cmd *cobra.Command, args []string) {
	if lintOptions.Template == "" {
		glog.Fatalln("--template must be set")
	}
	tpl, values, err := loadChart(lintOptions.Template, lintOptions.Values)
	if err != nil {
		glog.Fatalf("Unable to load template and values: %v", err)
	}

	if lintOptions.Standard != "" {
		standardValues, err := ioutil.ReadFile(lintOptions.Standard)
		if err != nil {
			glog.Fatalf("Unable to load standard values: %v", err)
		}
		validateValues(string(standardValues), values)
	}

	r := render.NewRender()
	c, err := r.Render(&render.RenderOptions{
		Namespace: "default",
		Release:   "release-name",
		Version:   1,
		Config:    values,
		Template:  tpl,
	})

	if err != nil {
		glog.Fatalln(err)
	}
	if lintOptions.Detail {
		fmt.Println(render.MergeResources(c.Resources()))
	}
}

func validateValues(standard, values string) {
	standardMap, err := valuesPath(standard)
	if err != nil {
		glog.Error(err)
	}
	targetMap, err := valuesPath(values)
	if err != nil {
		glog.Error(err)
	}
	for path, types := range targetMap {
		st := standardMap[path]
		if len(st) <= 0 {
			glog.Warningf("%s is not in standard", path)
			continue
		}
		for _, t := range types {
			valid := false
			for _, s := range st {
				if t == s {
					valid = true
					break
				}
			}
			if !valid {
				glog.Errorf("%s has wrong type %s, must be in %v", path, t, st)
			}
		}
	}
}

func valuesPath(values string) (map[string][]string, error) {
	o := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(values), &o)
	if err != nil {
		return nil, err
	}
	types := walkthrough("", o)
	result := map[string][]string{}
	for k, ts := range types {
		arr := result[k]
		for t, _ := range ts {
			arr = append(arr, t)
		}
		result[k] = arr
	}
	return result, nil
}

func walkthrough(prefix string, obj interface{}) map[string]map[string]bool {
	if obj == nil {
		return nil
	}
	result := map[string]map[string]bool{}
	switch target := obj.(type) {
	case map[string]interface{}:
		for k, v := range target {
			r := walkthrough(prefix+"."+k, v)
			mergeMap(result, r)
		}
	case []interface{}:
		for _, o := range target {
			r := walkthrough(prefix+"[]", o)
			mergeMap(result, r)
		}
	case string:
		result[prefix] = map[string]bool{reflect.String.String(): true}
	case bool:
		result[prefix] = map[string]bool{reflect.Bool.String(): true}
	case float64:
		result[prefix] = map[string]bool{reflect.Float64.String(): true}
	default:
		// Unreachable
		glog.Fatalf("Unknown object type: %s", reflect.TypeOf(obj).String())
	}
	return result
}

func mergeMap(dst, src map[string]map[string]bool) {
	for k, v := range src {
		types := dst[k]
		if types == nil {
			types = v
		} else {
			for t, _ := range v {
				types[t] = true
			}
		}
		dst[k] = types
	}
}
