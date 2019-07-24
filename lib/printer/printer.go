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
}

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
		"  start=0", // Needed for placement determinism.
	}

	for _, node := range graph.Nodes() {
		fileContent = append(fileContent, printNodeToDot(config, node))
		fileContent = append(fileContent, printEdgesToDot(config, node)...)
	}
	fileContent = append(fileContent, "}")

	if _, err = out.WriteString(strings.Join(fileContent, "\n") + "\n"); err != nil {
		config.Logger.WithError(err).Error("Failed to write temporary DOT file.")
		return fmt.Errorf("could not write to %q", out.Name())
	}
	return nil
}

func printNodeToDot(config *PrintConfig, node *depgraph.Node) string {
	scaling := math.Log2(float64(len(node.Predecessors())+len(node.Successors()))) / 5
	nodeOptions := []string{
		fmt.Sprintf("width=%.2f,height=%.2f", 5*scaling, scaling),
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

func printEdgesToDot(config *PrintConfig, node *depgraph.Node) []string {
	var dots []string
	for _, dep := range node.Successors() {
		var edgeOptions []string
		if config.Annotate {
			edgeOptions = append(edgeOptions, fmt.Sprintf("label=<<font point-size=\"10\">%s</font>>", dep.RequiredVersion()))
		}

		dot := "  \"" + node.Name() + "\" -> \"" + dep.End() + "\""
		if len(edgeOptions) > 0 {
			dot += " [" + strings.Join(edgeOptions, ",") + "]"
		}
		dots = append(dots, dot)
	}
	return dots
}
