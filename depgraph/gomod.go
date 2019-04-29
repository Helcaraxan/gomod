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

var (
	depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)
)

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
	module, selectedVersions, replacements, err := getSelectedModules(logger)
	if err != nil {
		return nil, err
	}

	rawDeps, err := runCommand(logger, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	graph := &DepGraph{
		logger: logger,
		module: module,
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
			beginNode = &Node{
				name:            beginNodeName,
				replacement:     replacements[beginNodeName],
				selectedVersion: selectedVersions[beginNodeName],
			}
			graph.nodes[beginNodeName] = beginNode
			logger.Debugf("Created new node: %+v", beginNode)
		}
		endNode := graph.nodes[endNodeName]
		if endNode == nil {
			endNode = &Node{
				name:            endNodeName,
				replacement:     replacements[endNodeName],
				selectedVersion: selectedVersions[endNodeName],
			}
			graph.nodes[endNodeName] = endNode
			logger.Debugf("Created new node: %+v", endNode)
		}

		if len(beginNode.selectedVersion) != 0 && len(beginNode.replacement) == 0 && beginNode.selectedVersion != beginVersion {
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

func getSelectedModules(logger *logrus.Logger) (string, map[string]string, map[string]string, error) {
	raw, err := runCommand(logger, "go", "list", "-json", "-m", "all")
	if err != nil {
		return "", nil, nil, err
	}
	raw = bytes.ReplaceAll(bytes.TrimSpace(raw), []byte("\n}\n"), []byte("\n},\n"))
	raw = append([]byte("[\n"), raw...)
	raw = append(raw, []byte("\n]")...)

	var modules []Module
	if err = json.Unmarshal(raw, &modules); err != nil {
		return "", nil, nil, fmt.Errorf("Unable to retrieve information from 'go list': %v", err)
	}

	var mainModule string
	versionInfo, replacements := map[string]string{}, map[string]string{}
	for _, module := range modules {
		if module.Error != nil {
			logger.Warnf("Unable to retrieve information for module %q: %s", module.Path, module.Error.Err)
		}

		if module.Main {
			mainModule = module.Path
			continue
		}

		if module.Replace == nil {
			versionInfo[module.Path] = module.Version
			logger.Debugf("Found dependency %q selected at %q.", module.Path, module.Version)
		} else {
			replacements[module.Path] = module.Replace.Path
			versionInfo[module.Path] = module.Replace.Version
			logger.Debugf("Found dependency %q (replaced by %q) selected at %q.", module.Path, module.Replace.Path, module.Replace.Version)
		}
	}
	if len(mainModule) == 0 {
		return "", nil, nil, errors.New("Could not determine main module.")
	}
	return mainModule, versionInfo, replacements, nil
}
