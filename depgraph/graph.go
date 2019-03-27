package depgraph

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// DepGraph represents a Go module's dependency graph.
type DepGraph struct {
	Logger *logrus.Logger
	Module string
	Nodes  map[string]*Node
}

// Dependency represents a dependency in a Go module's dependency graph.
type Dependency struct {
	Begin   string
	End     string
	Version ModuleVersion
}

// Node represents a module in a Go module's dependency graph.
type Node struct {
	Predecessors    []*Dependency
	Successors      []*Dependency
	SelectedVersion ModuleVersion
}

// DeepCopy returns a separate copy of the current dependency graph that can be
// safely modified without affecting the original graph. The logger argument can
// be nil in which case nothing will be logged.
func (g *DepGraph) DeepCopy() *DepGraph {
	newGraph := &DepGraph{
		Logger: g.Logger,
		Module: g.Module,
		Nodes:  map[string]*Node{},
	}
	for name, node := range g.Nodes {
		newGraph.Nodes[name] = &Node{SelectedVersion: node.SelectedVersion}
	}

	for name, node := range g.Nodes {
		for _, successor := range node.Successors {
			newDependency := &Dependency{
				Begin:   successor.Begin,
				End:     successor.End,
				Version: successor.Version,
			}
			newGraph.Nodes[name].Successors = append(newGraph.Nodes[name].Successors, newDependency)
			newGraph.Nodes[successor.End].Predecessors = append(newGraph.Nodes[successor.End].Predecessors, newDependency)
		}
	}
	g.Logger.Debug("Created a deep copy of graph.")
	return newGraph
}

type PrintConfig struct {
	Logger     *logrus.Logger
	OutputPath string
	Force      bool
	Visual     bool
}

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
	for name, node := range g.Nodes {
		for _, dep := range node.Successors {
			fileContent = append(fileContent, fmt.Sprintf("  \"%s\" -> \"%s\"", name, dep.End))
		}
	}
	fileContent = append(fileContent, "}")

	if _, err = out.WriteString(strings.Join(fileContent, "\n") + "\n"); err != nil {
		logger.WithError(err).Error("Failed to write temporary DOT file.")
		return fmt.Errorf("could not write to %q", out.Name())
	}
	return nil
}
