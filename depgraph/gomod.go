package depgraph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

type Module struct {
	Path    string       // module path
	Version string       // module version
	Replace *Module      // replaced by this module
	Time    *time.Time   // time version was created
	Update  *Module      // available update, if any (with -u)
	Main    bool         // is this the main module?
	Error   *ModuleError // error loading module
}

type ModuleError struct {
	Err string // the error itself
}

// GetDepGraph should be called from within a Go module. It will return the dependency
// graph for this module.
func GetDepGraph(logger *logrus.Logger) (*DepGraph, error) {
	logger.Debug("Creating dependency graph.")
	main, modules, err := getSelectedModules(logger)
	if err != nil {
		return nil, err
	}

	rawDeps, err := runCommand(logger, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	graph := &DepGraph{
		logger: logger,
		module: main.Path,
		nodes:  map[string]*Node{},
	}
	if graph.logger == nil {
		graph.logger = logrus.New()
		graph.logger.SetOutput(ioutil.Discard)
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(rawDeps)), "\n") {
		logger.Debugf("Parsing dependency: %s", dep)

		depContent := depRE.FindStringSubmatch(dep)
		if len(depContent) == 0 {
			logger.Warnf("Ill-formed line in 'go mod graph' output: %s", dep)
			continue
		}

		beginNodeName, beginVersion := depContent[1], depContent[2]
		endNodeName, endVersion := depContent[3], depContent[4]

		beginNode := graph.nodes[beginNodeName]
		if beginNode == nil {
			beginNode, err = createNewNode(beginNodeName, modules)
			if err != nil {
				return nil, err
			}
			graph.nodes[beginNodeName] = beginNode
			logger.Debugf("Created new node: %+v", beginNode)
		}
		endNode := graph.nodes[endNodeName]
		if endNode == nil {
			endNode, err = createNewNode(endNodeName, modules)
			if err != nil {
				return nil, err
			}
			graph.nodes[endNodeName] = endNode
			logger.Debugf("Created new node: %+v", endNode)
		}

		if len(beginNode.SelectedVersion()) != 0 && beginNode.module.Replace == nil && beginNode.SelectedVersion() != beginVersion {
			logger.Warnf("Encountered unexpected version %q for edge starting at node %q.", beginVersion, beginNodeName)
		}
		newDependency := &Dependency{
			begin:   beginNodeName,
			end:     endNodeName,
			version: endVersion,
		}
		beginNode.successors = append(beginNode.successors, newDependency)
		endNode.predecessors = append(endNode.predecessors, newDependency)
		logger.Debugf("Created new dependency: %+v", newDependency)
	}
	return graph, nil
}

func createNewNode(name string, modules map[string]*Module) (*Node, error) {
	module := modules[name]
	if module == nil {
		return nil, fmt.Errorf("No module information for %q.", name)
	}
	return &Node{module: module}, nil
}

func getSelectedModules(logger *logrus.Logger) (*Module, map[string]*Module, error) {
	raw, err := runCommand(logger, "go", "list", "-json", "-m", "all")
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
		return nil, nil, errors.New("Could not determine main module.")
	}
	return &main, modules, nil
}
