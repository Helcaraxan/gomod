package depgraph

import (
	"regexp"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/modules"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

// GetGraph will return the dependency graph for the Go module that can be found at the specified
// path. The 'logger' parameter can be 'nil' which will result in no output or logging information
// being provided.
func GetGraph(log *zap.Logger, path string) (*Graph, error) {
	if log == nil {
		log = zap.NewNop()
	}
	log.Debug("Creating dependency graph.")

	mainModule, moduleInfo, err := modules.GetDependencies(log, path)
	if err != nil {
		return nil, err
	}

	g := NewGraph(log, path, mainModule)
	for _, module := range moduleInfo {
		g.addModule(module)
	}

	if err = g.buildImportGraph(); err != nil {
		return nil, err
	}

	if err = g.overlayModuleDependencies(); err != nil {
		return nil, err
	}

	for _, module := range g.Graph.GetLevel(0).List() {
		if module.Predecessors().Len() == 0 && module.Successors().Len() == 0 {
			g.log.Debug("Removing module as it not connected to the final graph.", zap.String("dependency", module.Name()))
			g.removeModule(module.Name())
		}
	}
	return g, nil
}
