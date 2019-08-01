package depgraph

// PruneUnsharedDeps returns a copy of the dependency graph with all nodes removed
// that are not part of a chain leading to a node with more than two predecessors.
func (g *DepGraph) PruneUnsharedDeps() *DepGraph {
	g.logger.Debug("Pruning dependencies that are not shared between multiple modules.")
	return g.prune(func(node *Dependency) bool {
		prune := node.Successors.Len() == 0 && node.Predecessors.Len() < 2
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
	for _, node := range g.Dependencies.List() {
		if _, ok := keep[node.Name()]; !ok {
			g.logger.Debugf("Pruning %q.", node.Name())
			subGraph.removeNode(node.Name())
		}
	}
	return subGraph
}

func (g *DepGraph) applyFilter(filter *DependencyFilter, keep map[string]struct{}) map[string]struct{} {
	filterNode, ok := g.Node(filter.Dependency)
	if !ok {
		return nil
	}
	filter.Dependency = filterNode.Module.Path

	keep[filter.Dependency] = struct{}{}

	var todo []string
	if filter.Version != "" {
		g.logger.Debugf("Marking relevant subgraph for dependency %q at version %q.", filter.Dependency, filter.Version)
		node, _ := g.Dependencies.Get(filter.Dependency)
		for _, pred := range node.Predecessors.List() {
			_, visited := keep[pred.Name()]
			if moduleMoreRecentThan(pred.VersionConstraint, filter.Version) && pred.Name() != g.Main.Name() && !visited {
				todo = append(todo, pred.Name())
				keep[pred.Name()] = struct{}{}
			}
		}
	} else {
		g.logger.Debugf("Marking relevant subgraph for dependency %q.", filter.Dependency)
		todo = []string{filter.Dependency}
	}

	for len(todo) > 0 {
		node, _ := g.Dependencies.Get(todo[0])
		for _, pred := range node.Predecessors.List() {
			if _, ok := keep[pred.Name()]; !ok {
				keep[pred.Name()] = struct{}{}
				todo = append(todo, pred.Name())
			}
		}
		todo = todo[1:]
	}
	return keep
}

func (g *DepGraph) prune(pruneFunc func(*Dependency) bool) *DepGraph {
	prunedGraph := g.DeepCopy()
	var done bool
	for !done {
		done = true
		for _, nodeReference := range prunedGraph.Dependencies.List() {
			if pruneFunc(nodeReference.Dependency) {
				done = false
				prunedGraph.removeNode(nodeReference.Name())
			}
		}
	}
	return prunedGraph
}

func (g *DepGraph) removeNode(name string) {
	g.logger.Debugf("Removing node with name %q.", name)
	if replaced, ok := g.replaces[name]; ok {
		delete(g.replaces, name)
		name = replaced
	}

	for replace, replaced := range g.replaces {
		if replaced == name {
			delete(g.replaces, replace)
		}
	}

	node, ok := g.Dependencies.Get(name)
	if !ok {
		return
	}
	for _, edge := range node.Successors.List() {
		g.removeEdge(node.Name(), edge.Name())
	}
	for _, edge := range node.Predecessors.List() {
		g.removeEdge(edge.Name(), node.Name())
	}
	g.Dependencies.Delete(name)
}

func (g *DepGraph) removeEdge(start string, end string) {
	g.logger.Debugf("Removing any edge between %q and %q.", start, end)
	if startNode, startOk := g.Node(start); startOk {
		startNode.Successors.Delete(end)
	}
	if endNode, endOk := g.Node(end); endOk {
		endNode.Predecessors.Delete(start)
	}
}
