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
	nodeReference, ok := g.nodes.Get(name)
	if !ok {
		return nil, false
	}
	return nodeReference.Node, true
}

func (g *DepGraph) Nodes() *NodeMap {
	return g.nodes
}

func (g *DepGraph) AddNode(module *Module) (*Node, bool) {
	if module == nil {
		return nil, false
	}
	if nodeReference, ok := g.nodes.Get(module.Path); ok && nodeReference != nil {
		return nodeReference.Node, true
	}
	newNodeReference := &NodeReference{
		Node: &Node{
			Module:       module,
			Predecessors: NewNodeMap(),
			Successors:   NewNodeMap(),
		},
		VersionConstraint: module.Version,
	}
	g.nodes.Add(newNodeReference)
	if module.Replace != nil {
		g.replaces[module.Replace.Path] = module.Path
	}
	return newNodeReference.Node, true
}

func (g *DepGraph) Depth() int {
	if g.main != nil {
		return 1
	}

	var maxDepth int
	todo := []*Node{g.Main()}
	depthMap := map[string]int{g.main.Name(): 1}
	for len(todo) > 0 {
		depth := depthMap[todo[0].Name()]
		for _, succ := range todo[0].Successors.List() {
			if depth+1 > depthMap[succ.Name()] {
				depthMap[succ.Name()] = depth + 1
				nodeReference, ok := g.nodes.Get(succ.Name())
				if !ok {
					g.logger.Errorf("Encountered an edge to an non-existent node '%s'.", succ.Name())
				} else {
					todo = append(todo, nodeReference.Node)
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
	Predecessors *NodeMap
	Successors   *NodeMap
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
		newNode, _ := newGraph.Node(node.Name())
		for _, predecessor := range node.Predecessors.List() {
			newPredecessor, ok := newGraph.Node(predecessor.Name())
			if !ok {
				g.logger.Warnf("Could not find node for '%s' listed in predecessors of '%s'.", predecessor.Name(), node.Name())
				continue
			}
			newNode.Predecessors.Add(&NodeReference{
				Node:              newPredecessor,
				VersionConstraint: predecessor.VersionConstraint,
			})
		}
		for _, successor := range node.Successors.List() {
			newSuccessor, ok := newGraph.Node(successor.Name())
			if !ok {
				g.logger.Warnf("Could not find node for '%s' listed in successors of '%s'.", successor.Name(), node.Name())
				continue
			}
			newNode.Successors.Add(&NodeReference{
				Node:              newSuccessor,
				VersionConstraint: successor.VersionConstraint,
			})
		}
	}

	for original, replacement := range g.replaces {
		newGraph.replaces[original] = replacement
	}

	g.logger.Debug("Created a deep copy of graph.")
	return newGraph
}
