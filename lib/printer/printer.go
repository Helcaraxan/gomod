package printer

import (
	"errors"
	"fmt"
	"io/ioutil"
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
	// Silence the output from underlying tool invocations.
	Quiet bool
	// Visual representation of the DepGraph. If true print out a PDF using
	// GraphViz, if false print out the graph in DOT format.
	Visual bool
	// Annotate edges and nodes with their respective versions.
	Annotate bool
	// Force overwriting of pre-existing files at the specified OutputPath.
	Force bool
	// Path at which the printed version of the DepGraph should be stored. If
	// set to a nil-string a temporary file will be created.
	OutputPath string
	// OutputFormat to use when writing files with the 'dot' tool.
	OutputFormat Format
}

// Print takes in a PrintConfig struct and dumps the content of this DepGraph
// instance according to parameters.
func Print(graph *depgraph.DepGraph, config *PrintConfig) error {
	var printer func(*depgraph.DepGraph, *PrintConfig) error
	if config.Visual {
		printer = PrintToVisual
	} else {
		printer = PrintToDOT
	}
	return printer(graph, config)
}

// PrintToVisual creates an image file at the specified target path that represents the dependency graph.
func PrintToVisual(graph *depgraph.DepGraph, config *PrintConfig) error {
	if config.OutputFormat == FormatUnknown {
		config.OutputFormat = StringToFormat[filepath.Ext(config.OutputPath)[1:]]
	}
	if config.OutputFormat == FormatUnknown {
		return errors.New("unknown format for output file")
	}

	tempDir, err := ioutil.TempDir("", "depgraph")
	if err != nil {
		config.Logger.WithError(err).Error("Could not create temporary directory.")
	}
	config.Logger.Debugf("Using temporary output folder %q.", tempDir)

	outputPath := config.OutputPath
	if len(outputPath) == 0 {
		outputPath = filepath.Join(tempDir, "out."+FormatToString[config.OutputFormat])
		config.Logger.Warnf("Printing to temporary file %q.", outputPath)
	} else {
		defer func() {
			config.Logger.Debugf("Cleaning up temporary output folder %q.", tempDir)
			_ = os.RemoveAll(tempDir)
		}()
	}

	if filepath.Ext(outputPath) != "."+FormatToString[config.OutputFormat] {
		config.Logger.Warnf("Specified output path %q does not have a valid extension.", outputPath)
		outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "." + FormatToString[config.OutputFormat]
		config.Logger.Warnf("Will be writing to %q instead.", outputPath)
	}

	if err := util.PrepareOutputPath(config.Logger, outputPath, config.Force); err != nil {
		return err
	}

	dotPrintConfig := *config
	dotPrintConfig.OutputPath = filepath.Join(tempDir, "out.dot")
	if err = PrintToDOT(graph, &dotPrintConfig); err != nil {
		return err
	}

	config.Logger.Debugf("Generating %q.", outputPath)
	_, err = util.RunCommand(config.Logger, config.Quiet, "dot", "-T"+FormatToString[config.OutputFormat], "-o"+outputPath, dotPrintConfig.OutputPath)
	return err
}

func PrintToDOT(graph *depgraph.DepGraph, config *PrintConfig) error {
	var err error
	out := os.Stdout
	if len(config.OutputPath) > 0 {
		if err = util.PrepareOutputPath(config.Logger, config.OutputPath, config.Force); err != nil {
			return err
		}

		out, err = os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			config.Logger.WithError(err).Errorf("Could not create output file %q.", config.OutputPath)
			return err
		}
		defer func() {
			_ = out.Close()
		}()
		config.Logger.Debugf("Writing DOT graph to %q.", config.OutputPath)
	} else {
		config.Logger.Debug("Writing DOT graph to terminal.")
	}

	var fileContent []string
	fileContent = append(fileContent, "strict digraph {", "  ranksep=3")
	for _, node := range graph.Nodes() {
		nodeOptions := []string{}
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
		if len(nodeOptions) > 0 {
			fileContent = append(fileContent, fmt.Sprintf("  \"%s\" [%s]", node.Name(), strings.Join(nodeOptions, ",")))
		}
		for _, dep := range node.Successors() {
			var edgeOptions []string
			if config.Annotate {
				edgeOptions = append(edgeOptions, fmt.Sprintf("label=<<font point-size=\"10\">%s</font>>", dep.RequiredVersion()))
			}
			fileContent = append(fileContent, fmt.Sprintf(
				"  \"%s\" -> \"%s\"%s",
				dep.Begin(),
				dep.End(),
				fmt.Sprintf(" [%s]", strings.Join(edgeOptions, ",")),
			))
		}
	}
	fileContent = append(fileContent, "}")

	if _, err = out.WriteString(strings.Join(fileContent, "\n") + "\n"); err != nil {
		config.Logger.WithError(err).Error("Failed to write temporary DOT file.")
		return fmt.Errorf("could not write to %q", out.Name())
	}
	return nil
}
