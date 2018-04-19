package main

import (
	"github.com/caicloud/clientset/kubernetes"
	"github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
)

func init() {
	root.AddCommand(create)
	fs := create.Flags()

	fs.StringVarP(&createOptions.Server, "server", "s", "", "Kubenetes master host")
	fs.StringVarP(&createOptions.BearerToken, "bearer-token", "b", "", "Kubenetes master bearer token")
	fs.StringVarP(&createOptions.Namespace, "namespace", "n", "", "Kubenetes namespace")
	fs.StringVarP(&createOptions.Values, "values", "c", "", "Chart values file path. Override values.yaml in template")
	fs.StringVarP(&createOptions.Template, "template", "t", "", "Chart template file path. Can be a tgz package or a chart directory")
}

var createOptions = struct {
	Server      string
	BearerToken string
	Namespace   string
	Values      string
	Template    string
}{}

var create = &cobra.Command{
	Use:   "create",
	Short: "Create a release into kubernetes cluster",
	Run:   runCreate,
}

func runCreate(cmd *cobra.Command, args []string) {
	if createOptions.Server == "" || createOptions.BearerToken == "" {
		glog.Fatalln("--server and --bearer-token must be set")
	}

	if createOptions.Namespace == "" {
		glog.Fatalln("--namespace must be set")
	}

	if createOptions.Template == "" {
		glog.Fatalln("--template must be set")
	}

	if len(args) <= 0 {
		glog.Fatalln("Must specify release name")
	}
	if len(args) > 1 {
		glog.Fatalln("Two or more release names is not allowed")
	}

	template, config, err := loadChart(createOptions.Template, createOptions.Values)
	if err != nil {
		glog.Fatalf("Unable to load template and values: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host:        createOptions.Server,
		BearerToken: createOptions.BearerToken,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
	if err != nil {
		glog.Fatalln(err)
	}
	rel := &v1alpha1.Release{}
	rel.Name = args[0]
	rel.Spec.Config = config
	rel.Spec.Template = template
	r, err := clientset.ReleaseV1alpha1().Releases(createOptions.Namespace).Create(rel)
	if err != nil {
		glog.Fatalln(err)
	}
	meta := [][]string{
		{"Name:", r.Name},
		{"Namespace:", r.Namespace},
		{"Start Time:", r.CreationTimestamp.String()},
	}
	printTable(meta)
}
