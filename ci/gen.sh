#!/usr/bin/env bash
# vim: set tabstop=4 shiftwidth=4 expandtab
set -u -e -o pipefail

PROJECT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."
cd "${PROJECT_ROOT}"

DOT_VERSION="${DOT_VERSION:-"2.40.1"}"

## Ensure 'dot' is available.
if [[ -z "$(command -v dot)" ]] || ! grep "${DOT_VERSION}" <<<"$(dot -V 2>&1)"; then
	echo "Please install the 'dot' tool at version ${DOT_VERSION}."
	exit 1
else
	echo "Found installed dot@${DOT_VERSION}."
fi

function generateDotAndJPG() {
	local basename="$1"
	shift 1 # Drop the basename.

	go run . graph \
		--force \
		--output "${PROJECT_ROOT}/images/${basename}.jpg" \
		"$@"

	shift 1 # Drop the 'visual' / 'style' flag.
	go run . graph \
		--force \
		--output "${PROJECT_ROOT}/images/${basename}.dot" \
		"$@"
}

generateDotAndJPG "dependency-chains" "--visual" "--dependencies=github.com/stretchr/testify,golang.org/x/sys" "--annotate"
generateDotAndJPG "full" "--style=scale_nodes=true,cluster=full"
generateDotAndJPG "shared-dependencies" "--visual" "--shared"
