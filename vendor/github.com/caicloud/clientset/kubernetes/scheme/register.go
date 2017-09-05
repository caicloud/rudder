/*
Copyright 2017 caicloud authors. All rights reserved.
*/

package scheme

import (
	configv1alpha1 "github.com/caicloud/clientset/pkg/apis/config/v1alpha1"
	releasev1alpha1 "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

var Scheme = scheme.Scheme
var Codecs = scheme.Codecs
var ParameterCodec = scheme.ParameterCodec

func init() {
	AddToScheme(Scheme)
}

// AddToScheme adds all types of this clientset into the given scheme. This allows composition
// of clientsets, like in:
//
//   import (
//     "k8s.io/client-go/kubernetes"
//     clientsetscheme "k8s.io/client-go/kuberentes/scheme"
//     aggregatorclientsetscheme "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/scheme"
//   )
//
//   kclientset, _ := kubernetes.NewForConfig(c)
//   aggregatorclientsetscheme.AddToScheme(clientsetscheme.Scheme)
//
// After this, RawExtensions in Kubernetes types will serialize kube-aggregator types
// correctly.
func AddToScheme(scheme *runtime.Scheme) {
	configv1alpha1.AddToScheme(scheme)
	releasev1alpha1.AddToScheme(scheme)

}