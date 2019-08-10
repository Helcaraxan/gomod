#!/usr/bin/env bash
# vim: set tabstop=4 shiftwidth=4 expandtab
set -u -e -o pipefail

if [[ "$(uname -s)" != "Linux" ]]; then
	echo "This script is only intended to be run on Linux as the used CLI tools might not be available or differ in their semantics."
	exit 1
fi

PROJECT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."
cd "${PROJECT_ROOT}"

# Ensure linter versions are specified or set the default values.
GOLANGCI_VERSION="${GOLANGCI_VERSION:-"1.17.1"}"
MARKDOWNLINT_VERSION="${MARKDOWNLINT_VERSION:-"0.5.0"}"
SHELLCHECK_VERSION="${SHELLCHECK_VERSION:-"0.6.0"}"
SHFMT_VERSION="${SHFMT_VERSION:-"2.6.4"}"
YAMLLINT_VERSION="${YAMLLINT_VERSION:-"1.10.0"}"

# Retrieve linters if necessary.
mkdir -p "${PWD}/bin"
PATH="${PATH}:${PWD}/bin"

## golangci-lint
if [[ -z "$(command -v golangci-lint)" ]] || ! grep "${GOLANGCI_VERSION}" <<<"$(golangci-lint --version)"; then
	echo "Installing golangci-lint@${GOLANGCI_VERSION}."
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | BINARY="golang-ci" bash -s -- -b "${PWD}/bin" "v${GOLANGCI_VERSION}"
else
	echo "Found installed golangci-lint@${GOLANGCI_VERSION}."
fi

## shellcheck
if [[ -z "$(command -v shellcheck)" ]] || ! grep "${SHELLCHECK_VERSION}" <<<"$(shellcheck --version)"; then
	echo "Installing shellcheck@${SHELLCHECK_VERSION}."
	curl -sfL "https://storage.googleapis.com/shellcheck/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.xz" |
		tar -xJv --strip-components=1 --directory="${PWD}/bin"
else
	echo "Found installed shellcheck@${SHELLCHECK_VERSION}."
fi

# shfmt
if [[ -z "$(command -v shfmt)" ]] || ! grep "${SHFMT_VERSION}" <<<"$(shfmt -version)"; then
	echo "Installing shfmt@${SHFMT_VERSION}."
	GOBIN="${PWD}/bin" go get -u "mvdan.cc/sh/cmd/shfmt@v${SHFMT_VERSION}"
else
	echo "Found installed shfmt@${SHFMT_VERSION}."
fi

## markdownlint
if [[ -z "$(command -v mdl)" ]] || ! grep "${MARKDOWNLINT_VERSION}" <<<"$(mdl --version)"; then
	echo "Installing mdl@${MARKDOWNLINT_VERSION}."
	gem install mdl -v "${MARKDOWNLINT_VERSION}"
	GEM_INSTALL_DIR="$(gem environment | grep -E -e "- INSTALLATION DIRECTORY" | sed -E 's/.* ([[:print:]]+)$/\1/')/bin"
	PATH="${PATH}:${GEM_INSTALL_DIR}"
else
	echo "Found installed mdl@${MARKDOWNLINT_VERSION}."
fi

## yamllint
if [[ -z "$(command -v yamllint)" ]]; then
	echo "Could not find yamllint@${YAMLLINT_VERSION}. Please install it manually."
	exit 1
elif ! grep "${YAMLLINT_VERSION}" <<<"$(yamllint --version)"; then
	echo "WARNING - yamllint found at non-default version '$(yamllint --version)'. Results might differ."
else
	echo "Found installed yamllint@${YAMLLINT_VERSION}."
fi

# Run linters.
echo "Ensuring that generated Go code is being kept up to date."
go generate ./...
git diff --exit-code --quiet || (
	echo "Please run 'go generate ./...' to update the generated Go code."
	false
)

# Verify that the generated images are up-to-date.
echo "Checking that generated images are up-to-date..."
./ci/gen.sh
git add .
git diff --cached --exit-code --quiet ./images/*.dot || (
	# We 'git diff' only against the text files and not the images as the 'dot'
	# tool sadly does not provide cross-platform deterministic generation even
	# when setting the starting seed in the graph's definition.
	git diff --cached ./images/*.dot
	echo "Please run './ci/gen.sh' to update generated documents and images."
	false
)

echo "Linting YAML files."
yamllint --strict --config-file=./.yamllint.yaml .

echo "Linting Go source code."
golangci-lint run ./...

echo "Ensuring that 'go.mod' and 'go.sum' are being kept up to date."
go mod tidy
git diff --exit-code --quiet || (
	echo "Please run 'go mod tidy' to clean up the 'go.mod' and 'go.sum' files."
	false
)

echo "Performing a static analysis of Bash scripts."
shell_failure=0
shell_vim_directives="# vim: set tabstop=4 shiftwidth=4 expandtab"
while read -r shell_file; do
	echo "Linting ${shell_file}"

	pushd "$(dirname "${shell_file}")"
	shell_file="$(basename "${shell_file}")"
	shellcheck --check-sourced --external-sources --shell=bash --severity=style "${shell_file}" || shell_failure=1
	if ! grep -q "^${shell_vim_directives}$" "${shell_file}"; then
		echo "'${shell_file}' is missing the compulsory VIm directives: ${shell_vim_directives}"
		shell_failure=1
	fi
	popd
done <<<"$(shfmt -f .)"
if ((shell_failure == 1)); then
	echo "Errors were detected while linting shell scripts."
	exit 1
fi

echo "Checking the formatting of Bash scripts."
shfmt -s -d .

echo "Linting Markdown files."
mdl .
