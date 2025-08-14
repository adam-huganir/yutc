#!/usr/bin/env bash

get_current_version() {
  local version
  version="$(grep -oP '\d+.\d+.\d+.*?' ./internal/version.go)"
  echo "${version}"
}

increment_version() {
  local version bump label patch
  bump="${1:-patch}"
  version="$(get_current_version)"
  declare -a components
  while [[ -n "$version" ]]; do
    sleep 0.5
    c="${version%%.*}"
    if [[ "${#components[@]}" -eq 2 ]]; then
      components+=( "${version}" )
      break
    else
      components+=( "$c" )
      version="${version#*.}"
    fi
  done
  patch="${components[2]%%-*}"
  if [[ "${components[2]}" != "$patch" ]]; then
    label="${components[2]#*-}"
  else
    label=""
  fi
  # we aren't doing anything with labels at the moment
  case "${bump}" in
    major)
      components[0]=$((components[0] + 1))
      components[1]=0
      components[2]=0
      ;;
    minor)
      components[1]=$((components[1] + 1))
      components[2]=0
      ;;
    patch)
      components[2]=$((components[2] + 1))
      ;;
    *)
      echo "Invalid bump type: ${bump}"
      return 1
      ;;
  esac
  base="$(printf '%d.%d.%d\n' "${components[@]}")"
  if [[ -n "${label}" ]]; then
    echo "${base}-${label}"
  else
    echo "${base}"
  fi
}
