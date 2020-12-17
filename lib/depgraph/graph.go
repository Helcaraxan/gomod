package depgraph

import (
	"time"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/modules"
)

// ModuleGraph represents a Go module's dependency graph.
type ModuleGraph struct {
	Path string

	Main         *Module
	Dependencies *DependencyMap

	log      *zap.Logger
	replaces map[string]string
}

// Transform can be used to transform a ModuleGraph instance. The particular transformation will
// depend on the underlying implementation but it can range from dependency pruning, to adding graph
// annotations, to edge manipulation.
type Transform interface {
	// Apply returns a, potentially, modified copy of the input ModuleGraph instance. The actual
	// modifications depend on the underlying type and implementation of the particular
	// GraphTransform.
	Apply(*zap.Logger, *ModuleGraph) *ModuleGraph
}

// NewGraph returns a new ModuleGraph instance which will use the specified
// logger for writing log output. If nil a null-logger will be used instead.
func NewGraph(log *zap.Logger, path string, main *modules.ModuleInfo) *ModuleGraph {
	if log == nil {
		log = zap.NewNop()
	}
	newGraph := &ModuleGraph{
		Path:         path,
		Dependencies: NewDependencyMap(),
		log:          log,
		replaces:     map[string]string{},
	}
	newGraph.Main = newGraph.AddModule(main)
	return newGraph
}

func (g *ModuleGraph) GetModule(name string) (*Module, bool) {
	if replaced, ok := g.replaces[name]; ok {
		name = replaced
	}
	dependencyReference, ok := g.Dependencies.Get(name)
	if !ok {
		return nil, false
	}
	return dependencyReference.Module, true
}

func (g *ModuleGraph) AddModule(module *modules.ModuleInfo) *Module {
	if module == nil {
		return nil
	} else if dependencyReference, ok := g.Dependencies.Get(module.Path); ok && dependencyReference != nil {
		return dependencyReference.Module
	}

	newDependencyReference := &DependencyReference{
		Module: &Module{
			Info:         module,
			Predecessors: NewDependencyMap(),
			Successors:   NewDependencyMap(),
		},
		VersionConstraint: module.Version,
	}
	g.Dependencies.Add(newDependencyReference)
	if module.Replace != nil {
		g.replaces[module.Replace.Path] = module.Path
	}
	return newDependencyReference.Module
}

func (g *ModuleGraph) RemoveModule(name string) {
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

	node, ok := g.Dependencies.Get(name)
	if !ok {
		return
	}
	for _, successor := range node.Successors.List() {
		successor.Predecessors.Delete(node.Name())
	}
	for _, predecessor := range node.Predecessors.List() {
		predecessor.Successors.Delete(node.Name())
	}
	g.Dependencies.Delete(name)
}

// DeepCopy returns a separate copy of the current dependency graph that can be
// safely modified without affecting the original graph.
func (g *ModuleGraph) DeepCopy() *ModuleGraph {
	g.log.Debug("Deep-copying dependency graph.", zap.String("module", g.Main.Name()))

	newGraph := NewGraph(g.log, g.Path, g.Main.Info)
	for _, dependency := range g.Dependencies.List() {
		if module := newGraph.AddModule(dependency.Info); module == nil {
			g.log.Error("Encountered an empty dependency.", zap.String("dependency", module.Name()))
		}
	}

	for _, dependency := range g.Dependencies.List() {
		newDependency, _ := newGraph.GetModule(dependency.Name())
		for _, predecessor := range dependency.Predecessors.List() {
			newPredecessor, ok := newGraph.GetModule(predecessor.Name())
			if !ok {
				g.log.Warn(
					"Could not find information for predecessor.",
					zap.String("predecessor", predecessor.Name()),
					zap.String("dependency", dependency.Name()),
				)
				continue
			}
			newDependency.Predecessors.Add(&DependencyReference{
				Module:            newPredecessor,
				VersionConstraint: predecessor.VersionConstraint,
			})
		}
		for _, successor := range dependency.Successors.List() {
			newSuccessor, ok := newGraph.GetModule(successor.Name())
			if !ok {
				g.log.Warn(
					"Could not find information for successor.",
					zap.String("successor", successor.Name()),
					zap.String("dependency", dependency.Name()),
				)
				continue
			}
			newDependency.Successors.Add(&DependencyReference{
				Module:            newSuccessor,
				VersionConstraint: successor.VersionConstraint,
			})
		}
	}

	for original, replacement := range g.replaces {
		newGraph.replaces[original] = replacement
	}

	g.log.Debug("Created a deep copy of graph.")
	return newGraph
}

func (g *ModuleGraph) Transform(transformations ...Transform) *ModuleGraph {
	graph := g
	for _, transformation := range transformations {
		graph = transformation.Apply(g.log, graph)
	}
	return graph
}

// Module represents a module in a Go module's dependency graph.
type Module struct {
	Info         *modules.ModuleInfo
	Predecessors *DependencyMap
	Successors   *DependencyMap
}

// Name of the module represented by this Dependency in the ModuleGraph instance.
func (n *Module) Name() string {
	return n.Info.Path
}

// SelectedVersion corresponds to the version of the dependency represented by
// this Dependency which was selected for use.
func (n *Module) SelectedVersion() string {
	if n.Info.Replace != nil {
		return n.Info.Replace.Version
	}
	return n.Info.Version
}

// Timestamp returns the time corresponding to the creation of the version at
// which this dependency is used.
func (n *Module) Timestamp() *time.Time {
	if n.Info.Replace != nil {
		return n.Info.Replace.Time
	}
	return n.Info.Time
}
