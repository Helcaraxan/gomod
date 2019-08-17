// Code generated. DO NOT EDIT.

package completion

const GomodCustomFunc = `#!/usr/bin/env bash
# vim: set tabstop=2 shiftwidth=2 expandtab
# shellcheck shell=bash disable=SC2154

function __gomod_graph_format() {
  local formats

  formats=(
    "gif"
    "jpg"
    "pdf"
    "png"
    "ps"
  )
  IFS=$'\n' read -r -d '\0' -a COMPREPLY < <(compgen -W "${formats[*]}" -- "${cur}")
}

function __gomod_graph_dependencies() {
  local cur_prefix cur_suffix out

  cur_prefix="${cur%,*}"
  cur_suffix="${cur##*,}"
  if [[ ${cur_prefix} == "${cur_suffix}" ]]; then
    cur_prefix=""
  else
    cur_prefix="${cur_prefix},"
  fi

  IFS=$'\n' read -r -d '\0' -a out < <(go list -m all 2>/dev/null | awk '{ print $1 }')
  if ((${#out[@]} > 0)); then
    read -r -d '\0' -a COMPREPLY < <(
      export cur_prefix
      compgen -W "${out[*]}" -- "${cur_suffix}" | awk '{ print ENVIRON["cur_prefix"]$1 }'
    )
  fi
}
`
