package printer

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/Helcaraxan/gomod/internal/graph"
)

func computeGraphClusters(g *graph.HierarchicalDigraph, config *PrintConfig) *graphClusters {
	graphClusters := &graphClusters{
		graph:           g,
		level:           config.Granularity,
		clusterMap:      map[string]*graphCluster{},
		cachedDepthMaps: map[string]map[string]int{},
	}

	hashToCluster := map[string]*graphCluster{}
	for _, node := range g.GetLevel(int(config.Granularity)).List() {
		clusterHash := computeClusterHash(config, node)
		cluster := hashToCluster[clusterHash]
		if cluster == nil {
			cluster = newGraphCluster(clusterHash)
			hashToCluster[clusterHash] = cluster
			graphClusters.clusterList = append(graphClusters.clusterList, cluster)
		}
		cluster.members = append(cluster.members, node)
		graphClusters.clusterMap[node.Hash()] = cluster
	}

	// Ensure determinism by sorting the nodes in each cluster. The order that is used puts nodes
	// with no dependencies first and those with at least one last, the rest of the ordering is done
	// by alphabetical order.
	for hash := range hashToCluster {
		cluster := hashToCluster[hash]
		sort.Slice(cluster.members, func(i int, j int) bool {
			hasDepsI := cluster.members[i].Successors().Len() > 0
			hasDepsJ := cluster.members[j].Successors().Len() > 0
			if (hasDepsI && !hasDepsJ) || (!hasDepsI && hasDepsJ) {
				return hasDepsJ
			}
			return cluster.members[i].Name() < cluster.members[j].Name()
		})
	}
	sort.Slice(graphClusters.clusterList, func(i int, j int) bool {
		return graphClusters.clusterList[i].hash < graphClusters.clusterList[j].hash
	})
	return graphClusters
}

func computeClusterHash(config *PrintConfig, node graph.Node) string {
	var hashElements []string
	for _, pred := range node.Predecessors().List() {
		hashElements = append(hashElements, nodeNameToHash(pred.Name()))
	}
	sort.Strings(hashElements)
	hash := strings.Join(hashElements, "_")

	// Depending on the configuration we need to generate more or less unique cluster names.
	if config.Style == nil || config.Style.Cluster == Off || (config.Style.Cluster == Shared && node.Predecessors().Len() > 1) {
		hash = node.Name() + "_from_" + hash
	}
	return hash
}

type graphClusters struct {
	graph *graph.HierarchicalDigraph
	level Level

	clusterMap  map[string]*graphCluster
	clusterList []*graphCluster

	cachedDepthMaps map[string]map[string]int
}

func (c *graphClusters) clusterDepthMap(nodeHash string) map[string]int {
	if m, ok := c.cachedDepthMaps[nodeHash]; ok {
		return m
	}

	depthMap := map[string]int{}
	levelNodes := c.graph.GetLevel(int(c.level))

	startNode, _ := levelNodes.Get(nodeHash)
	workStack := []graph.Node{startNode}
	workMap := map[string]int{nodeHash: 0}
	pathLength := 0
	for len(workStack) > 0 {
		pathLength++
		curr := workStack[len(workStack)-1]
		if counter, ok := workMap[curr.Hash()]; ok && counter > 0 { // Reached leaf of the DFS or cycle detected.
			workStack = workStack[:len(workStack)-1]
			pathLength--
			if counter == pathLength-1 { // Reached leaf of the DFS.
				delete(workMap, curr.Hash())
			}
			continue
		}
		workMap[curr.Hash()] = pathLength

		currentDepth := depthMap[curr.Hash()]
		baseEdgeLength := c.clusterMap[curr.Hash()].getHeight()
		for _, pred := range curr.Predecessors().List() {
			predNode, _ := levelNodes.Get(pred.Hash())
			edgeLength := baseEdgeLength + c.clusterMap[curr.Hash()].getDepCount()/20 // Give bonus space for larger numbers of edges.
			if depthMap[pred.Hash()] >= currentDepth+edgeLength {
				continue
			}
			depthMap[pred.Hash()] = currentDepth + edgeLength
			if _, ok := workMap[pred.Hash()]; !ok { // Only allow one instance of a node in the queue.
				workMap[pred.Hash()] = 0
				workStack = append(workStack, predNode)
			}
		}
	}
	c.cachedDepthMaps[nodeHash] = depthMap

	return depthMap
}

type graphCluster struct {
	id      int
	hash    string
	members []graph.Node

	cachedDepCount int
	cachedWidth    int
}

var clusterIDCounter int

func newGraphCluster(hash string) *graphCluster {
	clusterIDCounter++
	return &graphCluster{
		id:             clusterIDCounter,
		hash:           hash,
		cachedDepCount: -1,
		cachedWidth:    -1,
	}
}

var alphaNumericalRange = []*unicode.RangeTable{unicode.Letter, unicode.Number}

func (c *graphCluster) name() string {
	if len(c.members) > 1 {
		return "cluster_" + c.hash
	}
	return c.hash
}

func (c *graphCluster) getRepresentative() string {
	if len(c.members) == 0 {
		return ""
	}
	return c.members[c.getWidth()/2].Name()
}

func (c *graphCluster) getDepCount() int {
	if c.cachedDepCount >= 0 {
		return c.cachedDepCount
	}

	var depCount int
	for idx := len(c.members) - 1; idx >= 0; idx-- {
		if c.members[idx].Successors().Len() == 0 {
			break
		}
		depCount += c.members[idx].Successors().Len()
	}
	c.cachedDepCount = depCount
	return depCount
}

func (c *graphCluster) getHeight() int {
	width := c.getWidth()
	heigth := len(c.members) / width
	if len(c.members)%width != 0 {
		heigth++
	}
	if heigth > 1 {
		heigth++
	}
	return heigth
}

func (c *graphCluster) getWidth() int {
	if c.cachedWidth >= 0 {
		return c.cachedWidth
	}

	membersWithDeps := 1
	for membersWithDeps < len(c.members) && c.members[len(c.members)-1-membersWithDeps].Successors().Len() > 0 {
		membersWithDeps++
	}

	clusterWidth := int(math.Floor(math.Sqrt(float64(len(c.members)))))
	if membersWithDeps > clusterWidth {
		clusterWidth = membersWithDeps
	}
	c.cachedWidth = clusterWidth
	return clusterWidth
}

func nodeNameToHash(nodeName string) string {
	var hash string
	for _, c := range nodeName {
		if unicode.IsOneOf(alphaNumericalRange, c) {
			hash += string(c)
		} else {
			hash += "_"
		}
	}
	return hash
}
