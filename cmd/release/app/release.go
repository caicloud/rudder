package app

import (
	"github.com/caicloud/clientset/informers"
	"github.com/caicloud/clientset/kubernetes"
	"github.com/caicloud/clientset/kubernetes/scheme"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/release-controller/cmd/release/app/options"
	"github.com/caicloud/release-controller/pkg/kube"
	"github.com/caicloud/release-controller/pkg/store"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	corev1 "k8s.io/client-go/pkg/api/v1"
	appsv1beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	batchv2alpha1 "k8s.io/client-go/pkg/apis/batch/v2alpha1"
	extensionsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
)

type ControllerContext struct {
	// Options provides access to init options.
	Options options.ReleaseServer
	// Scheme is a scheme which contains all available api types.
	Scheme *runtime.Scheme
	// Codec is a common tool for converting between resources and objects.
	Codec kube.Codec
	// Resources Provides all resources which api server can receive.
	Resources kube.APIResources
	// KubeClient provides the client for controllers to use.
	KubeClient kubernetes.Interface
	// ClientPool is a pool for dynamic client.
	ClientPool kube.ClientPool
	// InformerFactory gives access to informers for controllers.
	InformerFactory informers.SharedInformerFactory
	// InformerStore provides generic informers for controllers.
	InformerStore store.IntegrationStore
	// AvailableKinds provides all kinds that controllers can handle.
	AvailableKinds []schema.GroupVersionKind
	// Stop is the stop channel
	Stop <-chan struct{}
}

// Run runs the ReleaseServer. This should never exit.
func Run(s *options.ReleaseServer) error {
	glog.Infof("Initialize release server")
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", s.Kubeconfig)
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	err = EnsureCRD(kubeClient)
	if err != nil {
		return err
	}
	resources, err := kube.NewAPIResources(kubeClient)
	if err != nil {
		return err
	}
	pool, err := kube.NewClientPool(scheme.Scheme, kubeConfig, resources)
	if err != nil {
		return err
	}
	stop := wait.NeverStop
	informerFactory := informers.NewSharedInformerFactory(kubeClient, s.ResyncPeriod)
	informerStore := store.NewIntegrationStore(resources, informerFactory, stop)
	ctx := ControllerContext{
		Options:         *s,
		Scheme:          scheme.Scheme,
		Codec:           kube.NewYAMLCodec(scheme.Scheme, scheme.Scheme),
		Resources:       resources,
		KubeClient:      kubeClient,
		ClientPool:      pool,
		InformerFactory: informerFactory,
		InformerStore:   informerStore,
		AvailableKinds:  AvailableKinds(),
		Stop:            stop,
	}
	initializers, err := NewControllerInitializers(s.Controllers)
	if err != nil {
		return err
	}
	err = StartControllers(ctx, initializers)
	if err != nil {
		return err
	}
	ctx.InformerFactory.Start(ctx.Stop)
	select {}
	panic("unreachable")
}

// AvailableKinds returns all kinds can be used by controllers.
func AvailableKinds() []schema.GroupVersionKind {
	return []schema.GroupVersionKind{
		releaseapi.SchemeGroupVersion.WithKind("Release"),
		releaseapi.SchemeGroupVersion.WithKind("ReleaseHistory"),
		batchv2alpha1.SchemeGroupVersion.WithKind("CronJob"),
		extensionsv1beta1.SchemeGroupVersion.WithKind("DaemonSet"),
		appsv1beta1.SchemeGroupVersion.WithKind("Deployment"),
		batchv1.SchemeGroupVersion.WithKind("Job"),
		appsv1beta1.SchemeGroupVersion.WithKind("StatefulSet"),
		apiv1.SchemeGroupVersion.WithKind("Service"),
		corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"),
	}
}
