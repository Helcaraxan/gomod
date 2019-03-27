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
		Logger: logger,
		Module: string(rawModule),
		Nodes:  map[string]*Node{},
	}
	if graph.Logger == nil {
		graph.Logger = logrus.New()
		graph.Logger.SetOutput(ioutil.Discard)
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(rawDeps)), "\n") {
		logger.Debugf("Parsing dependency: %s", dep)

		depContent := depRE.FindStringSubmatch(dep)
		if len(depContent) < 4 {
			logger.Warnf("Ill-formed line in 'go mod graph' output: %s", dep)
			continue
		}

		var beginNodeName, beginVersion string
		var endNodeName, endVersion string
		if len(depContent) == 5 {
			beginNodeName, beginVersion = depContent[1], depContent[2]
			endNodeName, endVersion = depContent[3], depContent[4]
		} else {
			beginNodeName = depContent[1]
			endNodeName, endVersion = depContent[2], depContent[3]
		}

		beginNode := graph.Nodes[beginNodeName]
		if beginNode == nil {
			beginNode = &Node{SelectedVersion: ModuleVersion(beginVersion)}
			graph.Nodes[beginNodeName] = beginNode
		}
		endNode := graph.Nodes[endNodeName]
		if endNode == nil {
			endNode = &Node{}
			graph.Nodes[endNodeName] = endNode
		}

		if len(beginNode.SelectedVersion) != 0 && beginNode.SelectedVersion != ModuleVersion(beginVersion) {
			logger.Warnf("Encountered unexpected version %q for edge starting at node %q.", beginVersion, beginNodeName)
		}
		newDependency := &Dependency{
			Begin:   beginNodeName,
			End:     endNodeName,
			Version: ModuleVersion(endVersion),
		}
		beginNode.Successors = append(beginNode.Successors, newDependency)
		endNode.Predecessors = append(endNode.Predecessors, newDependency)
	}
	return graph, nil
}
