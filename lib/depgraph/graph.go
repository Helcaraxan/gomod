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
	Main         *Dependency
	Dependencies *NodeMap

	logger   *logrus.Logger
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
		logger:       logger,
		Dependencies: NewNodeMap(),
		replaces:     map[string]string{},
	}
	newGraph.Main, _ = newGraph.AddNode(main)
	return newGraph
}

func (g *DepGraph) Node(name string) (*Dependency, bool) {
	if replaced, ok := g.replaces[name]; ok {
		name = replaced
	}
	nodeReference, ok := g.Dependencies.Get(name)
	if !ok {
		return nil, false
	}
	return nodeReference.Dependency, true
}

func (g *DepGraph) AddNode(module *Module) (*Dependency, bool) {
	if module == nil {
		return nil, false
	}
	if nodeReference, ok := g.Dependencies.Get(module.Path); ok && nodeReference != nil {
		return nodeReference.Dependency, true
	}
	newNodeReference := &NodeReference{
		Dependency: &Dependency{
			Module:       module,
			Predecessors: NewNodeMap(),
			Successors:   NewNodeMap(),
		},
		VersionConstraint: module.Version,
	}
	g.Dependencies.Add(newNodeReference)
	if module.Replace != nil {
		g.replaces[module.Replace.Path] = module.Path
	}
	return newNodeReference.Dependency, true
}

func (g *DepGraph) Depth() int {
	if g.Main != nil {
		return 1
	}

	var maxDepth int
	todo := []*Dependency{g.Main}
	depthMap := map[string]int{g.Main.Name(): 1}
	for len(todo) > 0 {
		depth := depthMap[todo[0].Name()]
		for _, succ := range todo[0].Successors.List() {
			if depth+1 > depthMap[succ.Name()] {
				depthMap[succ.Name()] = depth + 1
				nodeReference, ok := g.Dependencies.Get(succ.Name())
				if !ok {
					g.logger.Errorf("Encountered an edge to an non-existent node '%s'.", succ.Name())
				} else {
					todo = append(todo, nodeReference.Dependency)
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

// Dependency represents a module in a Go module's dependency graph.
type Dependency struct {
	Module       *Module
	Predecessors *NodeMap
	Successors   *NodeMap
}

// Name of the module represented by this Node in the DepGraph instance.
func (n *Dependency) Name() string {
	return n.Module.Path
}

// SelectedVersion corresponds to the version of the dependency represented by
// this Node which was selected for use.
func (n *Dependency) SelectedVersion() string {
	if n.Module.Replace != nil {
		return n.Module.Replace.Version
	}
	return n.Module.Version
}

func (n *Dependency) Timestamp() *time.Time {
	if n.Module.Replace != nil {
		return n.Module.Replace.Time
	}
	return n.Module.Time
}

// DeepCopy returns a separate copy of the current dependency graph that can be
// safely modified without affecting the original graph. The logger argument can
// be nil in which case nothing will be logged.
func (g *DepGraph) DeepCopy() *DepGraph {
	g.logger.Debugf("Deep-copying dependency graph for %q.", g.Main.Name())

	newGraph := NewGraph(g.logger, g.Main.Module)
	for name, node := range g.Dependencies.List() {
		if _, ok := newGraph.AddNode(node.Module); !ok {
			g.logger.Errorf("Encountered an empty node for %q.", name)
		}
	}

	for _, node := range g.Dependencies.List() {
		newNode, _ := newGraph.Node(node.Name())
		for _, predecessor := range node.Predecessors.List() {
			newPredecessor, ok := newGraph.Node(predecessor.Name())
			if !ok {
				g.logger.Warnf("Could not find node for '%s' listed in predecessors of '%s'.", predecessor.Name(), node.Name())
				continue
			}
			newNode.Predecessors.Add(&NodeReference{
				Dependency:        newPredecessor,
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
				Dependency:        newSuccessor,
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
