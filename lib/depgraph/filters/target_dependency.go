package filters

import (
	"strings"

	"github.com/blang/semver"
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

// TargetDependencies implements the `depgraph.Filter` interface. It removes any edges that are not
// part of a chain leading to one of the specified dependencies. If for a given dependency has a
// version set, we only keep edges that prevent the use of the dependency at that given version
// under the constraints of minimal version selection.
type TargetDependencies struct {
	Targets []*struct {
		Module  string
		Version string
	}
}

// Apply returns a copy of the dependency graph with all dependencies that are part of chains
// that need to be modified for the specified dependency to be set to a given target version
// annotated as such.
func (f *TargetDependencies) Apply(log *zap.Logger, graph *depgraph.Graph) *depgraph.Graph {
	if len(f.Targets) == 0 {
		return graph
	}

	keep := map[string]struct{}{}
	for _, dep := range f.Targets {
		applyFilter(log, graph, &targetDependencyFilter{
			module:  dep.Module,
			version: dep.Version,
		}, keep)
	}

	log.Debug("Pruning the dependency graph of irrelevant paths.")
	subGraph := graph.DeepCopy()
	for _, dependency := range graph.Modules.List() {
		if _, ok := keep[dependency.Name()]; !ok {
			log.Debug("Pruning dependency.", zap.String("dependency", dependency.Name()))
			subGraph.RemoveModule(dependency.Name())
		}
	}
	return subGraph
}

type targetDependencyFilter struct {
	module  string
	version string
}

func applyFilter(
	logger *zap.Logger,
	graph *depgraph.Graph,
	filter *targetDependencyFilter,
	keep map[string]struct{},
) {
	filterModule, ok := graph.GetModule(filter.module)
	if !ok {
		return
	}

	keep[filterModule.Name()] = struct{}{}

	logger.Debug("Marking subgraph.", zap.String("dependency", filter.module))
	if filter.version != "" {
		logger.Debug("Only considering dependencies preventing use of a specific version.", zap.String("version", filter.version))
	}
	var todo []*depgraph.ModuleReference
	for _, ref := range filterModule.Predecessors.List() {
		predecessor := ref.(*depgraph.ModuleReference)
		if dependencyMatchesFilter(predecessor, filter) {
			logger.Debug("Keeping dependency", zap.String("dependency", predecessor.Name()))
			keep[predecessor.Name()] = struct{}{}
			todo = append(todo, predecessor)
		}
	}

	for len(todo) > 0 {
		dependency := todo[0]
		for _, ref := range dependency.Predecessors.List() {
			predecessor := ref.(*depgraph.ModuleReference)
			if _, ok := keep[predecessor.Name()]; !ok {
				logger.Debug("Keeping dependency", zap.String("dependency", predecessor.Name()))
				keep[predecessor.Name()] = struct{}{}
				todo = append(todo, predecessor)
			}
		}
		todo = todo[1:]
	}
}

func dependencyMatchesFilter(dependency *depgraph.ModuleReference, filter *targetDependencyFilter) bool {
	if dependency.VersionConstraint == "" || filter.version == "" {
		return true
	}
	constraint := semver.MustParse(strings.TrimLeft(dependency.VersionConstraint, "v"))
	depVersion := semver.MustParse(strings.TrimLeft(filter.version, "v"))
	return constraint.GT(depVersion)
}
