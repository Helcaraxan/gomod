package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Helcaraxan/gomod/internal/completion"
	"github.com/Helcaraxan/gomod/internal/parsers"
	"github.com/Helcaraxan/gomod/lib/analysis"
	"github.com/Helcaraxan/gomod/lib/depgraph"
	"github.com/Helcaraxan/gomod/lib/printer"
	"github.com/Helcaraxan/gomod/lib/reveal"
)

type commonArgs struct {
	logger *logrus.Logger
}

func main() {
	commonArgs := &commonArgs{
		logger: logrus.New(),
	}

	var verbose, quiet bool
	rootCmd := &cobra.Command{
		Use:   "gomod",
		Short: "A tool to visualise and analyse a Go module's dependency graph.",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := checkGoModulePresence(commonArgs.logger); err != nil {
				return err
			}
			if verbose {
				commonArgs.logger.SetLevel(logrus.DebugLevel)
			} else if quiet {
				commonArgs.logger.SetLevel(logrus.ErrorLevel)
			}
			return nil
		},
		BashCompletionFunction: completion.GomodCustomFunc,
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Silence output from go tool invocations")

	rootCmd.AddCommand(
		initAnalyseCmd(commonArgs),
		initCompletionCommand(commonArgs),
		initGraphCmd(commonArgs),
		initRevealCmd(commonArgs),
	)

	if err := rootCmd.Execute(); err != nil {
		commonArgs.logger.WithError(err).Debug("Exited with an error.")
		os.Exit(1)
	}
}

type completionArgs struct {
	*commonArgs

	rootCmd    *cobra.Command
	shell      completion.ShellType
	outputPath string
}

func initCompletionCommand(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &completionArgs{
		commonArgs: cArgs,
	}

	completionCommand := &cobra.Command{
		Use:   "completion",
		Short: "Commands to generate shell completion for various environments.",
	}

	completionCommand.PersistentFlags().StringVarP(&cmdArgs.outputPath, "output", "o", "", "Output path for the generated completion script.")
	completionCommand.PersistentFlags().Lookup("output").Annotations = map[string][]string{cobra.BashCompFilenameExt: {"", "sh"}}

	completionCommand.AddCommand(
		&cobra.Command{
			Use:   "bash",
			Short: "Generates a bash completion script ready to be sourced.",
			Long: `To load 'gomod' completion rules for a single shell simply run
. <(gomod completion bash)

To load 'gomod' completion for each new bash shell by default add the following to your ~/.bashrc (or equivalent).
# ~/.bashrc or ~/.profile
[[ -n "$(which gomod)" ]] && . <(gomod completion bash)
`,
			RunE: func(cmd *cobra.Command, _ []string) error {
				cmdArgs.shell = completion.BASH
				cmdArgs.rootCmd = cmd.Root()
				return runCompletionCommand(cmdArgs)
			},
		},
		&cobra.Command{
			Use:   "ps",
			Short: "Generates a Powershell completion script ready to be sourced.",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cmdArgs.shell = completion.POWERSHELL
				cmdArgs.rootCmd = cmd.Root()
				return runCompletionCommand(cmdArgs)
			},
		},
		&cobra.Command{
			Use:   "zsh",
			Short: "Generates a zsh completion script ready to be sourced.",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cmdArgs.shell = completion.ZSH
				cmdArgs.rootCmd = cmd.Root()
				return runCompletionCommand(cmdArgs)
			},
		},
	)

	return completionCommand
}

