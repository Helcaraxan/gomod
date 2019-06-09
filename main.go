package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/gomod/analysis"
	"github.com/Helcaraxan/gomod/depgraph"
)

type commonArgs struct {
	logger *logrus.Logger
	quiet  bool
}

func main() {
	commonArgs := &commonArgs{
		logger: logrus.New(),
	}

	var verbose bool
	rootCmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "A tool to visualize and analyze a Go module's dependency graph.",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := checkGoModulePresence(commonArgs.logger); err != nil {
				return err
			}
			if verbose {
				commonArgs.logger.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&commonArgs.quiet, "quiet", "q", false, "Silence output from go tool invocations")

	rootCmd.AddCommand(
		initGraphCmd(commonArgs),
		initAnalyseCmd(commonArgs),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type graphArgs struct {
	*commonArgs

	visual       bool
	annotate     bool
	force        bool
	outputPath   string
	outputFormat string

	shared       bool
	dependencies []string
}

func initGraphCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &graphArgs{
		commonArgs: cArgs,
	}

	graphCmd := &cobra.Command{
		Use:   "graph",
		Short: "Visualise the dependency graph of a Go module.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := checkToolDependencies(cmdArgs.logger); err != nil {
				return err
			}
			return runGraphCmd(cmdArgs)
		},
	}

	// Flags controlling output.
	graphCmd.PersistentFlags().BoolVarP(&cmdArgs.visual, "visual", "V", false, "Format the output as a PDF image")
	graphCmd.PersistentFlags().BoolVarP(&cmdArgs.annotate, "annotate", "a", false, "Annotate the resulting graph's nodes and edges with version information")
	graphCmd.PersistentFlags().BoolVarP(&cmdArgs.force, "force", "f", false, "Overwrite any existing files")
	graphCmd.PersistentFlags().StringVarP(&cmdArgs.outputPath, "output", "o", "", "If set dump the output to this location")
	graphCmd.PersistentFlags().StringVarP(&cmdArgs.outputFormat, "format", "F", "", "Output format for any image file (pdf, png, gif, ...)")

	// Flags controlling graph filtering.
	graphCmd.Flags().BoolVarP(&cmdArgs.shared, "shared", "s", false, "Filter out unshared dependencies (i.e. only required by one Go module)")
	graphCmd.Flags().StringSliceVarP(&cmdArgs.dependencies, "dependencies", "d", nil, "Dependency for which to show the dependency graph")

	return graphCmd
}

func runGraphCmd(args *graphArgs) error {
	if args.shared && len(args.dependencies) > 0 {
		return errors.New("'shared' and 'dependencies' filters cannot be used simultaneously")
	}

	graph, err := depgraph.GetDepGraph(args.logger, args.quiet)
	if err != nil {
		return err
	}

	if args.shared {
		graph = graph.PruneUnsharedDeps()
	} else {
		var versionFilter []*depgraph.DependencyFilter
		for _, dependency := range args.dependencies {
			filter := strings.Split(dependency+"@", "@")
			versionFilter = append(versionFilter, &depgraph.DependencyFilter{
				Dependency: filter[0],
				Version:    filter[1],
			})
		}
		graph = graph.SubGraph(versionFilter)
	}
	return printResult(graph, args)
}

type analyseArgs struct {
	*commonArgs
}

func initAnalyseCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &analyseArgs{
		commonArgs: cArgs,
	}

	analyseCmd := &cobra.Command{
		Use:   "analyse",
		Short: "Analyse the graph of dependencies for this Go module and output interesting statistics.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAnalyseCmd(cmdArgs)
		},
	}
	return analyseCmd
}

func runAnalyseCmd(args *analyseArgs) error {
	graph, err := depgraph.GetDepGraph(args.logger, args.quiet)
	if err != nil {
		return err
	}
	analysisResult := analysis.Analyse(graph)
	return analysisResult.Print(os.Stdout)
}

func checkToolDependencies(logger *logrus.Logger) error {
	tools := []string{
		"dot",
		"go",
	}

	success := true
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			success = false
			logger.Errorf("The %q tool dependency does not seem to be available. Please install it first.", tool)
		}
	}
	if !success {
		return errors.New("missing tool dependencies")
	}
	return nil
}

func checkGoModulePresence(logger *logrus.Logger) error {
	path, err := os.Getwd()
	if err != nil {
		logger.WithError(err).Error("Could not determine the current working directory.")
		return err
	}

	for {
		if _, err = os.Stat(filepath.Join(path, "go.mod")); err == nil {
			return nil
		}
		if path != filepath.VolumeName(path)+string(filepath.Separator) {
			break
		}
	}
	logrus.Error("This tool should be run from within a Go module.")
	return errors.New("missing go module")
}

func printResult(graph *depgraph.DepGraph, args *graphArgs) error {
	return graph.Print(&depgraph.PrintConfig{
		Logger:       args.logger,
		OutputPath:   args.outputPath,
		Force:        args.force,
		Visual:       args.visual,
		Annotate:     args.annotate,
		OutputFormat: depgraph.StringToFormat[args.outputFormat],
	})
}
