package filters

import (
	"strings"

	"github.com/blang/semver"
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/depgraph"
	"github.com/Helcaraxan/gomod/internal/graph"
)

// TargetModules implements the `depgraph.Filter` interface. It removes any edges that are not part
// of a chain leading to one of the specified dependencies. If for a given dependency has a version
// set, we only keep edges that prevent the use of the dependency at that given version under the
// constraints of minimal version selection.
type TargetModules struct {
	Targets []*struct {
		Module  string
		Version string
	}
}

// Apply returns a copy of the dependency graph with all dependencies that are part of chains
// that need to be modified for the specified dependency to be set to a given target version
// annotated as such.
func (f *TargetModules) Apply(log *zap.Logger, g *depgraph.DepGraph) *depgraph.DepGraph {
	if len(f.Targets) == 0 {
		return g
	}

	keep := map[string]struct{}{}
	for _, dep := range f.Targets {
		applyFilter(log, g, &targetDependencyFilter{
			module:  dep.Module,
			version: dep.Version,
		}, keep)
	}

	log.Debug("Pruning the dependency graph of irrelevant paths.")
	for _, dependency := range g.Graph.GetLevel(int(depgraph.LevelModules)).List() {
		if _, ok := keep[dependency.Name()]; !ok {
			log.Debug("Pruning dependency.", zap.String("dependency", dependency.Name()))
			_ = g.Graph.DeleteNode(dependency.Hash())
		}
	}
	return g
}

type targetDependencyFilter struct {
	module  string
	version string
}

func applyFilter(logger *zap.Logger, g *depgraph.DepGraph, filter *targetDependencyFilter, keep map[string]struct{}) {
	filterNode, _ := g.Graph.GetLevel(int(depgraph.LevelModules)).Get("module " + filter.module)
	if filterNode == nil {
		return
	}

	keep[filterNode.Name()] = struct{}{}

	logger.Debug("Marking subgraph.", zap.String("dependency", filter.module))
	if filter.version != "" {
		logger.Debug("Only considering dependencies preventing use of a specific version.", zap.String("version", filter.version))
	}
	var todo []graph.Node
	for _, node := range filterNode.Predecessors().List() {
		if dependencyMatchesFilter(node.(*depgraph.Module), filter) {
			logger.Debug("Keeping dependency", zap.String("dependency", node.Name()))
			keep[node.Name()] = struct{}{}
			todo = append(todo, node)
		}
	}

	for len(todo) > 0 {
		dependency := todo[0]
		for _, node := range dependency.Predecessors().List() {
			if _, ok := keep[node.Name()]; !ok {
				logger.Debug("Keeping dependency", zap.String("dependency", node.Name()))
				keep[node.Name()] = struct{}{}
				todo = append(todo, node)
			}
		}
		todo = todo[1:]
	}
}

func dependencyMatchesFilter(dependency *depgraph.Module, filter *targetDependencyFilter) bool {
	if dependency.Info.Version == "" || filter.version == "" {
		return true
	}
	constraint := semver.MustParse(strings.TrimLeft(dependency.Info.Version, "v"))
	depVersion := semver.MustParse(strings.TrimLeft(filter.version, "v"))
	return constraint.GT(depVersion)
}
