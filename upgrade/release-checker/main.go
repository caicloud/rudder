package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/caicloud/clientset/kubernetes"
	"github.com/caicloud/clientset/kubernetes/scheme"
	"github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/clientset/pkg/apis/resource/v1beta1"
	"github.com/caicloud/go-common/kubernetes/client"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/render"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getClientByCluster(c v1beta1.Cluster) (kubernetes.Interface, error) {
	var (
		err           error
		clusterconfig *rest.Config
		rawConfig     = c.Spec.Auth.KubeConfig
	)
	if rawConfig == nil {
		clusterconfig = &rest.Config{
			Username: c.Spec.Auth.KubeUser,
			Password: c.Spec.Auth.KubePassword,
			Host:     fmt.Sprintf("https://%s:%s", c.Spec.Auth.EndpointIP, c.Spec.Auth.EndpointPort),
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		}
	} else {
		clusterconfig, err = clientcmd.NewDefaultClientConfig(*rawConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, err
		}
	}

	return client.NewFromConfig(clusterconfig)
}

func getAllRelease(config *rest.Config) ([]v1alpha1.Release, error) {
	var releases []v1alpha1.Release
	kubeClient, err := client.NewFromConfig(config)
	if err != nil {
		return nil, err
	}
	clusterList, err := kubeClient.ResourceV1beta1().Clusters().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, c := range clusterList.Items {
		client, err := getClientByCluster(c)
		if err != nil {
			fmt.Printf("WARN: cluster %s get release failed, %v\n", c.Name, err)
			continue
		}
		releasesList, err := client.ReleaseV1alpha1().Releases("").List(metav1.ListOptions{})
		if err != nil {
			fmt.Printf("WARN: cluster %s get release failed, %v\n", c.Name, err)
			continue
		}
		for i, r := range releasesList.Items {
			if r.Namespace != "default" && r.Namespace != "kube-system" {
				releasesList.Items[i].ClusterName = c.Name
				releases = append(releases, releasesList.Items[i])
			}
		}
	}

	return releases, nil
}

func getSortedResources(resources []string) ([]string, error) {
	var ret []string
	codec := kube.NewYAMLCodec(scheme.Scheme, scheme.Scheme)
	objs, err := codec.ResourcesToObjects(resources)
	if err != nil {
		return nil, err
	}
	kube.InstallOrder.Sort(objs)
	for _, o := range objs {
		gvk := o.GetObjectKind().GroupVersionKind()
		if gvk.Kind == "Deployment" || gvk.Kind == "StatefulSet" || gvk.Kind == "DaemonSet" || gvk.Kind == "CronJob" {
			yamlData, err := yaml.Marshal(o)
			if err != nil {
				return nil, err
			}
			ret = append(ret, string(yamlData))
		}
	}
	return ret, nil
}

func main() {
	kubeConfig := flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "absolute path to the kubeconfig file")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err)
	}
	releases, err := getAllRelease(config)
	if err != nil {
		panic(err)
	}

	var suspend bool
	for _, r := range releases {
		if r.Namespace != "default" && r.Namespace != "kube-system" {
			fmt.Printf("----------%s/%s/%s----------------\n", r.ClusterName, r.Namespace, r.Name)
			realChart, err := GenerateChart(r.Spec.Config)
			if err != nil {
				fmt.Printf("WARN: %s/%s/%s GenerateChart failed, %#v\n", r.ClusterName, r.Namespace, r.Name, err)
				continue
			}
			tplData, err := Archive(realChart)
			if err != nil {
				fmt.Printf("WARN: %s/%s/%s Archive failed, %#v\n", r.ClusterName, r.Namespace, r.Name, err)
				continue
			}
			carrier, err := render.NewRender().Render(&render.Options{
				Namespace: r.Namespace,
				Release:   r.Name,
				Version:   r.Status.Version,
				Template:  tplData,
				Config:    r.Spec.Config,
				Suspend:   &suspend,
			})
			if err != nil {
				fmt.Printf("WARN: %s/%s/%s render failed, %#v\n", r.ClusterName, r.Namespace, r.Name, err)
				continue
			}
			renderedManifests, err := getSortedResources(carrier.Resources())
			if err != nil {
				fmt.Printf("WARN: %s/%s/%s getSortedResources failed, %#v\n", r.ClusterName, r.Namespace, r.Name, err)
				continue
			}
			originManifests, err := getSortedResources(render.SplitManifest(r.Status.Manifest))
			if err != nil {
				fmt.Printf("WARN: %s/%s/%s getSortedResources failed, %#v\n", r.ClusterName, r.Namespace, r.Name, err)
				continue
			}
			fmt.Println(cmp.Diff(renderedManifests, originManifests))
		}
	}
}
