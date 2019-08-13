package filters

import (
	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

type ArbitraryDependencies struct {
	Dependencies []string
}

func (f *ArbitraryDependencies) Apply(logger *logrus.Logger, graph *depgraph.DepGraph) *depgraph.DepGraph {
	filteredGraph := graph.DeepCopy()
	for _, dependency := range f.Dependencies {
		filteredGraph.RemoveDependency(dependency)
	}
	return filteredGraph
}
