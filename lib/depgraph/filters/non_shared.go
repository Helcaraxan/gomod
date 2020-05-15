package filters

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

type NonSharedDependencies struct {
	Excludes []string
}

func (f *NonSharedDependencies) Apply(log *zap.Logger, graph *depgraph.DepGraph) *depgraph.DepGraph {
	log.Debug("Pruning dependencies that are not shared between multiple modules.")
	if len(f.Excludes) > 0 {
		log.Debug("Excluding dependency from prune.", zap.Strings("exclude-list", f.Excludes))
	}

	excludeMap := make(map[string]struct{}, len(f.Excludes))
	for idx := range f.Excludes {
		excludeMap[f.Excludes[idx]] = struct{}{}
	}

	prunedGraph := graph.DeepCopy()
	for {
		// Find the next unshared dependency.
		var target *depgraph.DependencyReference
		for _, dependency := range prunedGraph.Dependencies.List() {
			_, ok := excludeMap[dependency.Name()]
			if !ok && len(dependency.Successors.List()) == 0 && len(dependency.Predecessors.List()) <= 1 {
				target = dependency
				break
			}
		}
		if target == nil {
			return prunedGraph
		}

		// Walk-up any chain of non-shared dependencies starting from the target one and prune them.
		pruneUnsharedChain(prunedGraph, excludeMap, target)
	}
}

func pruneUnsharedChain(graph *depgraph.DepGraph, excludeMap map[string]struct{}, leaf *depgraph.DependencyReference) {
	for {
		if len(leaf.Predecessors.List()) == 0 {
			graph.RemoveDependency(leaf.Name())
			return
		}
		newLeaf := leaf.Predecessors.List()[0]
		graph.RemoveDependency(leaf.Name())
		_, ok := excludeMap[newLeaf.Name()]
		if ok || len(newLeaf.Successors.List()) != 0 || len(newLeaf.Predecessors.List()) > 1 {
			return
		}
		leaf = newLeaf
	}
}
