package depgraph

// PruneUnsharedDeps returns a copy of the dependency graph with all dependencies removed
// that are not part of a chain leading to a dependency with more than two predecessors.
func (g *DepGraph) PruneUnsharedDeps() *DepGraph {
	g.logger.Debug("Pruning dependencies that are not shared between multiple modules.")
	return g.prune(func(dependency *Dependency) bool {
		prune := dependency.Successors.Len() == 0 && dependency.Predecessors.Len() < 2
		if prune {
			g.logger.Debugf("Prune %+v.", dependency)
		} else {
			g.logger.Debugf("Not pruning: %+v.", dependency)
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

// SubGraph returns a copy of the dependency graph with all dependencies that are part of chains
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

	g.logger.Debug("Pruning the dependency graph of irrelevant paths.")
	subGraph := g.DeepCopy()
	for _, dependency := range g.Dependencies.List() {
		if _, ok := keep[dependency.Name()]; !ok {
			g.logger.Debugf("Pruning %q.", dependency.Name())
			subGraph.RemoveDependency(dependency.Name())
		}
	}
	return subGraph
}

func (f *DependencyFilter) matchesFilter(dependency *DependencyReference) bool {
	if f.Version == "" {
		return true
	}
	return moduleMoreRecentThan(dependency.VersionConstraint, f.Version)
}

func (g *DepGraph) applyFilter(filter *DependencyFilter, keep map[string]struct{}) map[string]struct{} {
	filterModule, ok := g.GetDependency(filter.Module)
	if !ok {
		return nil
	}

	if keep == nil {
		keep = map[string]struct{}{}
	}
	keep[filterModule.Name()] = struct{}{}

	g.logger.Debugf("Marking subgraph for dependency %q.", filter.Module)
	if filter.Version != "" {
		g.logger.Debugf("Only considering dependencies preventing use of version %q.", filter.Version)
	}
	var todo []*DependencyReference
	for _, predecessor := range filterModule.Predecessors.List() {
		if filter.matchesFilter(predecessor) {
			todo = append(todo, predecessor)
			keep[predecessor.Name()] = struct{}{}
		}
	}

	for len(todo) > 0 {
		dependency := todo[0]
		for _, predecessor := range dependency.Predecessors.List() {
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
		for _, dependencyReference := range prunedGraph.Dependencies.List() {
			if pruneFunc(dependencyReference.Dependency) {
				done = false
				prunedGraph.RemoveDependency(dependencyReference.Name())
			}
		}
	}
	return prunedGraph
}
