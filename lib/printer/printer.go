package printer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/depgraph"
	"github.com/Helcaraxan/gomod/lib/internal/util"
)

type Format int

const (
	FormatUnknown Format = iota
	FormatPDF
	FormatPNG
	FormatPS
	FormatJPG
	FormatGIF
)

var (
	FormatToString = map[Format]string{
		FormatPDF: "pdf",
		FormatPNG: "png",
		FormatPS:  "ps",
		FormatJPG: "jpg",
		FormatGIF: "gif",
	}
	StringToFormat = map[string]Format{
		"pdf": FormatPDF,
		"png": FormatPNG,
		"ps":  FormatPS,
		"jpg": FormatJPG,
		"gif": FormatGIF,
	}
)

// PrintConfig allows for the specification of parameters that should be passed
// to the Print function of a DepGraph.
type PrintConfig struct {
	// Logger that should be used to show progress while printing the DepGraph.
	Logger *logrus.Logger

	// Annotate edges and nodes with their respective versions.
	Annotate bool
	// Options for generating a visual representation of the DepGraph. If the
	// field is non-nil, print out an image file using GraphViz, if false print
	// out the graph in DOT format.
	Style *StyleOptions

	// Force overwriting of pre-existing files at the specified OutputPath.
	Force bool
	// Path at which the printed version of the DepGraph should be stored. If
	// set to a nil-string a temporary file will be created.
	OutputPath string
	// OutputFormat to use when writing files with the 'dot' tool.
	OutputFormat Format
}

type StyleOptions struct {
	// Scale nodes according to their number of dependencies of the module they
	// represent and the number of modules depending on it.
	ScaleNodes bool
	// Level at which to cluster nodes in the printed graph. This can be very
	// beneficial for larger dependency graphs that might be unreadable with the
	// default settings.
	Cluster ClusterLevel
}

// Level at which to performing clustering when generating the image of the
// dependency graph.
type ClusterLevel int

const (
	// No clustering. Each node is printed as is.
	Off ClusterLevel = iota
	// Cluster non-shared dependencies to reduce the complexity and size of the
	// graph.
	Shared
	// Cluster non-shared dependencies as well as any group of node that all
	// share the same predecessors in the graph.
	Full
)

// Print takes in a PrintConfig struct and dumps the content of this DepGraph
// instance according to parameters.
func Print(graph *depgraph.DepGraph, config *PrintConfig) error {
	var printer func(*depgraph.DepGraph, *PrintConfig) error
	if config.Style != nil {
		printer = PrintToVisual
	} else {
		printer = PrintToDOT
	}
	return printer(graph, config)
}

// PrintToVisual creates an image file at the specified target path that represents the dependency graph.
func PrintToVisual(graph *depgraph.DepGraph, config *PrintConfig) error {
	tempDir, err := ioutil.TempDir("", "depgraph")
	if err != nil {
		config.Logger.WithError(err).Error("Could not create a temporary directory.")
	}
	defer func() {
		config.Logger.Debugf("Cleaning up temporary output folder %q.", tempDir)
		_ = os.RemoveAll(tempDir)
	}()
	config.Logger.Debugf("Using temporary output folder %q.", tempDir)

	if len(config.OutputPath) == 0 {
		if config.OutputFormat == FormatUnknown {
			config.OutputFormat = FormatPNG
		}
		config.OutputPath = fmt.Sprintf("graph." + FormatToString[config.OutputFormat])
	} else {
		if config.OutputFormat == FormatUnknown {
			config.OutputFormat = StringToFormat[filepath.Ext(config.OutputPath)[1:]]
		} else if filepath.Ext(config.OutputPath) != "."+FormatToString[config.OutputFormat] {
			config.Logger.Errorf(
				"The given output file's extension '%s' does not match the specified output format '%s'.",
				filepath.Base(config.OutputPath),
				FormatToString[config.OutputFormat],
			)
			return errors.New("mismatched output filename and specified output format")
		}
	}
	if config.OutputFormat == FormatUnknown {
		config.Logger.Error("Could not determine the output format from either the specified output path or format.")
	}

	out, err := util.PrepareOutputPath(config.Logger, config.OutputPath, config.Force)
	if err != nil {
		return err
	}
	_ = out.Close() // Will be written by the 'dot' tool.

	dotPrintConfig := *config
	dotPrintConfig.OutputPath = filepath.Join(tempDir, "out.dot")
	if err = PrintToDOT(graph, &dotPrintConfig); err != nil {
		return err
	}

	config.Logger.Debugf("Generating %q.", config.OutputPath)
	_, _, err = util.RunCommand(config.Logger, "dot", "-T"+FormatToString[config.OutputFormat], "-o"+config.OutputPath, dotPrintConfig.OutputPath)
	return err
}

