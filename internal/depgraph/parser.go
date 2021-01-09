package depgraph

import (
	"os"
	"regexp"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/modules"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

// GetGraph will return the dependency graph for the Go module that can be found at the specified
// path.
func GetGraph(dl *logger.Builder, path string) (*DepGraph, error) {
	if dl == nil {
		dl = logger.NewBuilder(os.Stderr)
	}
	log := dl.Domain(logger.GraphDomain)
	log.Debug("Creating dependency graph.")

	mainModule, moduleInfo, err := modules.GetDependencies(dl.Domain(logger.ModuleInfoDomain), path)
	if err != nil {
		return nil, err
	}

	g := NewGraph(log, path, mainModule)
	for _, module := range moduleInfo {
		g.AddModule(module)
	}

	if err = g.buildImportGraph(dl); err != nil {
		return nil, err
	}

	if err = g.overlayModuleDependencies(dl); err != nil {
		return nil, err
	}

	var roots []graph.Node
	for _, module := range g.Graph.GetLevel(0).List() {
		if module.Predecessors().Len() == 0 && module.Hash() != g.Main.Hash() {
			roots = append(roots, module)
		}
	}

	for len(roots) > 0 {
		next := roots[0]
		roots = roots[1:]

		for _, child := range next.Successors().List() {
			if child.Predecessors().Len() == 1 {
				roots = append(roots, child)
			}
		}

		log.Debug("Removing module as it not connected to the final graph.", zap.String("dependency", next.Name()))
		if err = g.Graph.DeleteNode(next.Hash()); err != nil {
			return nil, err
		}
	}

	return g, nil
}
