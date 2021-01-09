package printer

import (
	"fmt"
	"math"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/depgraph"
	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/util"
)

type Level uint8

const (
	LevelModules Level = iota
	LevelPackages
)

// PrintConfig allows for the specification of parameters that should be passed to the Print
// function of a Graph.
type PrintConfig struct {
	// Logger that should be used to show progress while printing the Graph.
	Log *logger.Logger

	// Which level of granularity to print the graph at (modules, packages).
	Granularity Level

	// Annotate edges and nodes with their respective versions.
	Annotate bool
	// Path at which the printed version of the Graph should be stored. If set to a nil-string a
	// temporary file will be created.
	OutputPath string
	// Options for generating a visual representation of the Graph. If the field is non-nil, print
	// out an image file using GraphViz, if false print out the graph in DOT format.
	Style *StyleOptions
}

type StyleOptions struct {
	// Scale nodes according to the number of their successors and predecssors.
	ScaleNodes bool
	// Level at which to cluster nodes in the printed graph. This can be very beneficial for larger
	// dependency graphs that might be unreadable with the default settings.
	Cluster ClusterLevel
}

// Level at which to performing clustering when generating the image of the
// dependency graph.
type ClusterLevel int

const (
	// No clustering. Each node is printed as is.
	Off ClusterLevel = iota
	// Cluster nodes that have the same parent.
	Parent
	// Cluster nodes that all have the same, unique, predecessor in the graph.
	Shared
	// Cluster nodes that all have the same (group of) predecessor(s) in the graph.
	Full
)

// Print takes in a PrintConfig struct and dumps the content of a HierarchicalDigraph instance
// according to parameters.
func Print(g *graph.HierarchicalDigraph, config *PrintConfig) error {
	var err error
	out := os.Stdout
	if len(config.OutputPath) > 0 {
		if out, err = util.PrepareOutputPath(config.Log, config.OutputPath); err != nil {
			return err
		}
		defer func() {
			_ = out.Close()
		}()
		config.Log.Debug("Writing DOT graph.", zap.String("path", config.OutputPath))
	} else {
		config.Log.Debug("Writing DOT graph to terminal.")
	}

	fileContent := []string{
		"strict digraph {",
	}
	fileContent = append(fileContent, determineGlobalOptions(g, config)...)

	clusters := computeGraphClusters(g, config)
	for _, cluster := range clusters.clusterList {
		fileContent = append(fileContent, printClusterToDot(cluster, config))
	}

	for _, node := range g.GetLevel(int(config.Granularity)).List() {
		fileContent = append(fileContent, printEdgesToDot(config, node, clusters)...)
	}

	fileContent = append(fileContent, "}")

	if _, err = out.WriteString(strings.Join(fileContent, "\n") + "\n"); err != nil {
		config.Log.Error("Failed to write DOT file.", zap.Error(err))
		return fmt.Errorf("could not write to %q", out.Name())
	}
	return nil
}

func determineGlobalOptions(g *graph.HierarchicalDigraph, config *PrintConfig) []string {
	globalOptions := []string{
		"  node [shape=box,style=rounded]",
		"  start=0", // Needed for placement determinism.
	}

	if config.Annotate {
		globalOptions = append(globalOptions, "  concentrate=true")
	} else {
		// Unfortunately we cannot use the "concentrate" option with 'ortho' splines as it leads to segfaults on large graphs.
		globalOptions = append(
			globalOptions,
			"  splines=ortho", // By far the most readable form of splines on larger graphs but incompatible with annotations.
		)
	}

	if config.Style != nil {
		if config.Style.Cluster > Off {
			globalOptions = append(
				globalOptions,
				"  graph [style=rounded]",
				"  compound=true", // Needed for edges targeted at subgraphs.
			)
		}
		if config.Style.ScaleNodes {
			rankSep := math.Log10(float64(g.GetLevel(int(config.Granularity)).Len())) - 1
			if rankSep < 0.3 {
				rankSep = 0.3
			}
			globalOptions = append(globalOptions, fmt.Sprintf("  ranksep=%.2f", rankSep))
		}
	}

	return globalOptions
}

