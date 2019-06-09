#!/usr/bin/env bash
set -u -e -x -o pipefail

# Run all the tests with the race detector and generate coverage.
go test -v -race -coverprofile c.out ./...
