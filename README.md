# Go Modules clarified

[![Build Status](https://travis-ci.com/Helcaraxan/gomod.svg?branch=master)](https://travis-ci.com/Helcaraxan/gomod)
[![Maintainability](https://api.codeclimate.com/v1/badges/42f5920cf5c46650945b/maintainability)](https://codeclimate.com/github/Helcaraxan/gomod/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/42f5920cf5c46650945b/test_coverage)](https://codeclimate.com/github/Helcaraxan/gomod/test_coverage)

`gomod` is a tool that helps Go project maintainers to understand their project's dependencies and
it can provide useful information to developers "modularising" non-module projects. It helps you by
visualising your dependency graph and, even more, analyse it for your profit. It will help you
answer typical questions such as:

- How can I visualise the network of my dependencies?
- How old are the versions of my dependencies that I depend on?
- Are different dependencies of my project using potentially conflicting forks of the same module?
- What dependency chains lead to `github.com/foo/bar` and what constraints do they put on versions?
- Why is dependency `github.com/foo/bar` used at version 1.12.0 and not at version `1.5.0` as I
  specified it to be?

## Detailed features

### `gomod graph`

Create a graphical representations of your dependency graph with the possibility to filter out
noise, add annotations and focus on the pieces of the graph that are of interest to you. You can for
example:

- Only show dependencies that are required by more than one package.
- Only show the dependency chains that lead to one or more specified packages.
- Annotate dependencies with the versions in which they are used and the versions constraint
  imposed by each edge of the graph.

This functionality requires the `dot` tool which you will need to install separately. You can
produce images in GIF, JPG, PDF, PNG and PS format.

### `gomod reveal`

Show all the places at which your (indirect) module dependencies use `replace` statements which you
might need to account for in your own `go.mod` in order to build your project.

### `gomod analyse`

Produce a short statistical report of what is going on with your dependencies. The report includes
things like (in)direct dependency counts, mean and max dependency ages, dependency age distribution,
and more.

**NB**: This command can also be invoked as `gomod analyze` for those who intuitively use American
spelling.

## Example output

### Shared dependencies

Graph with only shared dependencies for the [Matterbridge](https://github.com/42wim/matterbridge)
project.
![Shared dependencies graph](./images/shared-dependencies.jpg)

### Dependency chains

Specific zoom on the dependency chains leading to the `github.com/stretchr/testify` module with
version annotations.
![Annotated dependency chains for `github.com/stretchr/testify`](./images/dependency-chains.jpg)

### Hidden `replace`'s

```text
 -> gomod reveal
'github.com/Rhymen/go-whatsapp' is replaced:
   maunium.net/go/mautrix-whatsapp -> github.com/tulir/go-whatsapp @ v0.0.2-0.20190528182350-fde573a2a73b

'gopkg.in/russross/blackfriday.v2' is replaced:
 ✓ maunium.net/go/mautrix            -> github.com/russross/blackfriday/v2 @ v2.0.1
 ✓ maunium.net/go/mautrix-appservice -> github.com/russross/blackfriday/v2 @ v2.0.1
 ✓ maunium.net/go/mautrix-whatsapp   -> github.com/russross/blackfriday/v2 @ v2.0.1

[✓] Match with a top-level replace in 'github.com/42wim/matterbridge'
```

### Dependency statistics

Statistical analysis of the [Matterbridge](https://github.com/42wim/matterbridge) dependency graph.

```text
 -> gomod analyse
-- Analysis for 'github.com/42wim/matterbridge' --
Dependency counts:
- Direct dependencies:   69
- Indirect dependencies: 51

Age statistics:
- Mean age of dependencies: 14 month(s) 15 day(s)
- Maximum dependency age:   82 month(s) 19 day(s)
- Age distribution per month:

  16.67 % |        #
          |        #
          |        #
          |        #
          |    #   #
          |#   #   #
          |#   #   #
          |#   #   #
          |#   # _ #
          |#   # # #
          |#   # # #
          |#   # # #
          |#   # # #
          |#   # # # # #   #
          |# _ # # # # #   #
          |# # # # # # #   #
          |# # # # # # # # #     #
          |# # # # # # # # #     #   # #
          |# # # # # # # # #     # # # #     #     # #
          |# # # # # # # # #   # # # # #   # # # # # # #     # #   #           #             #
   0.00 % |___________________________________________________________________________________
           0                                                                                84

Reverse dependency statistics:
- Mean number of reverse dependencies:    1.40
- Maximum number of reverse dependencies: 9
- Reverse dependency count distribution:

  83.33 % |  #
          |  #
          |  #
          |  #
          |  #
          |  #
          |  #
          |  #
          |  #
          |  # _
   0.00 % |___________________
           0                10
```
