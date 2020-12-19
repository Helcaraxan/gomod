package filters

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

type ArbitraryDependencies struct {
	Dependencies []string
}

func (f *ArbitraryDependencies) Apply(log *zap.Logger, graph *depgraph.Graph) *depgraph.Graph {
	filteredGraph := graph.DeepCopy()
	for _, dependency := range f.Dependencies {
		filteredGraph.RemoveModule(dependency)
	}
	return filteredGraph
}
