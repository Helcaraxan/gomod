package filters

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/depgraph"
)

type ArbitraryModules struct {
	Modules []string
}

func (f *ArbitraryModules) Apply(log *zap.Logger, g *depgraph.Graph) *depgraph.Graph {
	for _, module := range f.Modules {
		if err := g.Graph.DeleteNode("module " + module); err != nil {
			log.Warn("Could not remove module from graph.", zap.String("module", module))
		}
	}
	return g
}
