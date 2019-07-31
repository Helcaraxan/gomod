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
	Module  string
	Version string
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

func (f *DependencyFilter) matchesFilter(dependency *NodeReference) bool {
	if f.Version == "" {
		return true
	}
	return moduleMoreRecentThan(dependency.VersionConstraint, f.Version)
}

func (g *DepGraph) applyFilter(filter *DependencyFilter, keep map[string]struct{}) map[string]struct{} {
	filterNode, ok := g.Node(filter.Module)
	if !ok {
		return nil
	}

	if keep == nil {
		keep = map[string]struct{}{}
	}
	keep[filterNode.Name()] = struct{}{}

	g.logger.Debugf("Marking subgraph for dependency %q.", filter.Module)
	if filter.Version != "" {
		g.logger.Debugf("Only considering dependencies preventing use of version %q.", filter.Version)
	}
	var todo []*NodeReference
	for _, predecessor := range filterNode.Predecessors.List() {
		if filter.matchesFilter(predecessor) {
			todo = append(todo, predecessor)
			keep[predecessor.Name()] = struct{}{}
		}
	}

	for len(todo) > 0 {
		node := todo[0]
		for _, predecessor := range node.Predecessors.List() {
			if _, ok := keep[predecessor.Name()]; !ok {
				keep[predecessor.Name()] = struct{}{}
				todo = append(todo, predecessor)
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
	for _, successor := range node.Successors.List() {
		successor.Predecessors.Delete(node.Name())
	}
	for _, predecessor := range node.Predecessors.List() {
		predecessor.Successors.Delete(node.Name())
	}
	g.Dependencies.Delete(name)
}
