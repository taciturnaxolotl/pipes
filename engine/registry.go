package engine

import (
	"fmt"
	"sync"

	"github.com/kierank/pipes/nodes"
	"github.com/kierank/pipes/nodes/sources"
	"github.com/kierank/pipes/nodes/transforms"
)

type Registry struct {
	mu       sync.RWMutex
	nodeImpls map[string]nodes.Node
}

func NewRegistry() *Registry {
	r := &Registry{
		nodeImpls: make(map[string]nodes.Node),
	}

	// Register built-in nodes
	r.Register(&sources.RSSSourceNode{})
	r.Register(&transforms.FilterNode{})
	r.Register(&transforms.SortNode{})
	r.Register(&transforms.LimitNode{})

	return r
}

func (r *Registry) Register(node nodes.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodeImpls[node.Type()] = node
}

func (r *Registry) Get(nodeType string) (nodes.Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, ok := r.nodeImpls[nodeType]
	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}

	return node, nil
}

func (r *Registry) GetAll() []nodes.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodeList := make([]nodes.Node, 0, len(r.nodeImpls))
	for _, node := range r.nodeImpls {
		nodeList = append(nodeList, node)
	}

	return nodeList
}
