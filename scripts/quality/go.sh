#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MODULE="$ROOT/apps/go-api"
REPORTS="$ROOT/.quality/reports"
BASELINE="$ROOT/configs/quality/go-lint-baseline.json"
source "$ROOT/scripts/dev-env.sh"
export PATH="$ROOT/.quality/bin:$PATH"
mkdir -p "$REPORTS"

usage() {
  cat <<'USAGE'
Usage: scripts/quality/go.sh [--fast|--full|--tests|--fix|--write-lint-baseline|--write-static-baselines|--write-coverage-baseline|--check-tools]
  --fast                  Format, build, vet, and lint with the existing-issue ratchet.
  --full                  Run all fast gates, mod tidy drift, deadcode, duplication,
                          integration-tagged tests with coverage, and complexity report.
  --tests                 Run only integration-tagged tests and write fresh coverage output.
  --fix                   Apply goimports formatting, then rerun formatting and lint checks.
  --write-lint-baseline   Explicitly replace the reviewed golangci diagnostic baseline.
  --write-static-baselines Rerun deadcode and dupl and explicitly refresh their baselines.
  --write-coverage-baseline Rerun integration tests and explicitly replace coverage floors.
  --check-tools           Verify that all pinned quality tools are installed.
USAGE
}

require_tools() {
  bash "$ROOT/scripts/quality/go-tools.sh" --check
}

go_files() {
  cd "$MODULE"
  find . -type f -name '*.go' ! -path '*/gen/*' ! -path './vendor/*' -print | sort | while IFS= read -r file; do
    if awk 'NR <= 10 && /^\/\/ Code generated .* DO NOT EDIT\.$/ { found = 1 } END { exit !found }' "$file"; then
      continue
    fi
    printf '%s\n' "$file"
  done
}

format_check() {
  local files gofmt_files import_files
  files="$(go_files)"
  [[ -z "$files" ]] && return 0
  gofmt_files="$(printf '%s\n' "$files" | sed "s#^#$MODULE/#" | xargs gofmt -l)"
  import_files="$(printf '%s\n' "$files" | sed "s#^#$MODULE/#" | xargs goimports -l)"
  if [[ -n "$gofmt_files" || -n "$import_files" ]]; then
    [[ -z "$gofmt_files" ]] || { printf 'gofmt required:\n%s\n' "$gofmt_files"; }
    [[ -z "$import_files" ]] || { printf 'goimports required:\n%s\n' "$import_files"; }
    return 1
  fi
}

format_fix() {
  local files
  files="$(go_files)"
  [[ -z "$files" ]] && return 0
  printf '%s\n' "$files" | sed "s#^#$MODULE/#" | xargs goimports -w
}

lint_gate() {
  local json="$REPORTS/go-lint.json"
  rm -f "$json"
  (
    cd "$MODULE"
    golangci-lint run --allow-serial-runners --config .golangci.yml --issues-exit-code=0 \
      --output.json.path="$json" --output.text.path=stderr --output.text.colors=false
  )
  node "$ROOT/scripts/quality/go-lint-ratchet.mjs" check "$json" "$BASELINE" "$MODULE"
}

write_lint_baseline() {
  local json="$REPORTS/go-lint.json"
  rm -f "$json"
  (
    cd "$MODULE"
    golangci-lint run --config .golangci.yml --issues-exit-code=0 \
      --output.json.path="$json" --output.text.path=stderr --output.text.colors=false
  )
  node "$ROOT/scripts/quality/go-lint-ratchet.mjs" write "$json" "$BASELINE" "$MODULE"
}

build_gate() {
  (cd "$MODULE" && go build -tags=integration ./...)
}

vet_gate() {
  (cd "$MODULE" && go vet -tags=integration ./...)
}

tidy_gate() {
  (cd "$MODULE" && go mod tidy -diff)
}

