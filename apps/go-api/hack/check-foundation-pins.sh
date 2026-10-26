#!/bin/sh

set -eu

check_module() {
  module=$1
  expected=$2
  actual=$(go list -m -f '{{.Version}}' "$module")
  if [ "$actual" != "$expected" ]; then
    echo "$module version is $actual; expected $expected" >&2
    exit 1
  fi
}

module_go=$(awk '$1 == "go" { print $2; exit }' go.mod)
if [ -z "$module_go" ]; then
  echo "Go version is missing from go.mod" >&2
  exit 1
fi

if [ "$(go env GOVERSION)" != "go$module_go" ]; then
  echo "Go version is $(go env GOVERSION); expected go$module_go from go.mod" >&2
  exit 1
fi

go_api_image="golang:${module_go}-trixie"
if ! grep -Fqx "FROM $go_api_image AS build" Dockerfile; then
  echo "Go API build image is not pinned to $go_api_image from go.mod" >&2
  exit 1
fi

check_module github.com/CaliLuke/loom v1.10.0-alpha.16
check_module github.com/CaliLuke/selvedge v0.7.0
check_module github.com/golang-migrate/migrate/v4 v4.20.1
check_module github.com/google/go-cmp v0.7.0
check_module github.com/jackc/pgx/v5 v5.11.0
check_module github.com/riverqueue/river v0.49.0
check_module github.com/riverqueue/river/cmd/river v0.49.0
check_module github.com/sqlc-dev/sqlc v1.31.1
check_module github.com/testcontainers/testcontainers-go v0.44.0
check_module github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0
check_module golang.org/x/net v0.61.0

echo "M0 and M1 foundation versions are pinned"
