package depgraph

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
)

// DepGraph represents a Go module's dependency graph.
type DepGraph struct {
	Path  string
	Main  *Module
	Graph *graph.HierarchicalDigraph

	log      *zap.Logger
	replaces map[string]string
}

type Level uint8

const (
	LevelModules Level = iota
	LevelPackages
)

// NewGraph returns a new Graph instance which will use the specified logger for writing log output.
// If a nil value is passed a null-logger will be used instead.
func NewGraph(log *zap.Logger, path string, main *modules.ModuleInfo) *DepGraph {
	if log == nil {
		log = zap.NewNop()
	}
	g := &DepGraph{
		Path:     path,
		Graph:    graph.NewHierarchicalDigraph(),
		replaces: map[string]string{},
		log:      log,
	}
	g.Main = g.AddModule(main)
	return g
}

func (g *DepGraph) getModule(name string) (*Module, bool) {
	if replaced, ok := g.replaces[name]; ok {
		name = replaced
	}
	node, err := g.Graph.GetNode(moduleHash(name))
	if err != nil {
		return nil, false
	}
	return node.(*Module), true
}

func (g *DepGraph) AddModule(module *modules.ModuleInfo) *Module {
	if module == nil {
		return nil
	} else if node, _ := g.Graph.GetNode(moduleHash(module.Path)); node != nil {
		return node.(*Module)
	}

	newModule := NewModule(module)

	_ = g.Graph.AddNode(newModule)
	if module.Replace != nil {
		g.replaces[module.Replace.Path] = module.Path
	}
	return newModule
}
