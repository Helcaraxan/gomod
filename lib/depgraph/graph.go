package depgraph

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
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

// DepGraph represents a Go module's dependency graph.
type DepGraph struct {
	logger  *logrus.Logger
	Module  *Module
	Modules map[string]*Module
	Nodes   map[string]*Node
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
		Nodes:  map[string]*Node{},
	}
}

func (g *DepGraph) Name() string {
	return g.Module.Path
}

func (g *DepGraph) createNewNode(name string) (*Node, error) {
	module := g.Modules[name]
	if module == nil {
		return nil, fmt.Errorf("no module information for %q", name)
	}
	return &Node{Module: module}, nil
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
	begin   string
	end     string
	version string
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
		logger:  g.logger,
		Module:  g.Module,
		Modules: g.Modules,
		Nodes:   map[string]*Node{},
	}
	for name, node := range g.Nodes {
		nodeCopy := *node
		nodeCopy.successors = nil
		nodeCopy.predecessors = nil
		newGraph.Nodes[name] = &nodeCopy
	}

	for name, node := range g.Nodes {
		for _, successor := range node.successors {
			dependencyCopy := *successor
			newGraph.Nodes[name].successors = append(newGraph.Nodes[name].successors, &dependencyCopy)
			newGraph.Nodes[successor.end].predecessors = append(newGraph.Nodes[successor.end].predecessors, &dependencyCopy)
		}
	}
	g.logger.Debug("Created a deep copy of graph.")
	return newGraph
}
