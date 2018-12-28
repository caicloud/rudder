package main

import (
	"fmt"
	"strings"

	"github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	root.AddCommand(list)
	fs := list.Flags()

	fs.StringVarP(&listOptions.Server, "server", "s", "", "Kubernetes master host")
	fs.StringVarP(&listOptions.BearerToken, "bearer-token", "b", "", "Kubernetes master bearer token")
	fs.StringVarP(&listOptions.Namespace, "namespace", "n", "", "Kubernetes namespace")
	fs.StringVarP(&listOptions.KubeconfigPath, "kubeconfig", "k", "", "Kubernetes config path")

}

var listOptions = struct {
	Server         string
	BearerToken    string
	KubeconfigPath string
	Namespace      string
}{}

var list = &cobra.Command{
	Use:   "list",
	Short: "List releases from a kubernetes cluster",
	Run:   runList,
}

func runList(cmd *cobra.Command, args []string) {
	if listOptions.KubeconfigPath == "" && (listOptions.Server == "" || listOptions.BearerToken == "") {
		glog.Fatalln("Must specify either --kubeconfig or --bearer-token and --server")
	}

	clientset, err := newClientSet(listOptions.KubeconfigPath, listOptions.Server, listOptions.BearerToken)
	if err != nil {
		glog.Fatalf("Unable to create k8s client set: %v", err)
	}

	list, err := clientset.ReleaseV1alpha1().Releases(listOptions.Namespace).List(metav1.ListOptions{})
	if err != nil {
		glog.Fatalln(err)
	}

	table := [][]string{{"NAMESPACE", "NAME", "VALID", "STATUS"}}
	for _, r := range list.Items {
		condition := "YES"
		for _, c := range r.Status.Conditions {
			if c.Type == v1alpha1.ReleaseFailure {
				condition = "NO"
			}

		}
		counter := make(v1alpha1.ResourceCounter, 0)
		for _, v := range r.Status.Details {
			for _, c := range v.Resources {
				for k, n := range c {
					if _, ok := counter[k]; ok {
						counter[k] += n
					} else {
						counter[k] = n
					}
				}
			}
		}
		table = append(table, []string{r.Namespace, r.Name, condition, printCounter(counter)})
	}
	printTable(table)
}

func printCounter(c v1alpha1.ResourceCounter) string {
	list := make([]string, 0)
	for k, v := range c {
		list = append(list, fmt.Sprintf("%v:%v", k, v))
	}
	return strings.Join(list, ",")
}
