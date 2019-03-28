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
	logger *logrus.Logger
	module string
	nodes  map[string]*Node
}

// NewGraph returns a new DepGraph instance which will use the specified
// logger for writing log output. If nil a null-logger will be used instead.
func NewGraph(logger *logrus.Logger) *DepGraph {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(ioutil.Discard)
	}
	return &DepGraph{
		logger: logger,
		nodes:  map[string]*Node{},
	}
}

// Module returns the name of the module to which this DepGraph instance applies.
func (g *DepGraph) Module() string {
	return g.module
}

// Nodes returns a slice with copies of all nodes belonging to this DepGraph
// instance. These Node copies can be interacted with without modifying the
// underlying DepGraph.
func (g *DepGraph) Nodes() []Node {
	var idx int
	nodes := make([]Node, len(g.nodes))
	for _, node := range g.nodes {
		nodes[idx] = *node
		idx++
	}
	return nodes
}

// Node represents a module in a Go module's dependency graph.
type Node struct {
	name            string
	predecessors    []*Dependency
	successors      []*Dependency
	selectedVersion ModuleVersion
}

// Name of the module represented by this Node in the DepGraph instance.
func (n *Node) Name() string {
	return n.name
}

// SelectedVersion corresponds to the version of the dependency represented by
// this Node which was selected for use.
func (n *Node) SelectedVersion() string {
	return string(n.selectedVersion)
}

// Predecessors returns a slice with copies of all the incoming Dependencies for
// this  Node. These Dependency copies can be interacted with without modifying
// the underlying DepGraph.
func (n *Node) Predecessors() []Dependency {
	var idx int
	predecessors := make([]Dependency, len(n.predecessors))
	for _, predecessor := range n.predecessors {
		predecessors[idx] = *predecessor
		idx++
	}
	return predecessors
}

// Successors returns a slice with copies of all the outgoing Dependencies for
// this  Node. These Dependency copies can be interacted with without modifying
// the underlying DepGraph.
func (n *Node) Successors() []Dependency {
	var idx int
	successors := make([]Dependency, len(n.successors))
	for _, successor := range n.successors {
		successors[idx] = *successor
		idx++
	}
	return successors
}

// Dependency represents a dependency in a DepGraph instance.
type Dependency struct {
	begin   string
	end     string
	version ModuleVersion
}

// Begin returns the name of the Go module at which this Dependency originates.
func (d *Dependency) Begin() string {
	return d.begin
}

// End returns the name of the Go module which this Dependency requires.
func (d *Dependency) End() string {
	return d.end
}

// RequiredVersion is the minimal required version of the Go module which this
// Dependency requires.
func (d *Dependency) RequiredVersion() string {
	return string(d.version)
}

// DeepCopy returns a separate copy of the current dependency graph that can be
// safely modified without affecting the original graph. The logger argument can
// be nil in which case nothing will be logged.
func (g *DepGraph) DeepCopy() *DepGraph {
	newGraph := &DepGraph{
		logger: g.logger,
		module: g.module,
		nodes:  map[string]*Node{},
	}
	for name, node := range g.nodes {
		newGraph.nodes[name] = &Node{selectedVersion: node.selectedVersion}
	}

	for name, node := range g.nodes {
		for _, successor := range node.successors {
			newDependency := &Dependency{
				begin:   successor.begin,
				end:     successor.end,
				version: successor.version,
			}
			newGraph.nodes[name].successors = append(newGraph.nodes[name].successors, newDependency)
			newGraph.nodes[successor.end].predecessors = append(newGraph.nodes[successor.end].predecessors, newDependency)
		}
	}
	g.logger.Debug("Created a deep copy of graph.")
	return newGraph
}

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
