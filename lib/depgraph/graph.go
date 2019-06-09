package depgraph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/internal/util"
)

type Module struct {
	Main    bool         // is this the main module?
	Path    string       // module path
	Replace *Module      // replaced by this module
	Version string       // module version
	Time    *time.Time   // time version was created
	Update  *Module      // available update, if any (with -u)
	GoMod   string       // the path to this module's go.mod file
	Error   *ModuleError // error loading module
}

type ModuleError struct {
	Err string // the error itself
}

func createNewNode(name string, modules map[string]*Module) (*Node, error) {
	module := modules[name]
	if module == nil {
		return nil, fmt.Errorf("No module information for %q.", name)
	}
	return &Node{Module: module}, nil
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
		return nil, nil, errors.New("Could not determine main module.")
	}
	return &main, modules, nil
}

// DepGraph represents a Go module's dependency graph.
type DepGraph struct {
	logger *logrus.Logger
	module string
	nodes  map[string]*Node
}

// NewGraph returns a new DepGraph instance which will use the specified
// logger for writing log output. If nil a null-logger will be used instead.
func NewGraph(logger *logrus.Logger) *DepGraph {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(ioutil.Discard)
	}
	return &DepGraph{
		logger: logger,
		nodes:  map[string]*Node{},
	}
}

// Module returns the name of the module to which this DepGraph instance applies.
func (g *DepGraph) Module() string {
	return g.module
}

// Nodes returns a slice with copies of all nodes belonging to this DepGraph
// instance. These Node copies can be interacted with without modifying the
// underlying DepGraph.
func (g *DepGraph) Nodes() []Node {
	var idx int
	nodes := make([]Node, len(g.nodes))
	for _, node := range g.nodes {
		nodes[idx] = *node
		idx++
	}
	return nodes
}

// Node represents a module in a Go module's dependency graph.
type Node struct {
	Module       *Module
	predecessors []*Dependency
	successors   []*Dependency
}

// Name of the module represented by this Node in the DepGraph instance.
func (n *Node) Name() string {
	return n.Module.Path
}

// SelectedVersion corresponds to the version of the dependency represented by
// this Node which was selected for use.
func (n *Node) SelectedVersion() string {
	if n.Module.Replace != nil {
		return n.Module.Replace.Version
	}
	return n.Module.Version
}

func (n *Node) Timestamp() *time.Time {
	if n.Module.Replace != nil {
		return n.Module.Replace.Time
	}
	return n.Module.Time
}

// Predecessors returns a slice with copies of all the incoming Dependencies for
// this  Node. These Dependency copies can be interacted with without modifying
// the underlying DepGraph.
func (n *Node) Predecessors() []Dependency {
	var idx int
	predecessors := make([]Dependency, len(n.predecessors))
	for _, predecessor := range n.predecessors {
		predecessors[idx] = *predecessor
		idx++
	}
	return predecessors
}

// Successors returns a slice with copies of all the outgoing Dependencies for
// this  Node. These Dependency copies can be interacted with without modifying
// the underlying DepGraph.
func (n *Node) Successors() []Dependency {
	var idx int
	successors := make([]Dependency, len(n.successors))
	for _, successor := range n.successors {
		successors[idx] = *successor
		idx++
	}
	return successors
}

// Dependency represents a dependency in a DepGraph instance.
type Dependency struct {
	begin     string
	end       string
	version   string
	offending bool
}

// Begin returns the name of the Go module at which this Dependency originates.
func (d *Dependency) Begin() string {
	return d.begin
}

// End returns the name of the Go module which this Dependency requires.
func (d *Dependency) End() string {
	return d.end
}

// RequiredVersion is the minimal required version of the Go module which this
// Dependency requires.
func (d *Dependency) RequiredVersion() string {
	return d.version
}

// DeepCopy returns a separate copy of the current dependency graph that can be
// safely modified without affecting the original graph. The logger argument can
// be nil in which case nothing will be logged.
func (g *DepGraph) DeepCopy() *DepGraph {
	newGraph := &DepGraph{
		logger: g.logger,
		module: g.module,
		nodes:  map[string]*Node{},
	}
	for name, node := range g.nodes {
		nodeCopy := *node
		nodeCopy.successors = nil
		nodeCopy.predecessors = nil
		newGraph.nodes[name] = &nodeCopy
	}

	for name, node := range g.nodes {
		for _, successor := range node.successors {
			dependencyCopy := *successor
			newGraph.nodes[name].successors = append(newGraph.nodes[name].successors, &dependencyCopy)
			newGraph.nodes[successor.end].predecessors = append(newGraph.nodes[successor.end].predecessors, &dependencyCopy)
		}
	}
	g.logger.Debug("Created a deep copy of graph.")
	return newGraph
}
