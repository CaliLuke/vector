#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MODULE="$ROOT/apps/go-api"
source "$ROOT/scripts/dev-env.sh"

readonly BIN_DIR="${GO_QUALITY_BIN_DIR:-$ROOT/.quality/bin}"
readonly GOLANGCI_VERSION="v2.15.0"
readonly XTOOLS_VERSION="v0.52.0"
readonly DUPL_VERSION="v1.1.0"
readonly GOCYCLO_VERSION="v0.6.0"

install_tools() {
  mkdir -p "$BIN_DIR"
  GOBIN="$BIN_DIR" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"$GOLANGCI_VERSION"
  GOBIN="$BIN_DIR" go install golang.org/x/tools/cmd/goimports@"$XTOOLS_VERSION"
  GOBIN="$BIN_DIR" go install golang.org/x/tools/cmd/deadcode@"$XTOOLS_VERSION"
  GOBIN="$BIN_DIR" go install github.com/mibk/dupl@"$DUPL_VERSION"
  GOBIN="$BIN_DIR" go install github.com/fzipp/gocyclo/cmd/gocyclo@"$GOCYCLO_VERSION"
}

check_tool() {
  local name="$1" module="$2" version="$3" path actual
  path="$BIN_DIR/$name"
  if [[ ! -x "$path" ]]; then
    path="$(command -v "$name" || true)"
  fi
  if [[ -z "$path" || ! -x "$path" ]]; then
    printf 'missing Go quality tool %s; run scripts/quality/go-tools.sh --install\n' "$name" >&2
    return 1
  fi
  actual="$(go version -m "$path" | awk -v module="$module" '$1 == "mod" && $2 == module {print $3; exit}')"
  if [[ "$actual" != "$version" ]]; then
    printf '%s version mismatch: found %s at %s, expected %s; run scripts/quality/go-tools.sh --install\n' "$name" "${actual:-unknown}" "$path" "$version" >&2
    return 1
  fi
}

mode="${1:---check}"
case "$mode" in
  --install)
    install_tools
    ;;
  --check)
    ;;
  *)
    printf 'usage: %s [--check|--install]\n' "$0" >&2
    exit 2
    ;;
esac

check_tool golangci-lint github.com/golangci/golangci-lint/v2 "$GOLANGCI_VERSION"
check_tool goimports golang.org/x/tools "$XTOOLS_VERSION"
check_tool deadcode golang.org/x/tools "$XTOOLS_VERSION"
check_tool dupl github.com/mibk/dupl "$DUPL_VERSION"
check_tool gocyclo github.com/fzipp/gocyclo "$GOCYCLO_VERSION"