func PrintToDOT(graph *depgraph.DepGraph, config *PrintConfig) error {
	var err error
	out := os.Stdout
	if len(config.OutputPath) > 0 {
		if out, err = util.PrepareOutputPath(config.Logger, config.OutputPath, config.Force); err != nil {
			return err
		}
		defer func() {
			_ = out.Close()
		}()
		config.Logger.Debugf("Writing DOT graph to %q.", config.OutputPath)
	} else {
		config.Logger.Debug("Writing DOT graph to terminal.")
	}

	fileContent := []string{
		"strict digraph {",
	}
	fileContent = append(fileContent, determineGlobalOptions(config, graph)...)

	clusters := computeGraphClusters(config, graph)
	for _, cluster := range clusters.clusterList {
		fileContent = append(fileContent, printClusterToDot(config, cluster))
	}

	for _, node := range graph.Nodes().List() {
		fileContent = append(fileContent, printEdgesToDot(config, node, clusters)...)
	}

	fileContent = append(fileContent, "}")

	if _, err = out.WriteString(strings.Join(fileContent, "\n") + "\n"); err != nil {
		config.Logger.WithError(err).Error("Failed to write temporary DOT file.")
		return fmt.Errorf("could not write to %q", out.Name())
	}
	return nil
}

func determineGlobalOptions(config *PrintConfig, graph *depgraph.DepGraph) []string {
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
			rankSep := math.Log10(float64(graph.Nodes().Len())) - 1
			if rankSep < 0.3 {
				rankSep = 0.3
			}
			globalOptions = append(globalOptions, fmt.Sprintf("  ranksep=%.2f", rankSep))
		}
	}
	return globalOptions
}

func printClusterToDot(config *PrintConfig, cluster *graphCluster) string {
	if len(cluster.members) == 0 {
		config.Logger.Warnf("Found an empty node cluster '%s' associated with '%s'.", cluster.name(), cluster.hash)
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

func printNodeToDot(config *PrintConfig, node *depgraph.Node) string {
	var nodeOptions []string
	if config.Style != nil && config.Style.ScaleNodes {
		scaling := math.Log2(float64(len(node.Predecessors())+len(node.Successors()))) / 5
		if scaling < 0.1 {
			scaling = 0.1
		}
		nodeOptions = append(nodeOptions, fmt.Sprintf("width=%.2f,height=%.2f", 5*scaling, scaling))
	}
	if config.Annotate && len(node.SelectedVersion()) != 0 {
		var replacement string
		if node.Module.Replace != nil {
			replacement = node.Module.Replace.Path + "<br />"
		}
		nodeOptions = append(nodeOptions, fmt.Sprintf(
			"label=<%s<br /><font point-size=\"10\">%s%s</font>>",
			node.Name(),
			replacement,
			node.SelectedVersion(),
		))
	}
	dot := "  \"" + node.Name() + "\""
	if len(nodeOptions) > 0 {
		dot += " [" + strings.Join(nodeOptions, ",") + "]"
	}
	return dot
}

func printEdgesToDot(config *PrintConfig, node *depgraph.Node, clusters *graphClusters) []string {
	clustersReached := map[int]struct{}{}

	var dots []string
	for _, dep := range node.Successors() {
		cluster, ok := clusters.clusterMap[dep.End()]
		if !ok {
			config.Logger.Errorf("No cluster reference found for dependency '%s'.", dep.End())
		}

		if _, ok = clustersReached[cluster.id]; ok {
			continue
		}
		clustersReached[cluster.id] = struct{}{}

		target := dep.End()
		var edgeOptions []string
		if minLength := clusters.getClusterDepthMap(dep.End())[node.Name()]; minLength > 1 {
			edgeOptions = append(edgeOptions, fmt.Sprintf("minlen=%d", minLength))
		}
		if len(cluster.members) > 1 {
			edgeOptions = append(edgeOptions, "lhead=\""+cluster.name()+"\"")
			target = cluster.getRepresentative()
		} else if config.Annotate { // We don't annotate an edge with version if it's leading to a cluster.
			edgeOptions = append(edgeOptions, fmt.Sprintf("label=<<font point-size=\"10\">%s</font>>", dep.RequiredVersion()))
		}

		dot := "  \"" + node.Name() + "\" -> \"" + target + "\""
		if len(edgeOptions) > 0 {
			dot += " [" + strings.Join(edgeOptions, ",") + "]"
		}
		dots = append(dots, dot)
	}
	return dots
}
