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
func GetDepGraph(logger *logrus.Logger, quiet bool) (*DepGraph, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(ioutil.Discard)
	}
	logger.Debug("Creating dependency graph.")

	main, modules, err := getSelectedModules(logger, quiet)
	if err != nil {
		return nil, err
	}

	graph := &DepGraph{
		logger:  logger,
		module:  main,
		modules: modules,
		nodes:   map[string]*Node{},
	}

	logger.Debug("Retrieving dependency information via 'go mod graph'")
	rawDeps, err := util.RunCommand(logger, quiet, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(rawDeps)), "\n") {
		if err = graph.addDependency(dep); err != nil {
			return nil, err
		}
	}
	return graph, nil
}

func getSelectedModules(logger *logrus.Logger, quiet bool) (*Module, map[string]*Module, error) {
	logger.Debug("Retrieving module information via 'go list'")
	raw, err := util.RunCommand(logger, quiet, "go", "list", "-json", "-m", "all")
	if err != nil {
		return nil, nil, err
	}
	raw = bytes.ReplaceAll(bytes.TrimSpace(raw), []byte("\n}\n"), []byte("\n},\n"))
	raw = append([]byte("[\n"), raw...)
	raw = append(raw, []byte("\n]")...)

	var moduleList []Module
	if err = json.Unmarshal(raw, &moduleList); err != nil {
		return nil, nil, fmt.Errorf("Unable to retrieve information from 'go list': %v", err)
	}

	var main Module
	modules := map[string]*Module{}
	for idx, module := range moduleList {
		if module.Error != nil {
			logger.Warnf("Unable to retrieve information for module %q: %s", module.Path, module.Error.Err)
		}

		if module.Main {
			main = module
		}
		modules[module.Path] = &moduleList[idx]
	}
	if len(main.Path) == 0 {
		return nil, nil, errors.New("could not determine main module")
	}
	return &main, modules, nil
}

func (g *DepGraph) addDependency(depString string) error {
	g.logger.Debugf("Parsing dependency: %s", depString)

	parsedDep, ok := g.parseDependency(depString)
	if !ok {
		return nil
	}

	var err error
	beginNode := g.nodes[parsedDep.beginNodeName]
	if beginNode == nil {
		if beginNode, err = g.createNewNode(parsedDep.beginNodeName); err != nil {
			return err
		}
		g.nodes[parsedDep.beginNodeName] = beginNode
		g.logger.Debugf("Created new node: %+v", beginNode)
	}
	endNode := g.nodes[parsedDep.endNodeName]
	if endNode == nil {
		if endNode, err = g.createNewNode(parsedDep.endNodeName); err != nil {
			return err
		}
		g.nodes[parsedDep.endNodeName] = endNode
		g.logger.Debugf("Created new node: %+v", endNode)
	}

	if len(beginNode.SelectedVersion()) != 0 && beginNode.Module.Replace == nil && beginNode.SelectedVersion() != parsedDep.beginVersion {
		g.logger.Warnf(
			"Encountered unexpected version %q for dependency of %q on %q.",
			parsedDep.beginVersion,
			parsedDep.beginNodeName,
			parsedDep.endNodeName,
		)
	}
	newDependency := &Dependency{
		begin:   parsedDep.beginNodeName,
		end:     parsedDep.endNodeName,
		version: parsedDep.endVersion,
	}
	beginNode.successors = append(beginNode.successors, newDependency)
	endNode.predecessors = append(endNode.predecessors, newDependency)
	g.logger.Debugf("Created new dependency: %+v", newDependency)
	return nil
}

type rawDependency struct {
	beginNodeName string
	beginVersion  string
	endNodeName   string
	endVersion    string
}

func (g *DepGraph) parseDependency(depString string) (*rawDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		g.logger.Warnf("Skipping ill-formed line in 'go mod graph' output: %s", depString)
		return nil, false
	}

	beginNodeName, beginVersion := depContent[1], depContent[2]
	endNodeName, endVersion := depContent[3], depContent[4]

	beginModule := g.modules[beginNodeName]
	endModule := g.modules[endNodeName]
	if beginModule == nil || endModule == nil {
		g.logger.Warnf("Encountered a dependency edge starting or ending at an unknown module %q -> %q.", beginNodeName, endNodeName)
		return nil, false
	} else if beginVersion != beginModule.Version {
		g.logger.Debugf("Skipping edge from %q at %q to %q as we are not using that version.", beginNodeName, beginVersion, endNodeName)
		return nil, false
	}
	return &rawDependency{
		beginNodeName: beginNodeName,
		beginVersion:  beginVersion,
		endNodeName:   endNodeName,
		endVersion:    endVersion,
	}, true
}
