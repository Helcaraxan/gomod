package filters

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

type ArbitraryDependencies struct {
	Dependencies []string
}

func (f *ArbitraryDependencies) Apply(log *zap.Logger, graph *depgraph.ModuleGraph) *depgraph.ModuleGraph {
	filteredGraph := graph.DeepCopy()
	for _, dependency := range f.Dependencies {
		filteredGraph.RemoveDependency(dependency)
	}
	return filteredGraph
}
