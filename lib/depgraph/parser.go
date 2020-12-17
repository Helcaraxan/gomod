package depgraph

import (
	"regexp"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/modules"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

// GetModuleGraph will return the dependency graph for the Go module that can be found at the
// specified path. The 'logger' parameter can be 'nil' which will result in no output or logging
// information being provided.
func GetModuleGraph(log *zap.Logger, path string) (*ModuleGraph, error) {
	if log == nil {
		log = zap.NewNop()
	}
	log.Debug("Creating dependency graph.")

	mainModule, moduleInfo, err := modules.GetDependencies(log, path)
	if err != nil {
		return nil, err
	}

	graph := NewGraph(log, path, mainModule)
	for _, module := range moduleInfo {
		graph.AddModule(module)
	}

	if err = graph.buildImportGraph(); err != nil {
		return nil, err
	}

	if err = graph.overlayModuleDependencies(); err != nil {
		return nil, err
	}

	for _, dependency := range graph.Dependencies.List() {
		if dependency.Predecessors.Len() == 0 && dependency.Successors.Len() == 0 {
			graph.RemoveModule(dependency.Name())
		}
	}
	return graph, nil
}
