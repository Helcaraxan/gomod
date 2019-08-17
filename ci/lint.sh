#!/usr/bin/env bash
# vim: set tabstop=2 shiftwidth=2 expandtab
set -u -e -o pipefail
[[ -n ${DEBUG:-} ]] && set -x

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "This script is only intended to be run on Linux as the used CLI tools might not be available or differ in their semantics."
  exit 1
fi

readonly PROJECT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."
cd "${PROJECT_ROOT}"

# Ensure linter versions are specified or set the default values.
readonly GOLANGCI_VERSION="${GOLANGCI_VERSION:-"1.27.0"}"
readonly MARKDOWNLINT_VERSION="${MARKDOWNLINT_VERSION:-"0.9.0"}"
readonly SHELLCHECK_VERSION="${SHELLCHECK_VERSION:-"0.7.1"}"
readonly SHFMT_VERSION="${SHFMT_VERSION:-"3.1.1"}"
readonly YAMLLINT_VERSION="${YAMLLINT_VERSION:-"1.23.0"}"

# Retrieve linters if necessary.
mkdir -p "${PWD}/bin"
export PATH="${PWD}/bin:${PATH}"

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
  curl -LSs "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.xz" |
    tar --extract --xz --strip-components=1 --directory="${PWD}/bin" "shellcheck-v${SHELLCHECK_VERSION}/shellcheck"
else
  echo "Found installed shellcheck@${SHELLCHECK_VERSION}."
fi

# shfmt
if [[ -z "$(command -v shfmt)" ]] || ! grep "${SHFMT_VERSION}" <<<"$(shfmt -version)"; then
  echo "Installing shfmt@${SHFMT_VERSION}."
  GOBIN="${PWD}/bin" go get -u "mvdan.cc/sh/v3/cmd/shfmt@v${SHFMT_VERSION}"
else
  echo "Found installed shfmt@${SHFMT_VERSION}."
fi

## markdownlint
if [[ -z "$(command -v mdl)" ]] || ! grep "${MARKDOWNLINT_VERSION}" <<<"$(mdl --version)"; then
  echo "Installing mdl@${MARKDOWNLINT_VERSION}."
  mkdir -p "${HOME}/.ruby"
  export GEM_HOME="${HOME}/.ruby"
  gem install mdl "--version=${MARKDOWNLINT_VERSION}" --bindir=./bin
else
  echo "Found installed mdl@${MARKDOWNLINT_VERSION}."
fi

## yamllint
if [[ -z "$(command -v yamllint)" ]] || ! grep "${YAMLLINT_VERSION}" <<<"$(yamllint --version)"; then
  echo "Installing yamllint@${YAMLLINT_VERSION}."
  pip install "yamllint==${YAMLLINT_VERSION}"
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

echo "Linting Bash scripts."
declare -a shell_files
while read -r file; do
  shell_files+=("${file}")
done <<<"$(shfmt -f .)"
shellcheck --external-sources --shell=bash --severity=style "${shell_files[@]}"

shell_failure=0
readonly shell_vim_directives="# vim: set tabstop=2 shiftwidth=2 expandtab"
for shell_file in "${shell_files[@]}"; do
  if ! grep -q "^${shell_vim_directives}$" "${shell_file}"; then
    echo "'${shell_file}' is missing the compulsory VIm directives: ${shell_vim_directives}"
    shell_failure=1
  fi
done
if ((shell_failure == 1)); then
  echo "Errors were detected while linting shell scripts."
  exit 1
fi

echo "Checking the formatting of Bash scripts."
shfmt -i 2 -s -w -d .

echo "Linting Markdown files."
mdl .
