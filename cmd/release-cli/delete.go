package main

import (
	"fmt"

	"github.com/caicloud/clientset/kubernetes"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
)

func init() {
	root.AddCommand(delete)
	fs := delete.Flags()

	fs.StringVarP(&deleteOptions.Server, "server", "s", "", "Kubenetes master host")
	fs.StringVarP(&deleteOptions.BearerToken, "bearer-token", "b", "", "Kubenetes master bearer token")
	fs.StringVarP(&deleteOptions.Namespace, "namespace", "n", "", "Kubenetes namespace")
}

var deleteOptions = struct {
	Server      string
	BearerToken string
	Namespace   string
}{}

var delete = &cobra.Command{
	Use:   "delete",
	Short: "Delete a release from kubernetes cluster",
	Run:   runDelete,
}

func runDelete(cmd *cobra.Command, args []string) {
	if deleteOptions.Server == "" || deleteOptions.BearerToken == "" {
		glog.Fatalln("--server and --bearer-token must be set")
	}

	if deleteOptions.Namespace == "" {
		glog.Fatalln("--namespace must be set")
	}

	if len(args) <= 0 {
		glog.Fatalln("Must specify release name")
	}

	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host:        deleteOptions.Server,
		BearerToken: deleteOptions.BearerToken,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
	if err != nil {
		glog.Fatalln(err)
	}

	for _, name := range args {
		err := clientset.ReleaseV1alpha1().Releases(deleteOptions.Namespace).Delete(name, nil)
		if err != nil {
			glog.Fatalln(err)
		}
		fmt.Printf("%s deleted\n", name)
	}
}
