#!/usr/bin/env bash
# vim: set tabstop=2 shiftwidth=2 expandtab
set -e -u -o pipefail

PROJECT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."
cd "${PROJECT_ROOT}"

# Run all the Go tests with the race detector and generate coverage.
printf "\nRunning Go test...\n"
go test -v -race -coverprofile c.out -coverpkg=all ./...
