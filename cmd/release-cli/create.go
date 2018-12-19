package main

import (
	"github.com/caicloud/clientset/kubernetes"
	"github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	root.AddCommand(create)
	fs := create.Flags()

	fs.StringVarP(&createOptions.Server, "server", "s", "", "Kubernetes master host")
	fs.StringVarP(&createOptions.BearerToken, "bearer-token", "b", "", "Kubernetes master bearer token")
	fs.StringVarP(&createOptions.Namespace, "namespace", "n", "", "Kubernetes namespace")
	fs.StringVarP(&createOptions.Values, "values", "c", "", "Chart values file path. Override values.yaml in template")
	fs.StringVarP(&createOptions.Template, "template", "t", "", "Chart template file path. Can be a tgz package or a chart directory")
	fs.StringVarP(&createOptions.KubeconfigPath, "kubeconfig", "k", "", "Kubernetes config path")
}

var createOptions = struct {
	Server         string
	BearerToken    string
	KubeconfigPath string
	Namespace      string
	Values         string
	Template       string
}{}

var create = &cobra.Command{
	Use:   "create",
	Short: "Create a release into kubernetes cluster",
	Run:   runCreate,
}

func runCreate(cmd *cobra.Command, args []string) {
	if createOptions.Server == "" {
		glog.Fatalln("--server must be set")
	}

	if createOptions.BearerToken == "" && createOptions.KubeconfigPath == "" {
		glog.Fatalln("Must specify either --bearer-token or --kubeconfig")
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

	var clientset *kubernetes.Clientset
	if createOptions.KubeconfigPath != "" {
		cfg, err := clientcmd.BuildConfigFromFlags(createOptions.Server, createOptions.KubeconfigPath)
		if err != nil {
			glog.Fatalln("Unable to build k8s Config: %v", err)
		}

		clientset, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			glog.Fatalln(err)
		}
	} else {
		clientset, err = kubernetes.NewForConfig(&rest.Config{
			Host:        createOptions.Server,
			BearerToken: createOptions.BearerToken,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		})
		if err != nil {
			glog.Fatalln(err)
		}
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
