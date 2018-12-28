package main

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func init() {
	root.AddCommand(delete)
	fs := delete.Flags()

	fs.StringVarP(&deleteOptions.Server, "server", "s", "", "Kubernetes master host")
	fs.StringVarP(&deleteOptions.BearerToken, "bearer-token", "b", "", "Kubernetes master bearer token")
	fs.StringVarP(&deleteOptions.Namespace, "namespace", "n", "", "Kubernetes namespace")
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
	if deleteOptions.KubeconfigPath == "" && (deleteOptions.Server == "" || deleteOptions.BearerToken == "") {
		glog.Fatalln("Must specify either --kubeconfig or --bearer-token and --server")
	}

	if deleteOptions.Namespace == "" {
		glog.Fatalln("--namespace must be set")
	}

	if len(args) <= 0 {
		glog.Fatalln("Must specify release name")
	}

	clientset, err := newClientSet(deleteOptions.KubeconfigPath, deleteOptions.Server, deleteOptions.BearerToken)
	if err != nil {
		glog.Fatalf("Unable to create k8s client set: %v", err)
	}

	for _, name := range args {
		err := clientset.ReleaseV1alpha1().Releases(deleteOptions.Namespace).Delete(name, nil)
		if err != nil {
			glog.Fatalln(err)
		}
		fmt.Printf("%s deleted\n", name)
	}
}
