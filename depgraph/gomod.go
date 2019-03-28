package depgraph

import (
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

var depRE = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)

// GetDepGraph should be called from within a Go module. It will return the dependency
// graph for this module.
func GetDepGraph(logger *logrus.Logger) (*DepGraph, error) {
	logger.Debug("Creating dependency graph.")
	rawModule, err := runCommand(logger, "go", "list", "-m")
	if err != nil {
		return nil, err
	}

	rawDeps, err := runCommand(logger, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}

	graph := &DepGraph{
		logger: logger,
		module: string(rawModule),
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

		var beginNodeName, beginVersion string
		var endNodeName, endVersion string
		if len(depContent[2]) == 0 {
			beginNodeName = depContent[1]
			endNodeName, endVersion = depContent[2], depContent[3]
		} else {
			beginNodeName, beginVersion = depContent[1], depContent[2]
			endNodeName, endVersion = depContent[3], depContent[4]
		}

		beginNode := graph.nodes[beginNodeName]
		if beginNode == nil {
			beginNode = &Node{selectedVersion: ModuleVersion(beginVersion)}
			graph.nodes[beginNodeName] = beginNode
		}
		endNode := graph.nodes[endNodeName]
		if endNode == nil {
			endNode = &Node{}
			graph.nodes[endNodeName] = endNode
		}

		if len(beginNode.selectedVersion) != 0 && beginNode.selectedVersion != ModuleVersion(beginVersion) {
			logger.Warnf("Encountered unexpected version %q for edge starting at node %q.", beginVersion, beginNodeName)
		}
		newDependency := &Dependency{
			begin:   beginNodeName,
			end:     endNodeName,
			version: ModuleVersion(endVersion),
		}
		beginNode.successors = append(beginNode.successors, newDependency)
		endNode.predecessors = append(endNode.predecessors, newDependency)
	}
	return graph, nil
}
