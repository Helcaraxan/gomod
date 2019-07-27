# Contributing to this project

This project welcomes contributions from the community. In order to help you make the best use of
your time while contributing you will find some guidelines below that should clarify any doubts you
might have and answers most of the usual questions that arise as part of general open-source
development.

## Table of contents

- [How to raise an Issue](#how-to-raise-an-issue)
  - [Opening a new Issue](#opening-a-new-issue)
  - [Do's and Don'ts](#dos-and-donts)
  - [Addressing an Issue](#addressing-an-issue)
- [How to create a Pull Request](#how-to-create-a-pull-request)
  - [Opening a new Pull Request](#opening-a-new-pull-request)
  - [Writing up a good description](#writing-up-a-good-description)
    - [Summary](#summary)
    - [Tests](#tests)
- [Code guidelines](#code-guidelines)
  - [Style and linters](#style-and-linters)
  - [Continuous integration](#continuous-integration)

## How to raise an Issue

### Opening a new Issue

Whether you think you have found a bug, encountered some unexpected behaviour, a problem that this
project could potentially solve or a suggestion for an improvement do not hesitate to open an
[Issue](https://github.com/Helcaraxan/gomod/issues) describing it. Do keep in mind the following
reminders to guarantee yourself the highest chance of getting a quick answer to your question and
any related code changes.

### Do's and Don'ts

Do:

- Follow the checklist provided by the template when creating a new Issue.
- Be polite and constructive in your questions and any follow-up discussions.
- Focus on the _What_ and the _Why_ first, instead of the _How_. This generally allows for more
  open-minded thinking and a wider range of solutions.

Don't:

- Make demands. Open-source projects are generally maintained on people's free time and they will be
  very unlikely to help you if they feel you do not value that aspect of their work.
- Tell someone in a discussion that they are wrong and you are right. Instead provide the arguments
  that explain why your approach has more advantages and / or less disadvantages than theirs.

### Addressing an Issue

An Issue has been discussed and a way forward has been found? Now it's time to actually implement
what has been agreed upon. Whether you are the person who raised the Issue first, a participant in
the discussion or simply someone who wants to contribute to the project your next step will be to
[open a Pull Request](#how-to-create-a-pull-request) with the necessary changes.

If the changes are complex or might require multiple consequetive Pull Requests it is best to update
the corresponding Issue and tell other participants that you will be taking on the work. Bonus
points if you can also provide a rough estimate for when you think you will be able to deliver the
work. This will help others understand what to expect and prevents two developers of working on the
same thing.

## How to create a Pull Request

### Opening a new Pull Request

If you have an idea for a change to the codebase take a pause before starting to write the actual
change you have in mind. It is generally useful to answer a few basic questions first. This will
help you decide whether to open up a PR with the suggested changes or if it might be more
appropriate to [raise an issue](#how-to-raise-an-issue) instead to discuss the changes first.

- What kind of change do you have in mind?
  - A simple bugfix.

    _Go and open that PR 😄

  - A complex bugfix.

    _Unless there is already a related open Issue where an agreement has been found on how to fix
    the bug it is best to first open an Issue (if it doesn't exist yet) or describe the fix you are
    suggesting._

  - An improvement of an existing feature.

    _Does the improvement serve a niche use-case of the feature? Will the change break or modify
    existing use-cases of the feature? Does the improvement require significant changes to the code?
    Is the improved behaviour hard to test?_

    _If you can answer yes to any of the above questions it is best to open an Issue first to
    discuss your idea._

  - A new feature.

    _Open an Issue first to discuss your idea and check that it is compatible with other planned
    features and has the blessing of the project maintainer._

### Writing up a good description

Below you will find some examples on how to properly fill in the description of your Pull Request. A
well-written description is often a pleasure to read and generally invites a project maintainer to
provide quick and high-quality feedback. On the other hand, a poorly written or even absent PR
description is less likely to get the attention of a project maintainer and **might even lead to
your PR being closed without review** until its description is up to standards.

#### Summary

An examples of a good summary is:
> Larger Go modules tend to generate huge dependency graph images that have large holes in them
> without any nodes.
>
> Following the discussions in issue #issue-ref, this PR adds support for the clustering of nodes
> that share the same predecessors in the dependency graph which leads to significantly smaller
> image files containing less holes. The clustering is implemented via a new `depCluster` type
> that exposes a few useful methods as well as some utility functions that compute the clusters for
> a given dependency graph.
>
> Support for printing dependency graphs while using the new `depCluster` type will be added in a
> follow-up PR.

This description briefly describes the nature of the change and why it is beneficial. It also
appropriately references the issue where the change was discussed up front before being implemented
and acknowledges that there will be more changes required which will be part of a future PR.
Alternatively you could also write something similar to:
> Ensuring that the documentation of the project is appropriately maintained as the set of
> available features changes and grows over time is tedious and error-prone if done manually. It is
> much better to automate all the maintenance actions we can. Besides facilitating the regeneration
> of the documentation when changed such automation also allows to test for any forgotten changes
> by running the generation script as part of continuous integration.
>
> This PR creates a script that automatically generates some parts of the documentation and updates
> the continuous integration test script to check that the documentation is being kept up-to-date.

Here the description makes a clear case for the "why" of the suggested change. The change is small
enough that there's not necessarily a need to raise an issue up front and any details can be
discussed as part of the PR's review process.

An examples of a bad summary would be:
> Fix for #issue-ref

This does not provide any context on the nature of the fix that is being implemented, nor why the
suggested change is the _best_ fix for this specific issue. Another bad example would be to simply
have no summary with the placeholder still left behind:
> _Explain what the goal of your PR is, why it is needed and how it achieves the described goal._

#### Tests

Again a good example first:
> This PR is a bugfix and does not change the functionality of the code it touches. It does add a
> few new testcases in the existing unit-tests to cover the edge-cases that exposed the bug in the
> first place.

Or otherwise:
> New tests have been added for each of the methods exposed by the new `OptionParser` type. The
> tests use a standard iteration over a list of testcases which together cover all expected usages
> of the new type.

Do not write something like:
> The feature set added by this PR is too wide for unit-tests. We might want to add integration
> tests at a later stage.

Although this explanation does acknowledge the lack of tests it does not provide an adequate reason
for not implementing the integration test framework first before implementing the proposed change.

## Code guidelines

### Style and linters

This project encourages the use of the general Go styleguide as described in
[Effective Go](https://golang.org/doc/effective_go.html) and on the
[Go wiki](https://github.com/golang/go/wiki/CodeReviewComments).

The more fine-grained codestyle points that are not covered by the general guidelines are enforced
via linting and static analysis. You can find the exact details of what these points are by reading
the corresponding [linter configuration](../.golangci.yaml). In order to get appropriate warnings
while editing the code you can
[configure your editor](https://github.com/golangci/golangci-lint/#editor-integration) to use
`golangci-lint` with this project's specific configuration.

### Continuous integration

The quality of this project's code is maintained by running each Pull Request through a continuous
integration pipeline provided by [Travis CI](https://travis-ci.com/Helcaraxan/gomod). A passing
build is required for each Pull Request before it can be merged.
