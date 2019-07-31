# Release notes

## Next release

**High-level overview**

**New features**

**Breaking changes**

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
