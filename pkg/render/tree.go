package render

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/helm/pkg/releaseutil"
)

// CarrierForManifest returns a carrier for manifest.
func CarrierForManifest(manifest string) (Carrier, error) {
	resources := SplitManifest(manifest)
	return CarrierForResources(resources)
}

// CarrierForResources returns a carrier for resources.
func CarrierForResources(resources []string) (Carrier, error) {
	var root *node
	var res resource
	for _, r := range resources {
		err := yaml.Unmarshal([]byte(r), &res)
		if err != nil {
			return nil, err
		}
		if res.Metadata.Annotations == nil {
			return nil, fmt.Errorf("unknwon resource object")
		}
		path, ok := res.Metadata.Annotations[string(releaseutil.DefaultPathKey)]
		if !ok || path == "" {
			return nil, fmt.Errorf("unknown resource for carrier")
		}
		// Split path by /
		paths := strings.Split(path, "/")
		if len(paths) <= 0 {
			return nil, fmt.Errorf("unknown resource path for carrier")
		}
		if root == nil {
			// Paths length must greater than 0
			root = newNode(paths[0], []string{})
		}
		err = root.add(paths, []string{r})
		if err != nil {
			return nil, err
		}
	}
	return &treeCarrier{
		root: root,
	}, nil
}

// Resource defines common fields of kubernetes resources
type resource struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
}

// treeCarrierFor creates carrier by resources.
func treeCarrierFor(resources map[string][]string) (*treeCarrier, error) {
	var root *node
	for file, resources := range resources {
		paths, err := logicPathForFile(file)
		if err != nil {
			return nil, err
		}
		if root == nil {
			// paths length must greater than 0
			root = newNode(paths[0], []string{})
		}
		err = root.add(paths, resources)
		if err != nil {
			return nil, err
		}
	}
	return &treeCarrier{
		root: root,
	}, nil
}

// logicPathForFile generate logic path for a file path.
// A file path should like:
//  a/templates/deployment.yaml
//  a/charts/b/templates/deployment.yaml
// The two paths should generate:
//  [a]
//  [a,b]
func logicPathForFile(file string) ([]string, error) {
	// Trim suffix "/templates/*"
	templatesPos := strings.LastIndex(file, "/templates/")
	if templatesPos <= 0 {
		return nil, fmt.Errorf("can't find dir templates in file path: %s", file)
	}
	base := file[:templatesPos]
	// Then we split the base path by '/'.
	// The logic path is the even elements.
	// paths shoud like:
	//  [a]
	//  [a,charts,b]
	paths := strings.Split(base, "/")
	if len(paths)%2 != 1 {
		return nil, fmt.Errorf("unexpected file path: %s", base)
	}
	length := 1 + len(paths)/2
	array := make([]string, length)
	for i := 0; i < length; i++ {
		array[i] = paths[i*2]
	}
	return array, nil
}

type node struct {
	name      string
	path      string
	resources []string
	children  map[string]*node
}

// newNode creates a tree node
func newNode(path string, resources []string) *node {
	segments := strings.Split(path, "/")
	return &node{
		name:      segments[len(segments)-1],
		path:      path,
		resources: resources,
		children:  make(map[string]*node),
	}
}

// add generates passing nodes in paths and save resources to the last node.
func (n *node) add(paths []string, resources []string) error {
	if len(paths) <= 0 {
		return fmt.Errorf("there is no path in paths")
	}
	if n.name != paths[0] {
		return fmt.Errorf("paths have a diverse root node: %s", paths[0])
	}
	parent := n
	for i, path := range paths[1:] {
		if target, ok := parent.children[path]; ok {
			parent = target
			continue
		}
		newNode := newNode(strings.Join(paths[:i+1], "/"), []string{})
		parent.children[path] = newNode
		parent = newNode
	}
	parent.resources = append(parent.resources, resources...)
	return nil
}

