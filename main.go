package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/analysis"
	"github.com/Helcaraxan/gomod/internal/depgraph"
	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/parsers"
	"github.com/Helcaraxan/gomod/internal/printer"
	"github.com/Helcaraxan/gomod/internal/query"
	"github.com/Helcaraxan/gomod/internal/reveal"
)

type commonArgs struct {
	log *logger.Builder
}

func main() {
	var verbose []string

	commonArgs := &commonArgs{
		log: logger.NewBuilder(os.Stderr),
	}

	rootCmd := &cobra.Command{
		Use:   "gomod",
		Short: gomodShort,
		Long:  gomodLong,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			for _, domain := range verbose {
				commonArgs.log.SetDomainLevel(domain, zapcore.DebugLevel)
			}

			log := commonArgs.log.Domain(logger.InitDomain)
			if err := checkToolDependencies(log); err != nil {
				return err
			} else if err = checkGoModulePresence(log); err != nil {
				return err
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringSliceVarP(
		&verbose,
		"verbose",
		"v",
		nil,
		"Verbose output. See 'gomod --help' for more information.",
	)
	v := rootCmd.Flag("verbose")
	v.NoOptDefVal = "all"

	rootCmd.AddCommand(
		initAnalyseCmd(commonArgs),
		initGraphCmd(commonArgs),
		initRevealCmd(commonArgs),
		initVersionCmd(commonArgs),
	)

	if err := rootCmd.Execute(); err != nil {
		commonArgs.log.Log().Debug("Exited with an error.", zap.Error(err))
		os.Exit(1)
	}
}

type graphArgs struct {
	*commonArgs

	annotate   bool
	outputPath string
	packages   bool
	style      *printer.StyleOptions

	query string
}

func initGraphCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &graphArgs{
		commonArgs: cArgs,
	}

	var style string
	graphCmd := &cobra.Command{
		Use:   "graph <query>",
		Short: graphShort,
		Long:  graphLong,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("style") {
				styleOptions, err := parsers.ParseStyleConfiguration(cmdArgs.log.Domain(logger.InitDomain), style)
				if err != nil {
					return err
				}
				cmdArgs.style = styleOptions
			}
			if len(args) == 1 {
				cmdArgs.query = args[0]
			}
			return runGraphCmd(cmdArgs)
		},
	}

	graphCmd.Flags().BoolVarP(&cmdArgs.annotate, "annotate", "a", false, "Annotate the graph's nodes and edges with version information")
	graphCmd.Flags().StringVarP(&cmdArgs.outputPath, "output", "o", "", "If set dump the output to this location")
	graphCmd.Flags().BoolVarP(&cmdArgs.packages, "packages", "p", false, "Operate at package-level instead of module-level on the dependency graph.")
	graphCmd.Flags().StringVar(&style, "style", "", "Set style options that add decorations and optimisations to the produced 'dot' output.")

	return graphCmd
}

func runGraphCmd(args *graphArgs) error {
	graph, err := depgraph.GetGraph(args.log, "")
	if err != nil {
		return err
	}

	q, err := query.Parse(args.log, args.query)
	if err != nil {
		return err
	}
	l := depgraph.LevelModules
	if args.packages {
		l = depgraph.LevelPackages
	}
	if err = graph.ApplyQuery(args.log, q, l); err != nil {
		return err
	}
	args.log.Log().Debug("Printing graph.")
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
		Short:   analyseShort,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAnalyseCmd(cmdArgs)
		},
	}
	return analyseCmd
}

func runAnalyseCmd(args *analyseArgs) error {
	graph, err := depgraph.GetGraph(args.log, "")
	if err != nil {
		return err
	}
	analysisResult, err := analysis.Analyse(args.log.Log(), graph)
	if err != nil {
		return err
	}
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
		Short: revealShort,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRevealCmd(cmdArgs)
		},
	}

	revealCmd.Flags().StringSliceVarP(&cmdArgs.sources, "sources", "s", nil, "Filter all places that are replacing dependencies.")
	revealCmd.Flags().StringSliceVarP(&cmdArgs.targets, "targets", "t", nil, "Filter all places that replace the specified modules.")

	return revealCmd
}

func runRevealCmd(args *revealArgs) error {
	graph, err := depgraph.GetGraph(args.log, "")
	if err != nil {
		return err
	}
	replacements, err := reveal.FindReplacements(args.log.Log(), graph)
	if err != nil {
		return err
	}
	return replacements.Print(args.log.Log(), os.Stdout, args.sources, args.targets)
}

type versionArgs struct {
	*commonArgs
}

func initVersionCmd(cArgs *commonArgs) *cobra.Command {
	cmdArgs := &versionArgs{
		commonArgs: cArgs,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: versionShort,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runVersionCmd(cmdArgs)
		},
	}

	return versionCmd
}

func runVersionCmd(args *versionArgs) error {
	fmt.Printf("%s - built on %s from %s\n", version, date, commit)
	return nil
}

func checkToolDependencies(log *logger.Logger) error {
	tools := []string{
		"go",
	}

	success := true
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			success = false
			log.Error("A tool dependency does not seem to be available. Please install it first.", zap.String("tool", tool))
		}
	}
	if !success {
		return errors.New("missing tool dependencies")
	}
	return nil
}

func checkGoModulePresence(log *logger.Logger) error {
	path, err := os.Getwd()
	if err != nil {
		log.Error("Could not determine the current working directory.", zap.Error(err))
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
	log.Error("This tool should be run from within a Go module.")
	return errors.New("missing go module")
}

func printResult(g *depgraph.DepGraph, args *graphArgs) error {
	l := printer.LevelModules
	if args.packages {
		l = printer.LevelPackages
	}
	return printer.Print(g.Graph, &printer.PrintConfig{
		Log:         args.log.Domain(logger.PrinterDomain),
		Granularity: l,
		OutputPath:  args.outputPath,
		Style:       args.style,
		Annotate:    args.annotate,
	})
}
