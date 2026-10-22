# M0-001: Go release candidate directive

Status: accepted

## Context

The approved plan requires `go1.27rc2` in both the `go` and `toolchain` directives.

Go removes a `toolchain` directive when it has the same version as the `go` directive. After that edit, Go reports module drift and refuses normal builds.

## Decision

Use `go 1.27rc2` in `go.mod`. Pin the same version in `.go-version`, the service image, and all documented commands.

Do not keep a redundant `toolchain go1.27rc2` line that makes the module fail `go mod tidy`.

This is a documented deviation from the plan text. It preserves the required toolchain version at every executable boundary while keeping the module valid for standard Go commands.

## Follow-up

Replace all four release-candidate pins in one dependency-only change when Go 1.27 becomes stable.