func printClusterToDot(cluster *graphCluster, config *PrintConfig) string {
	if len(cluster.members) == 0 {
		config.Log.Warn("Found an empty node cluster associated with.", zap.String("cluster", cluster.name()), zap.String("hash", cluster.hash))
		return ""
	} else if len(cluster.members) == 1 {
		return printNodeToDot(config, cluster.members[0])
	}

	dot := "  subgraph " + cluster.name() + "{\n"
	for _, node := range cluster.members {
		dot += "  " + printNodeToDot(config, node) + "\n"
	}

	// Print invisible nodes and edges that help node placement by forcing a grid layout.
	dot += "    // The nodes and edges part of this subgraph defined below are only used to\n"
	dot += "    // improve node placement but do not reflect actual dependencies.\n"
	dot += "    node [style=invis]\n"
	dot += "    edge [style=invis,minlen=1]\n"
	dot += "    graph [color=blue]\n" //nolint:misspell

	rowSize := cluster.getWidth()
	firstRowSize := len(cluster.members) % rowSize
	firstRowOffset := (rowSize - firstRowSize) / 2
	if firstRowSize > 0 {
		for idx := 0; idx < firstRowOffset; idx++ {
			dot += fmt.Sprintf("    \"%s_%d\"\n", cluster.name(), idx)
			dot += fmt.Sprintf("    \"%s_%d\" -> \"%s\"\n", cluster.name(), idx, cluster.members[idx+firstRowSize].Name())
		}
		for idx := firstRowOffset + firstRowSize; idx < rowSize; idx++ {
			dot += fmt.Sprintf("    \"%s_%d\"\n", cluster.name(), idx)
			dot += fmt.Sprintf("    \"%s_%d\" -> \"%s\"\n", cluster.name(), idx, cluster.members[idx+firstRowSize].Name())
		}
	}
	for idx := 0; idx < firstRowSize; idx++ {
		dot += fmt.Sprintf("    \"%s\" -> \"%s\"\n", cluster.members[idx].Name(), cluster.members[idx+firstRowOffset+firstRowSize].Name())
	}
	for idx := firstRowSize; idx < len(cluster.members); idx++ {
		if idx+rowSize < len(cluster.members) {
			dot += fmt.Sprintf("   \"%s\" -> \"%s\"\n", cluster.members[idx].Name(), cluster.members[idx+rowSize].Name())
		}
	}
	return dot + "  }"
}

type annotated interface {
	NodeAttributes(annotate bool) []string
	EdgeAttributes(target graph.Node, annotate bool) []string
}

var (
	_ annotated = &depgraph.Module{}
	//_ annotated = &depgraph.Package{}
)

func printNodeToDot(config *PrintConfig, node graph.Node) string {
	var nodeOptions []string
	if config.Style != nil && config.Style.ScaleNodes {
		scaling := math.Log2(float64(node.Predecessors().Len()+node.Successors().Len())) / 5
		if scaling < 0.1 {
			scaling = 0.1
		}
		nodeOptions = append(nodeOptions, fmt.Sprintf("width=%.2f,height=%.2f", 5*scaling, scaling))
	}

	if a, ok := node.(annotated); ok {
		nodeOptions = append(nodeOptions, a.NodeAttributes(config.Annotate)...)
	}

	dot := "  \"" + node.Name() + "\""
	if len(nodeOptions) > 0 {
		dot += " [" + strings.Join(nodeOptions, ",") + "]"
	}
	return dot
}

func printEdgesToDot(config *PrintConfig, node graph.Node, clusters *graphClusters) []string {
	clustersReached := map[int]struct{}{}

	var dots []string
	for _, dep := range node.Successors().List() {
		cluster, ok := clusters.clusterMap[dep.Hash()]
		if !ok {
			config.Log.Error("No cluster reference found for dependency.", zap.String("node", node.Hash()), zap.String("dep", dep.Hash()))
			continue
		} else if _, ok = clustersReached[cluster.id]; ok {
			continue
		}
		clustersReached[cluster.id] = struct{}{}

		target := dep.Name()
		var edgeAnnotations []string
		if minLength := clusters.clusterDepthMap(dep.Hash())[node.Hash()]; minLength > 1 {
			edgeAnnotations = append(edgeAnnotations, fmt.Sprintf("minlen=%d", minLength))
		}

		annotate := config.Annotate
		if len(cluster.members) > 1 {
			annotate = false
			target = cluster.getRepresentative()
			edgeAnnotations = append(edgeAnnotations, "lhead=\""+cluster.name()+"\"")
		}

		if a, ok := node.(annotated); ok {
			edgeAnnotations = append(edgeAnnotations, a.EdgeAttributes(dep, annotate)...)
		}

		dot := "  \"" + node.Name() + "\" -> \"" + target + "\""
		if len(edgeAnnotations) > 0 {
			dot += " [" + strings.Join(edgeAnnotations, ",") + "]"
		}
		dots = append(dots, dot)
	}
	return dots
}
