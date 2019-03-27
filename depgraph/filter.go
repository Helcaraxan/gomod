package depgraph

// PruneUnsharedDeps returns a copy of the dependency graph with all nodes removed
// that are not part of a chain leading to a node with more than two predecessors.
func (g *DepGraph) PruneUnsharedDeps() *DepGraph {
	return g.prune(func(node *Node) bool {
		prune := len(node.Successors) == 0 && len(node.Predecessors) < 2
		if prune {
			g.Logger.Debugf("Prune %+v.", node)
		}
		return prune
	})
}

// SubGraph returns a copy of the dependency graph with all nodes removed that are
// not part of a chain leading to the specified dependency. Returns an empty graph
// if the specified depedendency does not exist in the graph.
func (g *DepGraph) SubGraph(dependency string) *DepGraph {
	if _, ok := g.Nodes[dependency]; !ok {
		g.Logger.Debugf("No node with name %q.", dependency)
		return &DepGraph{
			Module: g.Module,
			Nodes:  map[string]*Node{},
		}
	}
	return g.prune(func(node *Node) bool {
		if len(node.Successors) > 0 {
			return false
		}
		prune := len(node.Predecessors) == 0 || node.Predecessors[0].End != dependency
		if prune {
			g.Logger.Debugf("Prune %+v.", node)
		}
		return prune
	})
}

// OffendingGraph returns a copy of the dependency graph with only nodes left that
// are part of dependency chains that need to be modified for the specified dependency
// to be set to a given target version.
func (g *DepGraph) OffendingGraph(dependency string, targetVersion string) *DepGraph {
	if _, ok := g.Nodes[dependency]; !ok {
		g.Logger.Debugf("No node with name %q.", dependency)
		return &DepGraph{
			Module: g.Module,
			Nodes:  map[string]*Node{},
		}
	}
	offendingGraph := g.DeepCopy()
	for _, dep := range offendingGraph.Nodes[dependency].Predecessors {
		g.Logger.Debugf("Dependency %q is required by %q in version %q.", dep.End, dep.Begin, dep.Version)
		if !dep.Version.MoreRecentThan(ModuleVersion(targetVersion)) {
			offendingGraph.removeEdge(dep.Begin, dep.End)
		}
	}
	return offendingGraph.SubGraph(dependency)
}

func (g *DepGraph) prune(pruneFunc func(*Node) bool) *DepGraph {
	prunedGraph := g.DeepCopy()
	var done bool
	for !done {
		done = true
		for name, node := range prunedGraph.Nodes {
			if pruneFunc(node) {
				done = false
				prunedGraph.removeNode(name)
			}
		}
	}
	return prunedGraph
}

func (g *DepGraph) removeNode(name string) {
	g.Logger.Debugf("Removing node with name %q.", name)
	node := g.Nodes[name]
	if node == nil {
		return
	}

	for _, dep := range node.Successors {
		g.removeEdge(dep.Begin, dep.End)
	}
	for _, dep := range node.Predecessors {
		g.removeEdge(dep.Begin, dep.End)
	}
	delete(g.Nodes, name)
}

func (g *DepGraph) removeEdge(start string, end string) {
	g.Logger.Debugf("Removing any edge between %q and %q.", start, end)
	startNode := g.Nodes[start]
	endNode := g.Nodes[end]
	if startNode == nil || endNode == nil {
		return
	}

	var newSuccessors []*Dependency
	for _, candidate := range startNode.Successors {
		if candidate.End != end {
			newSuccessors = append(newSuccessors, candidate)
		}
	}
	startNode.Successors = newSuccessors

	var newPredecessors []*Dependency
	for _, candidate := range endNode.Predecessors {
		if candidate.Begin != start {
			newPredecessors = append(newPredecessors, candidate)
		}
	}
	endNode.Predecessors = newPredecessors
}
