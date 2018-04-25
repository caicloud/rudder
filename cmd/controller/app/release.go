package app

import (
	"github.com/caicloud/clientset/informers"
	"github.com/caicloud/clientset/kubernetes"
	"github.com/caicloud/clientset/kubernetes/scheme"
	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/caicloud/rudder/cmd/controller/app/options"
	"github.com/caicloud/rudder/pkg/kube"
	"github.com/caicloud/rudder/pkg/store"
	"github.com/caicloud/rudder/pkg/version"
	"github.com/golang/glog"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
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
	// IgnoredKinds provides kinds which need be ignored when deleted.
	IgnoredKinds []schema.GroupVersionKind
	// Stop is the stop channel
	Stop <-chan struct{}
}

// Run runs the ReleaseServer. This should never exit.
func Run(s *options.ReleaseServer) error {
	glog.Infof("Initialize release server")
	glog.Infof("Rudder Build Information, %v", version.Get().Pretty())
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
		IgnoredKinds:    IgnoredKinds(),
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
}

// AvailableKinds returns all kinds can be used by controllers.
func AvailableKinds() []schema.GroupVersionKind {
	return []schema.GroupVersionKind{
		releaseapi.SchemeGroupVersion.WithKind("Release"),
		releaseapi.SchemeGroupVersion.WithKind("ReleaseHistory"),
		batchv1beta1.SchemeGroupVersion.WithKind("CronJob"),
		apps.SchemeGroupVersion.WithKind("DaemonSet"),
		apps.SchemeGroupVersion.WithKind("Deployment"),
		batch.SchemeGroupVersion.WithKind("Job"),
		apps.SchemeGroupVersion.WithKind("StatefulSet"),
		core.SchemeGroupVersion.WithKind("Service"),
		core.SchemeGroupVersion.WithKind("PersistentVolumeClaim"),
		core.SchemeGroupVersion.WithKind("ConfigMap"),
		core.SchemeGroupVersion.WithKind("Secret"),
	}
}

// IgnoredKinds provides kinds which need be ignored when deleted.
func IgnoredKinds() []schema.GroupVersionKind {
	return []schema.GroupVersionKind{
		core.SchemeGroupVersion.WithKind("PersistentVolumeClaim"),
	}
}
