package main

import (
	"fmt"

	"github.com/caicloud/clientset/kubernetes"
	"github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func init() {
	root.AddCommand(list)
	fs := list.Flags()

	fs.StringVarP(&listOptions.Server, "server", "s", "", "Kubenetes master host")
	fs.StringVarP(&listOptions.BearerToken, "bearer-token", "b", "", "Kubenetes master bearer token")
	fs.StringVarP(&listOptions.Namespace, "namespace", "n", "", "Kubenetes namespace")
}

var listOptions = struct {
	Server      string
	BearerToken string
	Namespace   string
}{}

var list = &cobra.Command{
	Use:   "list",
	Short: "List releases from a kubernetes cluster",
	Run:   runList,
}

func runList(cmd *cobra.Command, args []string) {
	if listOptions.Server == "" || listOptions.BearerToken == "" {
		glog.Fatalln("--server and --bearer-token must be set")
	}
	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host:        listOptions.Server,
		BearerToken: listOptions.BearerToken,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
	if err != nil {
		glog.Fatalln(err)
	}

	list, err := clientset.ReleaseV1alpha1().Releases(listOptions.Namespace).List(metav1.ListOptions{})
	if err != nil {
		glog.Fatalln(err)
	}

	table := [][]string{{"NAMESPACE", "NAME", "VALID", "AVAILABLE", "PROCESSING", "FAILURE"}}
	for _, r := range list.Items {
		condition := "YES"
		for _, c := range r.Status.Conditions {
			if c.Type == v1alpha1.ReleaseFailure {
				condition = "NO"
			}

		}
		available := int32(0)
		processing := int32(0)
		failure := int32(0)
		for _, v := range r.Status.Details {
			for _, c := range v.Resources {
				available += c.Available
				processing += c.Progressing
				failure += c.Failure
			}
		}
		table = append(table, []string{r.Namespace, r.Name, condition,
			fmt.Sprint(available),
			fmt.Sprint(processing),
			fmt.Sprint(failure),
		})
	}
	printTable(table)
}
