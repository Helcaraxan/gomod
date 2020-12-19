package depgraph

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/modules"
)

// Graph represents a Go module's dependency graph.
type Graph struct {
	Path string

	Main     *Module
	Modules  Edges
	Packages Edges

	log      *zap.Logger
	replaces map[string]string
}

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
	newGraph := &Graph{
		Path:     path,
		Modules:  NewEdges(),
		Packages: NewEdges(),
		log:      log,
		replaces: map[string]string{},
	}
	newGraph.Main = newGraph.AddModule(main)
	return newGraph
}

func (g *Graph) GetModule(name string) (*Module, bool) {
	if replaced, ok := g.replaces[name]; ok {
		name = replaced
	}
	node, ok := g.Modules.Get(name)
	if !ok {
		return nil, false
	}
	return node.(*ModuleReference).Module, true
}

func (g *Graph) AddModule(module *modules.ModuleInfo) *Module {
	if module == nil {
		return nil
	} else if node, ok := g.Modules.Get(module.Path); ok && node != nil {
		return node.(*ModuleReference).Module
	}

	newDependencyReference := &ModuleReference{
		Module: &Module{
			Info:         module,
			predecessors: NewEdges(),
			successors:   NewEdges(),
		},
		VersionConstraint: module.Version,
	}
	g.Modules.Add(newDependencyReference)
	if module.Replace != nil {
		g.replaces[module.Replace.Path] = module.Path
	}
	return newDependencyReference.Module
}

func (g *Graph) RemoveModule(name string) {
	g.log.Debug("Removing dependency.", zap.String("dependency", name))
	if replaced, ok := g.replaces[name]; ok {
		delete(g.replaces, name)
		name = replaced
	}

	for replace, replaced := range g.replaces {
		if replaced == name {
			delete(g.replaces, replace)
		}
	}

	node, ok := g.Modules.Get(name)
	if !ok {
		return
	}
	mod := node.(*ModuleReference)
	for _, node = range mod.Successors().List() {
		node.Predecessors().Delete(mod.Name())
	}
	for _, node = range mod.Predecessors().List() {
		node.Successors().Delete(mod.Name())
	}
	g.Modules.Delete(name)
}

// DeepCopy returns a separate copy of the current dependency graph that can be
// safely modified without affecting the original graph.
func (g *Graph) DeepCopy() *Graph {
	g.log.Debug("Deep-copying dependency graph.", zap.String("module", g.Main.Name()))

	newGraph := NewGraph(g.log, g.Path, g.Main.Info)
	for _, module := range g.Modules.List() {
		if newModule := newGraph.AddModule(module.(*ModuleReference).Info); newModule == nil {
			g.log.Error("Encountered an empty dependency.", zap.String("dependency", newModule.Name()))
		}
	}

	for _, module := range g.Modules.List() {
		dependency := module.(*ModuleReference)

		newDependency, _ := newGraph.GetModule(dependency.Name())
		for _, predecessor := range dependency.Predecessors().List() {
			newPredecessor, ok := newGraph.GetModule(predecessor.Name())
			if !ok {
				g.log.Warn(
					"Could not find information for predecessor.",
					zap.String("predecessor", predecessor.Name()),
					zap.String("dependency", dependency.Name()),
				)
				continue
			}
			newDependency.Predecessors().Add(&ModuleReference{
				Module:            newPredecessor,
				VersionConstraint: predecessor.(*ModuleReference).VersionConstraint,
			})
		}
		for _, successor := range dependency.Successors().List() {
			newSuccessor, ok := newGraph.GetModule(successor.Name())
			if !ok {
				g.log.Warn(
					"Could not find information for successor.",
					zap.String("successor", successor.Name()),
					zap.String("dependency", dependency.Name()),
				)
				continue
			}
			newDependency.Successors().Add(&ModuleReference{
				Module:            newSuccessor,
				VersionConstraint: successor.(*ModuleReference).VersionConstraint,
			})
		}
	}

	for original, replacement := range g.replaces {
		newGraph.replaces[original] = replacement
	}

	g.log.Debug("Created a deep copy of graph.")
	return newGraph
}

func (g *Graph) Transform(transformations ...Transform) *Graph {
	graph := g
	for _, transformation := range transformations {
		graph = transformation.Apply(g.log, graph)
	}
	return graph
}
