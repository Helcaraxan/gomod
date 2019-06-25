#!/usr/bin/env bash
# vim: set tabstop=4 shiftwidth=4 noexpandtab
set -e -u -o pipefail

# Run all the Go tests with the race detector and generate coverage.
printf "\nRunning Go test...\n"
go test -v -race -coverprofile c.out ./...

# Run all the Bash tests.
printf "\nRunning Bash tests...\n"
./internal/completion/scripts/go
