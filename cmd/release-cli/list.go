package main

import (
	"github.com/caicloud/clientset/kubernetes"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func init() {
	root.AddCommand(list)
	fs := list.Flags()

	fs.StringVarP(&listOptions.Server, "server", "s", "", "Kubenetes master host")
	fs.StringVarP(&listOptions.BearerToken, "bearer-token", "t", "", "Kubenetes master bearer token")
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
		glog.Fatalln("--server or --bearer-token must be set")
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

	table := [][]string{{"NAMESPACE", "NAME", "STATUS"}}
	for _, r := range list.Items {
		if len(r.Status.Conditions) > 0 {
			conditions := r.Status.Conditions
			table = append(table, []string{r.Namespace, r.Name, string(conditions[0].Type)})
		} else {
			table = append(table, []string{r.Namespace, r.Name, "N/A"})
		}
	}
	printTable(table)
}
