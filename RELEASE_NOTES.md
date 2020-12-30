# Release notes

## Next release

**High-level overview**

- Dependency graph filtering has been moved to be a well-defined feature with appropriate support
  and logic in its own package.
- The generated dependency graphs now reflect the true dependency paths for non-module projects
  instead of the artificial dependencies injected by the module system to provide reproducibility.

**New features**

- The new `filters.ArbitraryDependencies` implements a filter that removes an arbitrary set of
  nodes.
- The new `depgraph.Transform` interface type allows for developers using `gomod` as a library to
  make use of external custom filters. This is supported via the new `DepGraph.Transform` method
  that takes the interface type as argument.
- The `depgraph.Graph` type now contains information for packages as well as for modules.
- A new `gomod version` enables the printing of the current version of the `gomod` tool.

**Breaking changes**

- All library functionalities have been, for now, moved into the `internal` tree while further work
  is being done on the actual API and end-functionality of `gomod`.
- `gomod` no longer wraps the invocation of `dot`. To get an image as output simply pipe `gomod`'s
  output into the `dot` binary on the command-line. This means that the `--format` and `--visual`
  flags have been removed and the the `--style` flag no longer implies `--visual`.

## 0.5.0

**High-level overview**

- A significant number of types, methods and functions have been renamed in preparation for a
  future `v1.0.0` release. These renames aim to create a more coherent interface for the
  functionalities exposed by the `depgraph` package.

**New features**

- The `depgraph.DepGraph` type now exposes a `RemoveDependency` method allowing to remove a given
  module including any edges starting or ending at this module.
- The new `lib/modules` package exposes methods to retrieve various levels of module information.
- The `depgraph.DepAnalysis` type now also contains information about the update backlog time of
  a module's dependencies. This reflects the timespan between the timestamp of the used version of a
  dependency and the timestamp of the newest available update.

**Breaking changes**

- Package split: the `depgraph.Module` and `depgraph.ModuleError` types have been extracted to a
  separate `lib/modules` package in preparation for future work that will expand the configurability
  of information loading to support new features.
- Type renames:
  - `depgraph.Node` has been renamed to `depgraph.Dependency` after the pre-existing type of that
    name has been removed in the `v0.4.0` release.
  - `depgraph.NodeReference` has been renamed to `depgraph.DependencyReference`.
  - `depgraph.NodeMap` has been renamed to `depgraph.DependencyMap` and the associated
    `NewNodeMap()` function has accordingly been renamed to `NewDependencyMap()`.
- The `depgraph.DepGraph` type's methods have changed:
  - `Main()` has been removed in favour of direct access to a field with the same name.
  - `Nodes()` has been removed in favour of direct access to a field named `Dependencies`.
  - `Node()` has been renamed to `GetDependency()`.
  - `AddNode()` has been renamed to `AddDependency` and now only returns a `*Dependency` instead of
    also a `bool`. The returned `value` is `nil` if the module passed as parameter could not be
    added.
- The `depgraph.DependencyFilter` type's `Dependency` field has been renamed to `Module`.
- The `depgraph.NewDepGraph()` function now also takes the path where the contained module lives.
- The `depgraph.GetDepGraph()` function now also takes a relative or absolute path to the directory
  where the targeted Go module lives.

## 0.4.0

**High-level overview**

- The presence of the `.dot` tool is now only required when specifying the `-V | --visual` flag to
  `gomod graph`.
- Support for node clustering in generated `.dot` files.
- More fine-grained control over graph generation via the new `--style` flag of `gomod graph`.

**New features**

- Generated `.dot` graphs are now using box nodes rather than the default ellipse style to reduce
  the size of the generated image files and improve readability.
- Specifying formatting options for image generation via `gomod graph` or the underlying library
  functions is now done via a dedicated configuration type.
- The `printer.PrintToDot` function can now generate improved layouts for dependency graphs via the
  use of node clustering, tightly packing modules that share common reverse dependencies together.
  This can result in significant improvements for larger depdendency graphs (_e.g. the PNG image of
  the full dependency graph for the [kubernetes](https://github.com/kubernetes/kubernetes) project
  has 42% less pixels and has a ~7x smaller binary size_).

**Breaking changes**

- The `depgraph.DepGraph` and it's associated methods have been reworked to facilitate
  reproducibility through determinism, meaning their signatures have changed. Both a `NodeReference`
  and `NodeMap` type have been introduced.
- The `depgraph.GetDepGraph()` method no longer takes a boolean to indicate what output should be
  forwarded from the invocations of underlying tools. Instead this is inferred from the level
  configured on the `logrus.Logger` instance argument that it takes. `logrus.WarnLevel` and below
  are considered the same as `--quiet`, `logrus.DebugLevel` and above are equivalent to `--verbose`.
- Output behaviour for the invocation of underlying tools has slightly changed:
  - By default only their `stderr` will be forwarded to the terminal output.
  - If the `-q | --quiet` flag is passed neither their `stderr`, not their `stdout` will be
    forwarded.
  - If the `-v | --verbose` flag is passed both `stderr` and `stdout` will be forwarded.

  In any case the full output of these invocations can be found in the debug logs.
- The `Visual` field of the `printer.PrinterConfig` type has been replaced by `Style` which is a
  pointer to a nested `printer.StyleOptions` type. The `printer.Print` method will generate an
  image if and only if `Style` has a non-`nil` value.

## 0.3.1

**High-level overview**

- Fixed graph printing which was broken between 0.2.1 and 0.3.0.
- Fixed printing of non-versioned (local) hidden replace statements.

## 0.3.0

**High-level overview**

- Added the gomod reveal command to find and highlight hidden replaces in (indirect) module dependencies.
- Added the analyze alias for gomod analyse.

## 0.2.1

**High-level overview**

- Only require the dot tool when running the gomod graph command.
- New developments in the project now have to go through CI before being merged.

## 0.2.0

**High-level overview**

- Redesigned the flags for gomod graph and their semantics for ease-of-use.
- Added the gomod analyse command to generate statistics about a module's dependencies.
- Numerous internal improvements.

## 0.1.0

**High-level overview**

- Created CLI binary with the graph command.
- Support for filtering the dependency graph on various criteria.
