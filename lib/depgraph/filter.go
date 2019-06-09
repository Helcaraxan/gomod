package depgraph

// PruneUnsharedDeps returns a copy of the dependency graph with all nodes removed
// that are not part of a chain leading to a node with more than two predecessors.
func (g *DepGraph) PruneUnsharedDeps() *DepGraph {
	g.logger.Debug("Pruning dependencies that are not shared between multiple modules.")
	return g.prune(func(node *Node) bool {
		prune := len(node.successors) == 0 && len(node.predecessors) < 2
		if prune {
			g.logger.Debugf("Prune %+v.", node)
		} else {
			g.logger.Debugf("Not pruning: %+v.", node)
		}
		return prune
	})
}

// DependencyFilter allows to specify a dependency graph filter that removes any edges that
// are not part of a chain leading to this dependency. If a version is given then we only keep
// edges that prevent the use of the dependency at that given version due to the Go module's
// minimal version selection.
type DependencyFilter struct {
	Dependency string
	Version    string
}

// SubGraph returns a copy of the dependency graph with all nodes that are part of dependency chains
// that need to be modified for the specified dependency to be set to a given target version
// annotated as such.
func (g *DepGraph) SubGraph(filters []*DependencyFilter) *DepGraph {
	if len(filters) == 0 {
		return g
	}

	keep := map[string]struct{}{}
	for _, filter := range filters {
		keep = g.applyFilter(filter, keep)
	}

	g.logger.Debug("Pruning the dependency graph of irrelevant nodes.")
	subGraph := g.DeepCopy()
	for node := range g.nodes {
		if _, ok := keep[node]; !ok {
			g.logger.Debugf("Pruning %q.", node)
			subGraph.removeNode(node)
		}
	}
	return subGraph
}

func (g *DepGraph) applyFilter(filter *DependencyFilter, keep map[string]struct{}) map[string]struct{} {
	keep[filter.Dependency] = struct{}{}

	var todo []string
	if filter.Version != "" {
		g.logger.Debugf("Marking relevant subgraph for dependency %q at version %q.", filter.Dependency, filter.Version)
		for _, pred := range g.nodes[filter.Dependency].predecessors {
			_, visited := keep[pred.begin]
			if moduleMoreRecentThan(pred.RequiredVersion(), filter.Version) && pred.begin != g.module.Path && !visited {
				todo = append(todo, pred.begin)
				keep[pred.begin] = struct{}{}
			}
		}
	} else {
		g.logger.Debugf("Marking relevant subgraph for dependency %q.", filter.Dependency)
		todo = []string{filter.Dependency}
	}

	for len(todo) > 0 {
		for _, pred := range g.nodes[todo[0]].predecessors {
			if _, ok := keep[pred.begin]; !ok {
				keep[pred.begin] = struct{}{}
				todo = append(todo, pred.begin)
			}
		}
		todo = todo[1:]
	}
	return keep
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
