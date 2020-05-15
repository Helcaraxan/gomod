package depgraph

import (
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/internal/util"
	"github.com/Helcaraxan/gomod/lib/modules"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

// GetDepGraph will return the dependency graph for the Go module that can be
// found at the specified path. The 'logger' parameter can be 'nil' which will
// result in no output or logging information being provided.
func GetDepGraph(log *zap.Logger, path string) (*DepGraph, error) {
	if log == nil {
		log = zap.NewNop()
	}
	log.Debug("Creating dependency graph.")

	mainModule, moduleInfo, err := modules.GetDependencies(log, path)
	if err != nil {
		return nil, err
	}

	graph := NewGraph(log, path, mainModule)

	log.Debug("Retrieving dependency information via 'go mod graph'")
	rawDeps, _, err := util.RunCommand(log, path, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	for _, depString := range strings.Split(strings.TrimSpace(string(rawDeps)), "\n") {
		graph.log.Debug("Parsing dependency", zap.String("reference", depString))
		rawDep, ok := graph.parseDependency(depString, moduleInfo)
		if !ok {
			continue
		}
		if err = graph.addDependency(rawDep); err != nil {
			return nil, err
		}
	}
	for _, dependency := range graph.Dependencies.List() {
		if dependency.Predecessors.Len() == 0 && dependency.Successors.Len() == 0 {
			graph.RemoveDependency(dependency.Name())
		}
	}
	return graph, nil
}

func (g *DepGraph) addDependency(rawDependency *rawDependency) error {
	beginDependency, ok := g.GetDependency(rawDependency.beginModule.Path)
	if !ok {
		if beginDependency = g.AddDependency(rawDependency.beginModule); beginDependency == nil {
			return fmt.Errorf("could not create dependency based on %+v", rawDependency.beginModule)
		}
		g.log.Debug("Created new dependency.", zap.String("source", rawDependency.beginModule.Path), zap.Any("dependency", beginDependency))
	}
	endDependency, ok := g.GetDependency(rawDependency.endModule.Path)
	if !ok {
		if endDependency = g.AddDependency(rawDependency.endModule); endDependency == nil {
			return fmt.Errorf("could not create dependency based on %+v", rawDependency.endModule)
		}
		g.log.Debug("Created new dependency.", zap.String("target", rawDependency.endModule.Path), zap.Any("dependency", endDependency))
	}

	if beginDependency.SelectedVersion() != "" &&
		beginDependency.Module.Replace == nil &&
		beginDependency.SelectedVersion() != rawDependency.beginVersion {
		g.log.Warn(
			"Encountered unexpected version for a dependency.",
			zap.String("version", rawDependency.beginVersion),
			zap.String("source", rawDependency.beginName),
			zap.String("target", rawDependency.endName),
		)
	}
	beginDependency.Successors.Add(&DependencyReference{
		Dependency:        endDependency,
		VersionConstraint: rawDependency.endVersion,
	})
	endDependency.Predecessors.Add(&DependencyReference{
		Dependency:        beginDependency,
		VersionConstraint: rawDependency.endVersion,
	})
	g.log.Debug(
		"Created new dependency.",
		zap.String("version", rawDependency.endVersion),
		zap.String("source", beginDependency.Name()),
		zap.String("target", endDependency.Name()),
	)
	return nil
}

type rawDependency struct {
	beginName    string
	beginVersion string
	beginModule  *modules.Module
	endName      string
	endVersion   string
	endModule    *modules.Module
}

func (g *DepGraph) parseDependency(depString string, moduleMap map[string]*modules.Module) (*rawDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		g.log.Warn("Skipping ill-formed line in 'go mod graph' output.", zap.String("line", depString))
		return nil, false
	}

	beginName, beginVersion := depContent[1], depContent[2]
	endName, endVersion := depContent[3], depContent[4]

	beginModule := moduleMap[beginName]
	endModule := moduleMap[endName]
	if beginModule == nil || endModule == nil {
		g.log.Warn(
			"Encountered a dependency edge starting or ending at an unknown module.",
			zap.String("source", beginName),
			zap.String("target", endName),
		)
		return nil, false
	} else if beginVersion != beginModule.Version {
		g.log.Debug(
			"Skipping edge as we are not using the specified version.",
			zap.String("source", beginName),
			zap.String("version", beginVersion),
			zap.String("target", endName),
		)
		return nil, false
	}
	return &rawDependency{
		beginName:    beginName,
		beginVersion: beginVersion,
		beginModule:  beginModule,
		endName:      endName,
		endVersion:   endVersion,
		endModule:    endModule,
	}, true
}
