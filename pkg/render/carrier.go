package render

import "context"

type CarrierOrder string

const (
	// An example of positive order:
	//  a ---> b
	//  |----> c
	// It executes b and c in parallel. If b and c finished with no error, It
	// executes a.
	PositiveOrder CarrierOrder = "PositiveOrder"
	// An example of reversed order:
	//  a ---> b
	//  |----> c
	// It executes a. If succeeded, executes b and c in parallel.
	ReversedOrder CarrierOrder = "ReversedOrder"
)

// CarrierHandler handles a bundle resources from a node.  The handler should
// return nil when it handled all resources and no error occurred.
type CarrierHandler func(ctx context.Context, node string, resources []string) error

// Carrier contains a bundle of Kubernetes resources. It also contains
// relationships of resources.
type Carrier interface {
	// Run executes all resources via handler. The ctx should be a cancelable
	// context. All nodes will be executed one by one. The order of nodes depends
	// on the dependencies of nodes. If any error occurred, it will cancel all
	// processes and return an error.
	Run(ctx context.Context, order CarrierOrder, handler CarrierHandler) error
	// Resources returns all resources.
	Resources() []string
	// ResourcesOf returns resources of a target. target is a path of resource node.
	// If there is no node for target, it returns an error.
	ResourcesOf(target string) ([]string, error)
}
