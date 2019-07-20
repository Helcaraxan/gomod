# `gomod` release notes

## Next release

**High-level overview**

**Breaking changes**

- The `lib/depgraph.GetDepGraph()` method no longer takes a boolean to indicate what output should be forwarded from the
  invocations of underlying tools. Instead this is inferred from the level configured on the `logrus.Logger` instance
  argument that it takes.
- Output behaviour for the invocation of underlying tools has slightly changed:
  - By default only their `stderr` will be forwarded to the terminal output.
  - If the `-q | --quiet` flag is passed neither their `stderr`, not their `stdout` will be forwarded.
  - If the `-v | --verbose` flag is passed both `stderr` and `stdout` will be forwarded.

  In any case the full output of these invocations can be found in the debug logs.

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
