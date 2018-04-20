package app

import (
	"fmt"

	"github.com/caicloud/rudder/pkg/controller/gc"
	"github.com/caicloud/rudder/pkg/controller/release"
	"github.com/caicloud/rudder/pkg/controller/status"
	"github.com/golang/glog"
)

// KnownControllers contains names of controllers
// var KnownControllers = []string{"release-controller", "status-controller", "garbage-collector"}
var KnownControllers = []string{"release-controller", "status-controller"}

// InitFunc is used to launch a particular controller.
type InitFunc func(ctx ControllerContext) error

// NewControllerInitializers is a public map of named controller groups paired to their InitFunc.
func NewControllerInitializers(availableControllers []string) (map[string]InitFunc, error) {
	allControllers := map[string]InitFunc{
		"release-controller": startReleaseController,
		"status-controller":  startStatusController,
		// "garbage-collector":  startGCController,
	}

	result := make(map[string]InitFunc)
	for _, name := range availableControllers {
		if initFunc, ok := allControllers[name]; !ok {
			return nil, fmt.Errorf("there is no controller named: %s", name)
		} else {
			result[name] = initFunc
		}
	}
	return result, nil
}

// StartControllers starts controllers.
func StartControllers(ctx ControllerContext, controllers map[string]InitFunc) error {
	for controllerName, initFn := range controllers {
		glog.V(1).Infof("Starting %q", controllerName)
		err := initFn(ctx)
		if err != nil {
			glog.Errorf("Error starting %q", controllerName)
			return err
		}
		glog.Infof("Started %q", controllerName)
	}
	return nil
}

func startReleaseController(ctx ControllerContext) error {
	releaseController, err := release.NewReleaseController(
		ctx.ClientPool,
		ctx.Codec,
		ctx.InformerStore,
		ctx.KubeClient.ReleaseV1alpha1(),
		ctx.InformerFactory.Release().V1alpha1().Releases(),
		ctx.IgnoredKinds,
	)
	if err != nil {
		return err
	}
	go releaseController.Run(ctx.Stop)
	return nil
}

func startStatusController(ctx ControllerContext) error {
	statusController, err := status.NewResourceStatusController(
		ctx.Codec,
		ctx.InformerStore,
		ctx.KubeClient.ReleaseV1alpha1(),
		ctx.InformerFactory.Release().V1alpha1().Releases(),
	)
	if err != nil {
		return err
	}
	go statusController.Run(ctx.Options.ConcurrentStatusSyncs, ctx.Stop)
	return nil
}

func startGCController(ctx ControllerContext) error {
	garbageCollector, err := gc.NewGarbageCollector(
		ctx.ClientPool,
		ctx.Codec,
		ctx.InformerStore,
		ctx.AvailableKinds,
		ctx.IgnoredKinds,
	)
	if err != nil {
		return err
	}
	go garbageCollector.Run(ctx.Options.ConcurrentGCSyncs, ctx.Stop)
	return nil
}
