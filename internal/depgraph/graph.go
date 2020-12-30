package depgraph

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
)

// Graph represents a Go module's dependency graph.
type Graph struct {
	Path string

	Graph    *graph.HierarchicalDigraph
	Main     *Module
	Replaces map[string]string

	log *zap.Logger
}

type Level uint8

const (
	LevelModules Level = iota
	LevelPackages
)

// Transform can be used to transform a Graph instance. The particular transformation will depend on
// the underlying implementation but it can range from dependency pruning, to adding graph
// annotations, to edge manipulation.
type Transform interface {
	// Apply returns a, potentially, modified copy of the input Graph instance. The actual
	// modifications depend on the underlying type and implementation of the particular
	// GraphTransform.
	Apply(*zap.Logger, *Graph) *Graph
}

// NewGraph returns a new Graph instance which will use the specified logger for writing log output.
// If a nil value is passed a null-logger will be used instead.
func NewGraph(log *zap.Logger, path string, main *modules.ModuleInfo) *Graph {
	if log == nil {
		log = zap.NewNop()
	}
	g := &Graph{
		Path:     path,
		Graph:    graph.NewHierarchicalDigraph(),
		Replaces: map[string]string{},
		log:      log,
	}
	g.Main = g.addModule(main)
	return g
}

func (g *Graph) Transform(transformations ...Transform) *Graph {
	ng := g
	for _, transformation := range transformations {
		ng = transformation.Apply(g.log, ng)
	}
	return ng
}

func (g *Graph) getModule(name string) (*Module, bool) {
	if replaced, ok := g.Replaces[name]; ok {
		name = replaced
	}
	node, err := g.Graph.GetNode(moduleHash(name))
	if err != nil {
		return nil, false
	}
	return node.(*ModuleReference).Module, true
}

func (g *Graph) addModule(module *modules.ModuleInfo) *Module {
	if module == nil {
		return nil
	} else if node, _ := g.Graph.GetNode(moduleHash(module.Path)); node != nil {
		return node.(*ModuleReference).Module
	}

	newDependencyReference := &ModuleReference{
		Module:            NewModule(module),
		VersionConstraint: module.Version,
	}
	_ = g.Graph.AddNode(newDependencyReference)
	if module.Replace != nil {
		g.Replaces[module.Replace.Path] = module.Path
	}
	return newDependencyReference.Module
}

func (g *Graph) removeModule(name string) {
	if replaced, ok := g.Replaces[name]; ok {
		delete(g.Replaces, name)
		name = replaced
	}

	for replace, replaced := range g.Replaces {
		if replaced == name {
			delete(g.Replaces, replace)
		}
	}

	node, err := g.Graph.GetNode(moduleHash(name))
	if err != nil {
		return
	}

	mod := node.(*ModuleReference)
	for _, node = range mod.Successors().List() {
		node.Predecessors().Delete(mod.Hash())
	}
	for _, node = range mod.Predecessors().List() {
		node.Successors().Delete(mod.Hash())
	}
	_ = g.Graph.DeleteNode(moduleHash(name))
}