// find finds specified node by paths.
func (n *node) find(paths []string) (*node, error) {
	if len(paths) <= 0 {
		return nil, fmt.Errorf("there is no path in paths")
	}
	if n.path != paths[0] {
		return nil, fmt.Errorf("paths have a diverse root node: %s", paths[0])
	}
	parent := n
	for _, path := range paths[1:] {
		if target, ok := parent.children[path]; ok {
			parent = target
			continue
		}
		return nil, fmt.Errorf("no node for paths: %s", paths)
	}
	return parent, nil
}

// walkthrough walkthroughs all nodes. The result of handler decides
// whether it should continue.
func (n *node) walkthrough(handler func(*node) bool) bool {
	nodes := []*node{n}
	for len(nodes) > 0 {
		n := nodes[0]
		ok := handler(n)
		if !ok {
			return false
		}
		nodes = nodes[1:]
		for _, n := range n.children {
			nodes = append(nodes, n)
		}
	}
	return true
}

func (n *node) handle(ctx context.Context, handler CarrierHandler) error {
	err := handler(ctx, n.path, n.resources)
	if err == nil {
		err = ctx.Err()
	}
	return err
}

type executor func(ctx context.Context, n *node, handler CarrierHandler, wg *sync.WaitGroup) error

func (n *node) handleChildren(ctx context.Context, handler CarrierHandler, exec executor) error {
	if len(n.children) < 0 {
		return nil
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(n.children))
	errSync := &sync.Mutex{}
	errList := make([]error, 0)
	for _, child := range n.children {
		go func(n *node) {
			if err := exec(ctx, n, handler, wg); err != nil {
				errSync.Lock()
				errList = append(errList, err)
				errSync.Unlock()
			}
		}(child)
	}
	wg.Wait()
	if len(errList) > 0 {
		return errors.NewAggregate(errList)
	}
	return nil
}

func (n *node) execPositively(ctx context.Context, handler CarrierHandler, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}
	err := n.handleChildren(ctx, handler,
		func(ctx context.Context, n *node, handler CarrierHandler, wg *sync.WaitGroup) error {
			return n.execPositively(ctx, handler, wg)
		})
	if err != nil {
		return err
	}
	return n.handle(ctx, handler)
}

func (n *node) execReversely(ctx context.Context, handler CarrierHandler, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}
	err := n.handle(ctx, handler)
	if err != nil {
		return err
	}
	return n.handleChildren(ctx, handler,
		func(ctx context.Context, n *node, handler CarrierHandler, wg *sync.WaitGroup) error {
			return n.execReversely(ctx, handler, wg)
		})
}

// treeCarrier implements a basic tree-like graph for chart now.
type treeCarrier struct {
	root *node
}

// Run executes all resources via handler. The ctx should be a cancelable
// context. All nodes will be executed one by one. The order of nodes depends
// on the dependencies of nodes. If any error occurred, it will cancel all
// processes and return an error.
func (tc *treeCarrier) Run(ctx context.Context, order CarrierOrder, handler CarrierHandler) error {
	switch order {
	case PositiveOrder:
		return tc.root.execPositively(ctx, handler, nil)
	case ReversedOrder:
		return tc.root.execReversely(ctx, handler, nil)
	default:
		return fmt.Errorf("unknown order: %s", order)
	}
}

// Resources returns all resources.
func (tc *treeCarrier) Resources() []string {
	resources := make([]string, 0, 10)
	tc.root.walkthrough(func(n *node) bool {
		if len(n.resources) > 0 {
			resources = append(resources, n.resources...)
		}
		return true
	})
	return resources
}

// ResourcesOf returns resources of a target. target is a path of resource node.
// If there is no node for target, it returns an error.
func (tc *treeCarrier) ResourcesOf(target string) ([]string, error) {
	paths := strings.Split(target, "/")
	node, err := tc.root.find(paths)
	if err != nil {
		return nil, err
	}
	return node.resources, nil
}
