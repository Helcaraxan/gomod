# RELEASE-NUMBER

## High-level overview

## Bug fixes

- Not all test-only dependent packages were adequately marked as such. This has been addressed and
  any packages only imported when building tests are now appropriately recognised as such.

## New features

- If no query is specified for `gomod graph` then it will return the full dependency graph by
  default.
- Nodes in the generated DOT graphs are now coloured based on their (parent) module's name.
  Test-only dependencies are distinguishable by a lighter colour-palette than core dependencies.
  Similary edges reflecting test-only dependencies are now marked with a distinct colour just as
  indirect module dependencies are reflected by dashed lines instead of continous ones.

## Breaking changes

- The query syntax for paths has been modified in favour of using glob-based matching. As a result
  the `foo/...` prefix-matching is no longer recognised. Instead the more flexible `foo/**` can be
  used which also allows for middle-of-path wildcards such as `foo/**/bar/*`.
