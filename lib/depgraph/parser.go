package depgraph

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/internal/util"
	"github.com/Helcaraxan/gomod/lib/modules"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

// GetDepGraph will return the dependency graph for the Go module that can be
// found at the specified path. The 'logger' parameter can be 'nil' which will
// result in no output or logging information being provided.
func GetDepGraph(logger *logrus.Logger, path string) (*DepGraph, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(ioutil.Discard)
	}
	logger.Debug("Creating dependency graph.")

	mainModule, moduleInfo, err := modules.GetDependencies(logger, path)
	if err != nil {
		return nil, err
	}

	graph := NewGraph(logger, path, mainModule)

	logger.Debug("Retrieving dependency information via 'go mod graph'")
	rawDeps, _, err := util.RunCommand(logger, path, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	for _, depString := range strings.Split(strings.TrimSpace(string(rawDeps)), "\n") {
		graph.logger.Debugf("Parsing dependency: %s", depString)
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
		g.logger.Debugf("Created new dependency %q: %+v", rawDependency.beginModule.Path, beginDependency)
	}
	endDependency, ok := g.GetDependency(rawDependency.endModule.Path)
	if !ok {
		if endDependency = g.AddDependency(rawDependency.endModule); endDependency == nil {
			return fmt.Errorf("could not create dependency based on %+v", rawDependency.endModule)
		}
		g.logger.Debugf("Created new dependency %q: %+v", rawDependency.endModule.Path, endDependency)
	}

	if len(beginDependency.SelectedVersion()) != 0 &&
		beginDependency.Module.Replace == nil &&
		beginDependency.SelectedVersion() != rawDependency.beginVersion {
		g.logger.Warnf(
			"Encountered unexpected version %q for dependency of %q on %q.",
			rawDependency.beginVersion,
			rawDependency.begineName,
			rawDependency.endName,
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
	g.logger.Debugf("Created new dependency from %q to %q with version %q.", beginDependency.Name(), endDependency.Name(), rawDependency.endVersion)
	return nil
}

type rawDependency struct {
	begineName   string
	beginVersion string
	beginModule  *modules.Module
	endName      string
	endVersion   string
	endModule    *modules.Module
}

func (g *DepGraph) parseDependency(depString string, moduleMap map[string]*modules.Module) (*rawDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		g.logger.Warnf("Skipping ill-formed line in 'go mod graph' output: %s", depString)
		return nil, false
	}

	beginName, beginVersion := depContent[1], depContent[2]
	endName, endVersion := depContent[3], depContent[4]

	beginModule := moduleMap[beginName]
	endModule := moduleMap[endName]
	if beginModule == nil || endModule == nil {
		g.logger.Warnf("Encountered a dependency edge starting or ending at an unknown module %q -> %q.", beginName, endName)
		return nil, false
	} else if beginVersion != beginModule.Version {
		g.logger.Debugf("Skipping edge from %q at %q to %q as we are not using that version.", beginName, beginVersion, endName)
		return nil, false
	}
	return &rawDependency{
		begineName:   beginName,
		beginVersion: beginVersion,
		beginModule:  beginModule,
		endName:      endName,
		endVersion:   endVersion,
		endModule:    endModule,
	}, true
}
