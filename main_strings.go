package main

var (
	toolVersion = "devel"
	toolDate    = "unknown"
)

const (
	gomodShort = "A tool to visualise and analyse a Go project's dependency graph."

	completionShort = "Commands to generate shell completion for various environments."

	completionBashShort = "Generate a bash completion script ready to be sourced."
	completionBashLong  = `To load 'gomod' completion rules for a single shell simply run
. <(gomod completion bash)

To load 'gomod' completion for each new bash shell by default add the following to your ~/.bashrc (or equivalent).
# ~/.bashrc or ~/.profile
[[ -n "$(which gomod)" ]] && . <(gomod completion bash)
`

	completionPSShort = "Generate a Powershell completion script ready to be sourced."

	completionZSHShort = "Generates a zsh completion script ready to be sourced."

	graphShort = "Visualise the dependency graph of a Go module."
	graphLong  = `Generate a visualisation of the dependency network used by the code in your Go
module.

The command requires a query to be passed to determine what part of the graph
should be printed. The query language itself supports the following syntax:

- Exact or prefix path queries: foo.com/bar or foo.com/bar/...
- Inclusion of test-only dependencies: test(foo.com/bar)
- Dependency queries: 'deps(foo.com/bar)' or 'rdeps(foo.com/bar)
- Depth-limited variants of the above: 'deps(foo.com/bar, 5)'
- Recursive removal of single-parent leaf-nodes: shared(foo.com/bar)'
- Various set operations: X + Y, X - Y, X inter Y, X delta Y.

An example query:

gomod graph -p 'deps(foo.com/bar/...) inter deps(test(test.io/pkg/tool))'

The generated graph's visual aspect (when run through the 'dot' tool) can be
tuned with the '--style' flag. You can specify any formatting options as
'<option>=<value>[,<option>=<value>]' out of the following list:

- 'scale_nodes': one of 'true' or 'false' (default 'false'). This will scale the
                 size of each node of the graph based on the number of inbound
                 and outbound dependencies it has.

- 'cluster':     one of 'off', 'shared', 'full' (default 'off'). This option
                 will generate clusters in the image that force the grouping of
                 shared dependencies together. The result is a tighter graph of
                 reduced size with less "holes" but which might have less
                 visible or understandable edges. When set to 'shared' only
                 dependencies with a single inbound edge are considered and
                 clustered according to the commonality of that ancestor. When
                 set to 'full' any two dependencies that have an identical set
                 of inbound edges are clustered together.
                
                 WARNING: Using the 'cluster' option can dramatically increase
                          the time required to generate image files, especially
                          for larger dependency graphs. But it's for the latter
                          that it can also greatly improve the readability of
                          the final image.
`

	analyseShort = `Analyse the graph of dependencies for this Go module and output interesting
statistics.`

	revealShort = "Reveal 'hidden' replace'd modules in your direct and direct independencies."

	versionShort = "Display the version of the gomod tool."
)
