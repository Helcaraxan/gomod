package depgraph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/internal/util"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

// GetDepGraph should be called from within a Go module. It will return the dependency
// graph for this module. The 'logger' parameter can be 'nil' which will result in no
// output or logging information to be provided.
func GetDepGraph(logger *logrus.Logger) (*DepGraph, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(ioutil.Discard)
	}
	logger.Debug("Creating dependency graph.")

	mainModule, modules, err := getSelectedModules(logger)
	if err != nil {
		return nil, err
	}

	graph := NewGraph(logger, mainModule)

	logger.Debug("Retrieving dependency information via 'go mod graph'")
	rawDeps, _, err := util.RunCommand(logger, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	for _, depString := range strings.Split(strings.TrimSpace(string(rawDeps)), "\n") {
		graph.logger.Debugf("Parsing dependency: %s", depString)
		rawDep, ok := graph.parseDependency(depString, modules)
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

func getSelectedModules(logger *logrus.Logger) (*Module, map[string]*Module, error) {
	logger.Debug("Retrieving module information via 'go list'")
	raw, _, err := util.RunCommand(logger, "go", "list", "-json", "-m", "all")
	if err != nil {
		return nil, nil, err
	}
	raw = bytes.ReplaceAll(bytes.TrimSpace(raw), []byte("\n}\n"), []byte("\n},\n"))
	raw = append([]byte("[\n"), raw...)
	raw = append(raw, []byte("\n]")...)

	var moduleList []*Module
	if err = json.Unmarshal(raw, &moduleList); err != nil {
		return nil, nil, fmt.Errorf("Unable to retrieve information from 'go list': %v", err)
	}

	var main *Module
	modules := map[string]*Module{}
	for idx, module := range moduleList {
		if module.Error != nil {
			logger.Warnf("Unable to retrieve information for module %q: %s", module.Path, module.Error.Err)
		}

		if module.Main {
			main = module
		}
		modules[module.Path] = moduleList[idx]
		if module.Replace != nil {
			modules[module.Replace.Path] = moduleList[idx]
		}
	}
	if main == nil || len(main.Path) == 0 {
		return nil, nil, errors.New("could not determine main module")
	}
	return main, modules, nil
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
	beginModule  *Module
	endName      string
	endVersion   string
	endModule    *Module
}

func (g *DepGraph) parseDependency(depString string, modules map[string]*Module) (*rawDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		g.logger.Warnf("Skipping ill-formed line in 'go mod graph' output: %s", depString)
		return nil, false
	}

	beginName, beginVersion := depContent[1], depContent[2]
	endName, endVersion := depContent[3], depContent[4]

	beginModule := modules[beginName]
	endModule := modules[endName]
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
