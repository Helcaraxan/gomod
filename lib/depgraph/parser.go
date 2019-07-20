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
	for _, node := range graph.nodes {
		if len(node.predecessors) == 0 && len(node.successors) == 0 {
			graph.removeNode(node.Name())
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
	var ok bool
	beginNode := g.Node(rawDependency.beginModule.Path)
	if beginNode == nil {
		if beginNode, ok = g.AddNode(rawDependency.beginModule); !ok {
			return fmt.Errorf("could not create node based on %+v", beginNode.Module)
		}
		g.logger.Debugf("Created new node %q: %+v", rawDependency.beginModule.Path, beginNode)
	}
	endNode := g.Node(rawDependency.endModule.Path)
	if endNode == nil {
		if endNode, ok = g.AddNode(rawDependency.endModule); !ok {
			return fmt.Errorf("could not create node based on %+v", beginNode.Module)
		}
		g.logger.Debugf("Created new node %q: %+v", rawDependency.endModule.Path, endNode)
	}

	if len(beginNode.SelectedVersion()) != 0 && beginNode.Module.Replace == nil && beginNode.SelectedVersion() != rawDependency.beginVersion {
		g.logger.Warnf(
			"Encountered unexpected version %q for dependency of %q on %q.",
			rawDependency.beginVersion,
			rawDependency.begineNodeName,
			rawDependency.endNodeName,
		)
	}
	newDependency := &Dependency{
		begin:   beginNode.Module.Path,
		end:     endNode.Module.Path,
		version: rawDependency.endVersion,
	}
	beginNode.successors = append(beginNode.successors, newDependency)
	endNode.predecessors = append(endNode.predecessors, newDependency)
	g.logger.Debugf("Created new dependency: %+v", newDependency)
	return nil
}

type rawDependency struct {
	begineNodeName string
	beginVersion   string
	beginModule    *Module
	endNodeName    string
	endVersion     string
	endModule      *Module
}

func (g *DepGraph) parseDependency(depString string, modules map[string]*Module) (*rawDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		g.logger.Warnf("Skipping ill-formed line in 'go mod graph' output: %s", depString)
		return nil, false
	}

	beginNodeName, beginVersion := depContent[1], depContent[2]
	endNodeName, endVersion := depContent[3], depContent[4]

	beginModule := modules[beginNodeName]
	endModule := modules[endNodeName]
	if beginModule == nil || endModule == nil {
		g.logger.Warnf("Encountered a dependency edge starting or ending at an unknown module %q -> %q.", beginNodeName, endNodeName)
		return nil, false
	} else if beginVersion != beginModule.Version {
		g.logger.Debugf("Skipping edge from %q at %q to %q as we are not using that version.", beginNodeName, beginVersion, endNodeName)
		return nil, false
	}
	return &rawDependency{
		begineNodeName: beginNodeName,
		beginVersion:   beginVersion,
		beginModule:    beginModule,
		endNodeName:    endNodeName,
		endVersion:     endVersion,
		endModule:      endModule,
	}, true
}