func runCompletionCommand(args *completionArgs) error {
	var err error
	writer := os.Stdout
	if args.outputPath != "" {
		if writer, err = os.OpenFile(args.outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err != nil {
			args.logger.WithError(err).Errorf("Failed to open %q to write completion script.", args.outputPath)
			return err
		}
	}
	return completion.GenerateCompletionScript(args.logger, args.rootCmd, args.shell, writer)
}

type graphArgs struct {
	*commonArgs

	annotate bool
	style    *printer.StyleOptions

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

	var visual bool
	var style string
	graphCmd := &cobra.Command{
		Use:   "graph",
		Short: "Visualise the dependency graph of a Go module.",
		Long: `Generate a visualisation of the dependency network used by the code in your Go module.

The generated graph can be either
- text based in GraphViz's 'dot' format (https://graphviz.gitlab.io/_pages/doc/info/lang.html), or
- an image using a configurable format (GIF, PDF, PNG, PS)

The content of the graph can be controlled via various options.

The '--annotate' flag can be used to add the selected version for each dependency as well as the
version requirements expressed by each dependency edge between modules.

The '--shared' flag prunes any dependencies from the graph that has only one predecessor and no
successor. Such "non-shared" dependencies are imported in the version expressed by the sole module
that requires them. This means that they tend to not intervene in any dependency conflicts or other
version selection issues.

The '--dependencies' flag allows to focus only on a subset of modules and prunes any modules that
are not part of any chain leading to one or more of the specified dependencies.

When generating an image the appearance of the graph can be further fine-tuned with the '--style'
flag. You can specify any formatting options as '<option>=<value>[,<option>=<value>]' out of the
following list:

- 'scale_nodes': one of 'true' or 'false' (default 'false'). This will scale the size of each node
                 of the graph based on the number of inbound and outbound dependencies it has.

- 'cluster':     one of 'off', 'shared', 'full' (default 'off'). This option will generate clusters
                 in the image that force the grouping of shared dependencies together. The result is
                 a tighter graph of reduced size with less "holes" but which might have less visible
                 or understandable edges. When set to 'shared' only dependencies with a single
                 inbound edge are considered and clustered according to the commonality of that
                 ancestor. When set to 'full' any two dependencies that have an identical set of
                 inbound edges are clustered together.

                 WARNING: Using the 'cluster' option can dramatically increase the time required to
                          generate image files, especially for larger dependency graphs.
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Only require 'dot' tool if outputting an image file.
			if visual || cmd.Flags().Changed("style") {
				if err := checkToolDependencies(cmdArgs.logger); err != nil {
					return err
				}
				visualOptions, err := parsers.ParseVisualConfig(cmdArgs.logger, style)
				if err != nil {
					return err
				}
				cmdArgs.style = visualOptions
			}
			return runGraphCmd(cmdArgs)
		},
	}

	// Flags controlling output.
	graphCmd.Flags().BoolVarP(&cmdArgs.annotate, "annotate", "a", false, "Annotate the graph's nodes and edges with version information")
	graphCmd.Flags().BoolVarP(&cmdArgs.force, "force", "f", false, "Overwrite any existing files")
	graphCmd.Flags().StringVarP(&cmdArgs.outputPath, "output", "o", "", "If set dump the output to this location")

	graphCmd.Flags().Lookup("output").Annotations = map[string][]string{cobra.BashCompFilenameExt: {"dot", "gif", "pdf", "png", "ps"}}

	// Flags controlling graph filtering.
	graphCmd.Flags().BoolVarP(&cmdArgs.shared, "shared", "s", false, "Filter out unshared dependencies (i.e. only required by one Go module)")
	graphCmd.Flags().StringSliceVarP(&cmdArgs.dependencies, "dependencies", "d", nil, "Dependency for which to show the dependency graph")

	graphCmd.Flags().Lookup("dependencies").Annotations = map[string][]string{cobra.BashCompCustom: {"__gomod_graph_dependencies"}}

	// Flags controlling image generation.
	graphCmd.Flags().BoolVarP(&visual, "visual", "V", false, "Produce an image of the graph instead of a '.dot' file.")
	graphCmd.Flags().StringVar(&style, "style", "", "Set style options for producing a graph image. Implies '--visual'.")
	graphCmd.Flags().StringVarP(&cmdArgs.outputFormat, "format", "F", "", "Output format for any image file (pdf, png, gif, ...)")

	graphCmd.Flags().Lookup("format").Annotations = map[string][]string{cobra.BashCompCustom: {"__gomod_graph_format"}}

	return graphCmd
}

func runGraphCmd(args *graphArgs) error {
	if args.shared && len(args.dependencies) > 0 {
		return errors.New("'shared' and 'dependencies' filters cannot be used simultaneously")
	}

	graph, err := depgraph.GetDepGraph(args.logger)
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
		Use:     "analyse",
		Aliases: []string{"analyze"}, // nolint
		Short:   "Analyse the graph of dependencies for this Go module and output interesting statistics.",
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

type revealArgs struct {
	*commonArgs
	sources []string
	targets []string
}

func initRevealCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &revealArgs{
		commonArgs: cArgs,
	}

	revealCmd := &cobra.Command{
		Use:   "reveal",
		Short: "Reveal 'hidden' replace'd modules in your direct and direct independencies.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRevealCmd(cmdArgs)
		},
	}

	revealCmd.Flags().StringSliceVarP(&cmdArgs.sources, "sources", "s", nil, "Filter all places that are replacing dependencies.")
	revealCmd.Flags().StringSliceVarP(&cmdArgs.targets, "targets", "t", nil, "Filter all places that replace the specified modules.")

	return revealCmd
}

func runRevealCmd(args *revealArgs) error {
	graph, err := depgraph.GetDepGraph(args.logger)
	if err != nil {
		return err
	}
	replacements, err := reveal.FindReplacements(args.logger, graph)
	if err != nil {
		return err
	}
	return replacements.Print(args.logger, os.Stdout, args.sources, args.targets)
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
	return printer.Print(graph, &printer.PrintConfig{
		Logger:       args.logger,
		OutputPath:   args.outputPath,
		Force:        args.force,
		Style:        args.style,
		Annotate:     args.annotate,
		OutputFormat: printer.StringToFormat[args.outputFormat],
	})
}
