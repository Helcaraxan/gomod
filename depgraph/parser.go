package depgraph

import (
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
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

	logger.Info("Retrieving dependency information via 'go mod graph'")
	rawDeps, err := runCommand(logger, quiet, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	graph := &DepGraph{
		logger: logger,
		module: main.Path,
		nodes:  map[string]*Node{},
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

		if beginModule := modules[beginNodeName]; beginModule == nil {
			logger.Warnf("Encountered a dependency edge starting at an unknown module %q.", beginNodeName)
			continue
		} else if beginVersion != beginModule.Version {
			logger.Debugf("Skipping edge from %q at %q to %q as we are not using that version.", beginNodeName, beginVersion, endNodeName)
			continue
		} else if endModule := modules[endNodeName]; endModule == nil {
			logger.Warnf("Encountered a dependency edge ending at an unknown module %q.", endNodeName)
			continue
		}

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
			logger.Warnf("Encountered unexpected version %q for dependency of %q on %q.", beginVersion, beginNodeName, endNodeName)
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
