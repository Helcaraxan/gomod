# RELEASE-NUMBER

## High-level overview

## Bug fixes

- Not all test-only dependent packages were adequately marked as such. This has been addressed and
  any packages only imported when building tests are now appropriately recognised as such.

## New features

- Nodes in the generated DOT graphs are now coloured based on their (parent) module's name.
  Test-only dependencies are distinguishable by a lighter colour-palette than core dependencies.
  Similary edges reflecting test-only dependencies are now marked with a distinct colour just as
  indirect module dependencies are reflected by dashed lines instead of continous ones.

## Breaking changes
