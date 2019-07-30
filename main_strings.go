package main

const (
	gomodShort = "A tool to visualise and analyse a Go module's dependency graph."

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
	graphLong  = `Generate a visualisation of the dependency network used by the code in your Go module.

The generated graph can be either
- text based in GraphViz's 'dot' format (https://graphviz.gitlab.io/_pages/doc/info/lang.html), or
- an image using a configurable format (GIF, JPG, PDF, PNG, PS)

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
`

	analyseShort = "Analyse the graph of dependencies for this Go module and output interesting statistics."

	revealShort = "Reveal 'hidden' replace'd modules in your direct and direct independencies."
)
