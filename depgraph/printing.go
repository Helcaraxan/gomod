package depgraph

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// PrintConfig allows for the specification of parameters that should be passed
// to the Print function of a DepGraph.
type PrintConfig struct {
	// Logger that should be used to show progress while printing the DepGraph.
	Logger *logrus.Logger
	// Path at which the printed version of the DepGraph should be stored. If
	// set to a nil-string a temporary file will be created.
	OutputPath string
	// Force overwriting of pre-existing files at the specified OutputPath.
	Force bool
	// Visual representation of the DepGraph. If true print out a PDF using
	// GraphViz, if false print out the graph in DOT format.
	Visual bool
}

// Print takes in a PrintConfig struct and dumps the content of this DepGraph
// instance according to parameters.
func (g *DepGraph) Print(config *PrintConfig) error {
	var printer func(*logrus.Logger, string, bool) error
	if config.Visual {
		printer = g.PrintToPDF
	} else {
		printer = g.PrintToDOT
	}
	return printer(config.Logger, config.OutputPath, config.Force)
}

// Visualize creates a PDF file at the specified target path that represents the dependency graph.
func (g *DepGraph) PrintToPDF(logger *logrus.Logger, outputPath string, force bool) error {
	tempDir, err := ioutil.TempDir("", "depgraph")
	if err != nil {
		logger.WithError(err).Error("Could not create temporary directory.")
	}
	logger.Debugf("Using temporary output folder %q.", tempDir)

	if len(outputPath) == 0 {
		outputPath = filepath.Join(tempDir, "out.pdf")
	} else {
		defer func() {
			logger.Debugf("Cleaning up temporary output folder %q.", tempDir)
			_ = os.RemoveAll(tempDir)
		}()
	}

	if filepath.Ext(outputPath) != ".pdf" {
		logger.Warnf("Specified output path %q is not a PDF file.", outputPath)
		outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".pdf"
		logger.Warnf("Will be writing to %q instead.", outputPath)
	}

	if err := prepareOutputPath(logger, outputPath, force); err != nil {
		return err
	}

	dotFile := filepath.Join(tempDir, "out.dot")
	if err = g.PrintToDOT(logger, dotFile, false); err != nil {
		return err
	}

	logger.Debugf("Generating PDF file %q.", outputPath)
	_, err = runCommand(logger, "dot", "-Tpdf", "-o", outputPath, dotFile)
	return err
}

func (g *DepGraph) PrintToDOT(logger *logrus.Logger, outputPath string, force bool) error {
	var err error
	out := os.Stdout
	if len(outputPath) > 0 {
		if err = prepareOutputPath(logger, outputPath, force); err != nil {
			return err
		}

		out, err = os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.WithError(err).Errorf("Could not create output file %q.", outputPath)
			return err
		}
		defer func() {
			_ = out.Close()
		}()
		logger.Debugf("Writing DOT graph to %q.", outputPath)
	} else {
		logger.Debug("Writing DOT graph to terminal.")
	}

	var fileContent []string
	fileContent = append(fileContent, "strict digraph {", "  ranksep=3")
	for name, node := range g.nodes {
		for _, dep := range node.successors {
			fileContent = append(fileContent, fmt.Sprintf("  \"%s\" -> \"%s\"", name, dep.end))
		}
	}
	fileContent = append(fileContent, "}")

	if _, err = out.WriteString(strings.Join(fileContent, "\n") + "\n"); err != nil {
		logger.WithError(err).Error("Failed to write temporary DOT file.")
		return fmt.Errorf("could not write to %q", out.Name())
	}
	return nil
}
