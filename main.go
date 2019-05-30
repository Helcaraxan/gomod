package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/gomod/analysis"
	"github.com/Helcaraxan/gomod/depgraph"
)

type commonArgs struct {
	logger       *logrus.Logger
	outputPath   string
	force        bool
	visual       bool
	annotate     bool
	outputFormat string
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
			if err := checkToolDependencies(commonArgs.logger); err != nil {
				return err
			}
			if verbose {
				commonArgs.logger.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
	}
	rootCmd.PersistentFlags().BoolVarP(&commonArgs.visual, "visual", "V", false, "Format the output as a PDF image")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&commonArgs.force, "force", "f", false, "Overwrite any existing files")
	rootCmd.PersistentFlags().StringVarP(&commonArgs.outputPath, "output", "o", "", "If set dump the output to this location")
	rootCmd.PersistentFlags().BoolVarP(&commonArgs.annotate, "annotate", "a", false, "Annotate the resulting graph's nodes and edges with version information")
	rootCmd.PersistentFlags().StringVarP(&commonArgs.outputFormat, "format", "F", "pdf", "Output format for any image file (pdf, png, gif, ...)")

	rootCmd.AddCommand(
		initFullCmd(commonArgs),
		initSharedCmd(commonArgs),
		initSubCmd(commonArgs),
		initAnalyseCmd(commonArgs),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type fullArgs struct {
	*commonArgs
}

func initFullCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &fullArgs{
		commonArgs: cArgs,
	}

	fullCmd := &cobra.Command{
		Use:   "full",
		Short: "Show the entire dependency graph of this Go module.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFullCmd(cmdArgs)
		},
	}
	return fullCmd
}

func runFullCmd(args *fullArgs) error {
	graph, err := depgraph.GetDepGraph(args.logger)
	if err != nil {
		return err
	}
	return printResult(graph, args.commonArgs)
}

type sharedArgs struct {
	*commonArgs
}

func initSharedCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &sharedArgs{
		commonArgs: cArgs,
	}

	sharedCmd := &cobra.Command{
		Use:   "shared",
		Short: "Show the graph of dependencies for this Go module that are required by multiple modules.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSharedCmd(cmdArgs)
		},
	}
	return sharedCmd
}

func runSharedCmd(args *sharedArgs) error {
	graph, err := depgraph.GetDepGraph(args.logger)
	if err != nil {
		return err
	}
	graph = graph.PruneUnsharedDeps()
	return printResult(graph, args.commonArgs)
}

type subArgs struct {
	*commonArgs
	dependency    string
	targetVersion string
	prune         bool
}

func initSubCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &subArgs{
		commonArgs: cArgs,
	}

	subCmd := &cobra.Command{
		Use:   "sub",
		Short: "Show the graph of dependencies for this Go module that needs to be downgraded to move a depencency to a specific version.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSubCmd(cmdArgs)
		},
	}
	subCmd.Flags().StringVarP(&cmdArgs.dependency, "dependency", "d", "", "Dependency for which to show the dependency graph.")
	subCmd.Flags().StringVarP(&cmdArgs.targetVersion, "target_version", "t", "", "Identify all nodes that restrict the move to this particular version of the dependency.")
	subCmd.Flags().BoolVarP(&cmdArgs.prune, "prune", "p", false, "Remove all nodes that do not restrict the move to the version specified via --target|-t instead of coloring the offending ones.")
	return subCmd
}

func runSubCmd(args *subArgs) error {
	graph, err := depgraph.GetDepGraph(args.logger)
	if err != nil {
		return err
	}
	graph = graph.SubGraph(args.dependency)
	if len(args.targetVersion) > 0 {
		graph = graph.OffendingGraph(args.dependency, args.targetVersion, args.prune)
	}
	return printResult(graph, args.commonArgs)
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
	graph, err := depgraph.GetDepGraph(args.logger)
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

func printResult(graph *depgraph.DepGraph, args *commonArgs) error {
	return graph.Print(&depgraph.PrintConfig{
		Logger:       args.logger,
		OutputPath:   args.outputPath,
		Force:        args.force,
		Visual:       args.visual,
		Annotate:     args.annotate,
		OutputFormat: depgraph.StringToFormat[args.outputFormat],
	})
}
