package depgraph

import (
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
	logger   *logrus.Logger
	main     *Node
	nodes    *NodeMap
	replaces map[string]string
}

// NewGraph returns a new DepGraph instance which will use the specified
// logger for writing log output. If nil a null-logger will be used instead.
func NewGraph(logger *logrus.Logger, main *Module) *DepGraph {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(ioutil.Discard)
	}
	newGraph := &DepGraph{
		logger:   logger,
		nodes:    NewNodeMap(),
		replaces: map[string]string{},
	}
	mainNode, _ := newGraph.AddNode(main)
	newGraph.main = mainNode
	return newGraph
}

func (g *DepGraph) Main() *Node {
	return g.main
}

func (g *DepGraph) Node(name string) (*Node, bool) {
	if replaced, ok := g.replaces[name]; ok {
		name = replaced
	}
	return g.nodes.Get(name)
}

func (g *DepGraph) Nodes() *NodeMap {
	return g.nodes
}

func (g *DepGraph) AddNode(module *Module) (*Node, bool) {
	if module == nil {
		return nil, false
	}
	if node, ok := g.nodes.Get(module.Path); ok && node != nil {
		return node, true
	}
	newNode := &Node{Module: module}
	g.nodes.Add(newNode)
	if module.Replace != nil {
		g.replaces[module.Replace.Path] = module.Path
	}
	return newNode, true
}

func (g *DepGraph) Depth() int {
	if g.main != nil {
		return 1
	}

	var maxDepth int
	todo := []*Node{g.main}
	depthMap := map[string]int{g.main.Name(): 1}
	for len(todo) > 0 {
		depth := depthMap[todo[0].Name()]
		for _, succ := range todo[0].successors {
			if depth+1 > depthMap[succ.End()] {
				depthMap[succ.End()] = depth + 1
				node, ok := g.nodes.Get(succ.End())
				if !ok {
					g.logger.Errorf("Encountered an edge to an non-existent node '%s'.", succ.End())
				} else {
					todo = append(todo, node)
				}
			}
		}
		if depth > maxDepth {
			maxDepth = depth
		}
		todo = todo[1:]
	}
	return maxDepth
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
	predecessors := make([]Dependency, 0, len(n.predecessors))
	for _, predecessor := range n.predecessors {
		predecessors = append(predecessors, *predecessor)
	}
	return predecessors
}

// Successors returns a slice with copies of all the outgoing Dependencies for
// this  Node. These Dependency copies can be interacted with without modifying
// the underlying DepGraph.
func (n *Node) Successors() []Dependency {
	successors := make([]Dependency, 0, len(n.successors))
	for _, successor := range n.successors {
		successors = append(successors, *successor)
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
	g.logger.Debugf("Deep-copying dependency graph for %q.", g.Main().Name())

	newGraph := NewGraph(g.logger, g.main.Module)
	for name, node := range g.nodes.List() {
		if _, ok := newGraph.AddNode(node.Module); !ok {
			g.logger.Errorf("Encountered an empty node for %q.", name)
		}
	}

	for _, node := range g.nodes.List() {
		beginNode, _ := newGraph.Node(node.Name())
		for _, successor := range node.successors {
			endNode, _ := newGraph.Node(successor.End())
			dependencyCopy := *successor
			beginNode.successors = append(beginNode.successors, &dependencyCopy)
			endNode.predecessors = append(endNode.predecessors, &dependencyCopy)
		}
	}
	g.logger.Debug("Created a deep copy of graph.")
	return newGraph
}
