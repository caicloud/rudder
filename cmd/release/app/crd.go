package app

import (
	"github.com/caicloud/clientset/kubernetes"
	apiextensionsv1beta1 "github.com/caicloud/clientset/pkg/apis/apiextensions/v1beta1"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The definitions of release and release history.
var crds = []apiextensionsv1beta1.CustomResourceDefinition{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "releases.release.caicloud.io",
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   "release.caicloud.io",
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Version: "v1alpha1",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Kind:     "Release",
				ListKind: "ReleaseList",
				Plural:   "releases",
				Singular: "release",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "releasehistories.release.caicloud.io",
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   "release.caicloud.io",
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Version: "v1alpha1",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Kind:     "ReleaseHistory",
				ListKind: "ReleaseHistoryList",
				Plural:   "releasehistories",
				Singular: "releasehistory",
			},
		},
	},
}

// EnsureCRD ensures release and releasehistory crd exist.
func EnsureCRD(kubeClient kubernetes.Interface) error {
	client := kubeClient.ApiextensionsV1beta1().CustomResourceDefinitions()
	for _, crd := range crds {
		_, err := client.Get(crd.Name, metav1.GetOptions{})
		if err == nil {
			glog.V(1).Infof("Existent CRD: %s", crd.Name)
			continue
		} else if !errors.IsNotFound(err) {
			return err
		}
		// Create CRD
		_, err = client.Create(&crd)
		if err != nil {
			return err
		}
		glog.V(1).Infof("Create CRD: %s", crd.Name)
	}
	return nil
}