test_gate() {
  local coverage="$REPORTS/go-integration.cover" test_json="$REPORTS/go-test.jsonl" test_status=0 result_mode="${1:-check}"
  node --test "$ROOT/scripts/quality/go-test-results.test.mjs" "$ROOT/scripts/quality/go-ratchets.test.mjs"
  rm -f "$coverage" "$test_json"
  (
    cd "$MODULE"
    go test -json -tags=integration -count=1 -p 2 -parallel 2 -timeout=20m \
      -coverprofile="$coverage" ./...
  ) >"$test_json" || test_status=$?
  node "$ROOT/scripts/quality/go-test-results.mjs" "$test_json" "$ROOT/.quality/go-test-skips.json"
  [[ "$test_status" -eq 0 ]] || return "$test_status"
  [[ -s "$coverage" ]] || { printf 'test run did not produce coverage: %s\n' "$coverage" >&2; return 1; }
  node "$ROOT/scripts/quality/go-coverage-ratchet.mjs" "$result_mode" "$coverage" \
    "$ROOT/configs/quality/go-coverage-baseline.json" "$MODULE"
}

deadcode_gate() {
  local report="$REPORTS/go-deadcode.json"
  local packages filter
  packages="$(cd "$MODULE" && go list -tags=integration -f '{{.ImportPath}}' ./... | rg -v '/(design|gen)(/|$)')"
  filter="$(printf '%s\n' "$packages" | sed -E 's/[][(){}.+*?^$|\\]/\\&/g' | paste -sd '|' -)"
  (
    cd "$MODULE"
    deadcode -test -tags=integration -json -filter="^($filter)$" $packages >"$report"
  )
  node "$ROOT/scripts/quality/go-deadcode-ratchet.mjs" check "$report" \
    "$ROOT/configs/quality/go-deadcode-baseline.json"
}

dupl_gate() {
  local report="$REPORTS/go-dupl.txt"
  (
    cd "$MODULE"
    dupl -plumbing -threshold 100 -files < <(go_files) >"$report"
  )
  node "$ROOT/scripts/quality/go-dupl-ratchet.mjs" check "$report" \
    "$ROOT/configs/quality/go-dupl-baseline.json" "$MODULE"
}

write_static_baselines() {
  local deadcode_report="$REPORTS/go-deadcode.json" dupl_report="$REPORTS/go-dupl.txt"
  local packages filter
  packages="$(cd "$MODULE" && go list -tags=integration -f '{{.ImportPath}}' ./... | rg -v '/(design|gen)(/|$)')"
  filter="$(printf '%s\n' "$packages" | sed -E 's/[][(){}.+*?^$|\\]/\\&/g' | paste -sd '|' -)"
  (
    cd "$MODULE"
    deadcode -test -tags=integration -json -filter="^($filter)$" $packages >"$deadcode_report"
  )
  node "$ROOT/scripts/quality/go-deadcode-ratchet.mjs" write "$deadcode_report" \
    "$ROOT/configs/quality/go-deadcode-baseline.json"
  (
    cd "$MODULE"
    dupl -plumbing -threshold 100 -files < <(go_files) >"$dupl_report"
  )
  node "$ROOT/scripts/quality/go-dupl-ratchet.mjs" write "$dupl_report" \
    "$ROOT/configs/quality/go-dupl-baseline.json" "$MODULE"
}

complexity_report() {
  local report="$REPORTS/go-complexity.txt"
  (
    cd "$MODULE"
    files="$(go_files | tr '\n' ' ')"
    if [[ -n "$files" ]]; then
      gocyclo -over 15 $files >"$report" || [[ $? -eq 1 ]]
      gocyclo -avg $files >>"$report"
    fi
  )
  printf 'Complexity report: %s (functions over 15 are reported; informational only).\n' "$report"
}

run_fast() {
  require_tools
  format_check
  build_gate
  vet_gate
  lint_gate
}

mode="${1:---fast}"
case "$mode" in
  --fast)
    run_fast
    ;;
  --full)
    run_fast
    tidy_gate
    deadcode_gate
    dupl_gate
    test_gate
    complexity_report
    ;;
  --tests)
    require_tools
    test_gate
    ;;
  --fix)
    require_tools
    format_fix
    format_check
    lint_gate
    ;;
  --write-lint-baseline)
    require_tools
    write_lint_baseline
    ;;
  --write-static-baselines)
    require_tools
    write_static_baselines
    ;;
  --write-coverage-baseline)
    require_tools
    test_gate write
    ;;
  --check-tools)
    require_tools
    ;;
  --help|-h)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
