package depgraph

// PruneUnsharedDeps returns a copy of the dependency graph with all nodes removed
// that are not part of a chain leading to a node with more than two predecessors.
func (g *DepGraph) PruneUnsharedDeps() *DepGraph {
	return g.prune(func(node *Node) bool {
		prune := len(node.successors) == 0 && len(node.predecessors) < 2
		if prune {
			g.logger.Debugf("Prune %+v.", node)
		}
		return prune
	})
}

// SubGraph returns a copy of the dependency graph with all nodes removed that are
// not part of a chain leading to the specified dependency. Returns an empty graph
// if the specified depedendency does not exist in the graph.
func (g *DepGraph) SubGraph(dependency string) *DepGraph {
	if _, ok := g.nodes[dependency]; !ok {
		g.logger.Debugf("No node with name %q.", dependency)
		return &DepGraph{
			module: g.module,
			nodes:  map[string]*Node{},
		}
	}
	return g.prune(func(node *Node) bool {
		if len(node.successors) > 0 {
			return false
		}
		prune := len(node.predecessors) == 0 || node.predecessors[0].end != dependency
		if prune {
			g.logger.Debugf("Prune %+v.", node)
		}
		return prune
	})
}

// OffendingGraph returns a copy of the dependency graph with only nodes left that
// are part of dependency chains that need to be modified for the specified dependency
// to be set to a given target version.
func (g *DepGraph) OffendingGraph(dependency string, targetVersion string) *DepGraph {
	if _, ok := g.nodes[dependency]; !ok {
		g.logger.Debugf("No node with name %q.", dependency)
		return &DepGraph{
			module: g.module,
			nodes:  map[string]*Node{},
		}
	}
	offendingGraph := g.DeepCopy()
	for _, dep := range offendingGraph.nodes[dependency].predecessors {
		g.logger.Debugf("Dependency %q is required by %q in version %q.", dep.end, dep.begin, dep.version)
		if !dep.version.MoreRecentThan(ModuleVersion(targetVersion)) {
			offendingGraph.removeEdge(dep.begin, dep.end)
		}
	}
	return offendingGraph.SubGraph(dependency)
}

func (g *DepGraph) prune(pruneFunc func(*Node) bool) *DepGraph {
	prunedGraph := g.DeepCopy()
	var done bool
	for !done {
		done = true
		for name, node := range prunedGraph.nodes {
			if pruneFunc(node) {
				done = false
				prunedGraph.removeNode(name)
			}
		}
	}
	return prunedGraph
}

func (g *DepGraph) removeNode(name string) {
	g.logger.Debugf("Removing node with name %q.", name)
	node := g.nodes[name]
	if node == nil {
		return
	}

	for _, dep := range node.successors {
		g.removeEdge(dep.begin, dep.end)
	}
	for _, dep := range node.predecessors {
		g.removeEdge(dep.begin, dep.end)
	}
	delete(g.nodes, name)
}

func (g *DepGraph) removeEdge(start string, end string) {
	g.logger.Debugf("Removing any edge between %q and %q.", start, end)
	startNode := g.nodes[start]
	endNode := g.nodes[end]
	if startNode == nil || endNode == nil {
		return
	}

	var newSuccessors []*Dependency
	for _, candidate := range startNode.successors {
		if candidate.end != end {
			newSuccessors = append(newSuccessors, candidate)
		}
	}
	startNode.successors = newSuccessors

	var newPredecessors []*Dependency
	for _, candidate := range endNode.predecessors {
		if candidate.begin != start {
			newPredecessors = append(newPredecessors, candidate)
		}
	}
	endNode.predecessors = newPredecessors
}
