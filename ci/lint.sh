#!/usr/bin/env bash
set -u -e -x -o pipefail

if [[ -z "${GOLANGCI_VERSION-}" ]]; then
  echo "Please specify the 'golangci-lint' version that should be used via the 'GOLANGCI_VERSION' environment variable."
  exit 1
fi

# Retrieve the golangci-lint linter binary.
curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | BINARY="golang-ci" bash -s -- -b ${GOPATH}/bin ${GOLANGCI_VERSION}

# Run the linter.
golangci-lint run ./...

# Check that dependencies are correctly being maintained.
go mod tidy
git diff --exit-code --quiet || (echo "Please run 'go mod tidy' to clean up the 'go.mod' and 'go.sum' files."; false)
