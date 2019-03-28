package depgraph

import (
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	depRE  = regexp.MustCompile(`^([^@\s]+)@?([^@\s]+)? ([^@\s]+)@([^@\s]+)$`)
	listRE = regexp.MustCompile(`^([^\s]+) ([^\s]+)(?: => ([^\s]+) ([^\s]+))?$`)
)

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
	raw, err := runCommand(logger, "go", "list", "-m", "all")
	if err != nil {
		return "", nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	module := lines[0]
	logger.Debugf("Found module %q.", module)
	versionInfo, replacements := map[string]string{}, map[string]string{}
	for _, line := range lines[1:] {
		dependencyInfo := listRE.FindStringSubmatch(line)
		if len(dependencyInfo) == 0 {
			logger.Warnf("Unexpected output from 'go list -m all': %s", line)
		}
		if len(dependencyInfo[3]) == 0 {
			versionInfo[dependencyInfo[1]] = dependencyInfo[2]
			logger.Debugf("Found dependency %q selected at %q.", dependencyInfo[1], dependencyInfo[2])
		} else {
			replacements[dependencyInfo[1]] = dependencyInfo[3]
			versionInfo[dependencyInfo[1]] = dependencyInfo[4]
			logger.Debugf("Found dependency %q (replaced by %q) selected at %q.", dependencyInfo[1], dependencyInfo[3], dependencyInfo[4])
		}
	}
	return module, versionInfo, replacements, nil
}
