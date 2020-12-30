package filters

import (
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/depgraph"
	"github.com/Helcaraxan/gomod/internal/graph"
)

type NonSharedModules struct {
	Excludes []string
}

func (f *NonSharedModules) Apply(log *zap.Logger, g *depgraph.Graph) *depgraph.Graph {
	log.Debug("Pruning modules that only have one predecessor in the dependency graph.")
	if len(f.Excludes) > 0 {
		log.Debug("Excluding dependency from prune.", zap.Strings("exclude-list", f.Excludes))
	}

	excludeMap := make(map[string]struct{}, len(f.Excludes))
	for idx := range f.Excludes {
		excludeMap[f.Excludes[idx]] = struct{}{}
	}

	for {
		// Find the next unshared dependency.
		var target graph.Node
		for _, node := range g.Graph.GetLevel(0).List() {
			_, ok := excludeMap[node.Name()]
			if !ok && len(node.Successors().List()) == 0 && len(node.Predecessors().List()) <= 1 {
				target = node
				break
			}
		}
		if target == nil {
			return g
		}

		// Walk-up any chain of non-shared dependencies starting from the target one and prune them.
		pruneUnsharedChain(g, excludeMap, target)
	}
}

func pruneUnsharedChain(g *depgraph.Graph, excludeMap map[string]struct{}, leaf graph.Node) {
	for {
		if len(leaf.Predecessors().List()) == 0 {
			_ = g.Graph.DeleteNode(leaf.Hash())
			return
		}
		newLeaf := leaf.Predecessors().List()[0].(*depgraph.Module)
		_ = g.Graph.DeleteNode(leaf.Hash())
		_, ok := excludeMap[newLeaf.Name()]
		if ok || len(newLeaf.Successors().List()) != 0 || len(newLeaf.Predecessors().List()) > 1 {
			return
		}
		leaf = newLeaf
	}
}
