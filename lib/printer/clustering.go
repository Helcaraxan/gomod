package printer

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

func computeGraphClusters(config *PrintConfig, graph *depgraph.Graph) *graphClusters {
	graphClusters := &graphClusters{
		graph:           graph,
		clusterMap:      map[string]*graphCluster{},
		cachedDepthMaps: map[string]map[string]int{},
	}

	hashToCluster := map[string]*graphCluster{}
	for _, node := range graph.Modules.List() {
		module := node.(*depgraph.ModuleReference).Module

		clusterHash := computeClusterHash(config, module)
		cluster := hashToCluster[clusterHash]
		if cluster == nil {
			cluster = newGraphCluster(clusterHash)
			hashToCluster[clusterHash] = cluster
			graphClusters.clusterList = append(graphClusters.clusterList, cluster)
		}
		cluster.members = append(cluster.members, module)
		graphClusters.clusterMap[module.Name()] = cluster
	}

	// Ensure determinism by sorting the modules in each cluster. The order that is used puts
	// nodes with no dependencies first and those with at least one last, the rest of the ordering
	// is done by alphabetical order.
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

func computeClusterHash(config *PrintConfig, node *depgraph.Module) string {
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
	graph       *depgraph.Graph
	clusterMap  map[string]*graphCluster
	clusterList []*graphCluster

	cachedDepthMaps map[string]map[string]int
}

func (c *graphClusters) getClusterDepthMap(nodeName string) map[string]int {
	if c.cachedDepthMaps[nodeName] == nil {
		c.cachedDepthMaps[nodeName] = c.computeClusterDepthMap(nodeName)
	}
	return c.cachedDepthMaps[nodeName]
}

func (c *graphClusters) computeClusterDepthMap(nodeName string) map[string]int {
	depthMap := map[string]int{}

	startNode, _ := c.graph.Modules.Get(nodeName)
	workStack := []*depgraph.Module{startNode.(*depgraph.ModuleReference).Module}
	workMap := map[string]int{nodeName: 0}
	pathLength := 0
	for len(workStack) > 0 {
		pathLength++
		curr := workStack[len(workStack)-1]
		if counter, ok := workMap[curr.Name()]; ok && counter > 0 { // Reached leaf of the DFS or cycle detected.
			workStack = workStack[:len(workStack)-1]
			pathLength--
			if counter == pathLength-1 { // Reached leaf of the DFS.
				delete(workMap, curr.Name())
			}
			continue
		}
		workMap[curr.Name()] = pathLength

		currentDepth := depthMap[curr.Name()]
		baseEdgeLength := c.clusterMap[curr.Name()].getHeight()
		for _, pred := range curr.Predecessors().List() {
			predNode, _ := c.graph.Modules.Get(pred.Name())
			edgeLength := baseEdgeLength + c.clusterMap[curr.Name()].getDepCount()/20 // Give bonus space for larger numbers of edges.
			if depthMap[pred.Name()] >= currentDepth+edgeLength {
				continue
			}
			depthMap[pred.Name()] = currentDepth + edgeLength
			if _, ok := workMap[pred.Name()]; !ok { // Only allow one instance of a node in the queue.
				workMap[pred.Name()] = 0
				workStack = append(workStack, predNode.(*depgraph.ModuleReference).Module)
			}
		}
	}
	c.cachedDepthMaps[nodeName] = depthMap
	return depthMap
}

type graphCluster struct {
	id      int
	hash    string
	members []*depgraph.Module

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
	if c.cachedDepCount < 0 {
		c.cachedDepCount = c.computeDepCount()
	}
	return c.cachedDepCount
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
	if c.cachedWidth < 0 {
		c.cachedWidth = c.computeWidth()
	}
	return c.cachedWidth
}

func (c *graphCluster) computeDepCount() int {
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

func (c *graphCluster) computeWidth() int {
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
