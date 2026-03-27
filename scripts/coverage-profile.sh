#!/bin/sh
set -eu

profile=${1:-.coverage/coverage.out}
raw_profile=${2:-.coverage/coverage.raw.out}
repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

profile_path=$profile
case "$profile_path" in
  /*) ;;
  *) profile_path="$repo_root/$profile_path" ;;
esac

raw_profile_path=$raw_profile
case "$raw_profile_path" in
  /*) ;;
  *) raw_profile_path="$repo_root/$raw_profile_path" ;;
esac

mkdir -p "$(dirname "$profile_path")" "$(dirname "$raw_profile_path")"

GOCACHE=${GOCACHE:-$repo_root/.gocache} \
go test ./... -coverpkg=./... -coverprofile="$raw_profile_path"

{
  printf 'mode: set\n'
  tail -n +2 "$raw_profile_path" | grep -v 'internal/perftest/'
} >"$profile_path"

GOCACHE=${GOCACHE:-$repo_root/.gocache} \
go tool cover -func="$profile_path" | tail -n 1
