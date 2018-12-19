package main

import (
	"fmt"

	"github.com/caicloud/clientset/kubernetes"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	root.AddCommand(delete)
	fs := delete.Flags()

	fs.StringVarP(&deleteOptions.Server, "server", "s", "", "Kubenetes master host")
	fs.StringVarP(&deleteOptions.BearerToken, "bearer-token", "b", "", "Kubenetes master bearer token")
	fs.StringVarP(&deleteOptions.Namespace, "namespace", "n", "", "Kubenetes namespace")
	fs.StringVarP(&deleteOptions.KubeconfigPath, "kubeconfig", "k", "", "Kubernetes config path")
}

var deleteOptions = struct {
	Server         string
	BearerToken    string
	KubeconfigPath string
	Namespace      string
}{}

var delete = &cobra.Command{
	Use:   "delete",
	Short: "Delete a release from kubernetes cluster",
	Run:   runDelete,
}

func runDelete(cmd *cobra.Command, args []string) {
	if deleteOptions.Server == "" {
		glog.Fatalln("--server must be set")
	}

	if deleteOptions.BearerToken == "" && deleteOptions.KubeconfigPath == "" {
		glog.Fatalln("Must specify either --bearer-token or --kubeconfig")
	}

	if deleteOptions.Namespace == "" {
		glog.Fatalln("--namespace must be set")
	}

	if len(args) <= 0 {
		glog.Fatalln("Must specify release name")
	}

	var clientset *kubernetes.Clientset
	var err error
	if deleteOptions.KubeconfigPath != "" {
		cfg, err := clientcmd.BuildConfigFromFlags(deleteOptions.Server, deleteOptions.KubeconfigPath)
		if err != nil {
			glog.Fatalln("Unable to build k8s Config: %v", err)
		}

		clientset, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			glog.Fatalln(err)
		}
	} else {
		clientset, err = kubernetes.NewForConfig(&rest.Config{
			Host:        deleteOptions.Server,
			BearerToken: deleteOptions.BearerToken,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		})
		if err != nil {
			glog.Fatalln(err)
		}
	}

	for _, name := range args {
		err := clientset.ReleaseV1alpha1().Releases(deleteOptions.Namespace).Delete(name, nil)
		if err != nil {
			glog.Fatalln(err)
		}
		fmt.Printf("%s deleted\n", name)
	}
}
