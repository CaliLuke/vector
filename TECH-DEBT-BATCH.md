# Backend error-handling batch

Commit to one bounded batch covering ERR-02's 190 unchecked-error findings, ERR-01's 52 configured findings, directly related DUP-01 duplication, and the existing uncommitted logout/check-reuse fixes. These are findings to reconcile, not 242 confirmed bugs. Preserve supported API behavior, authorization, tenant isolation, stored payloads, receipt identities, and durable effects. No frontend test audit, dependency upgrade, generated transport edits, broad coverage campaign, or unrelated architecture work.

## Status

- 2026-11-02 — Implementation and focused regressions completed: 32 unchecked-error findings and 11 configured findings removed. All 242 original occurrences are reconciled; 189 are retained and 10 require explicit policy/transaction decisions. Self-review and peer review passed, including the anonymous-logout correction. Publication hooks are the remaining proof; their results belong to the commit/push log rather than another documentation-only gate.

## Milestones

### Milestone 1: Freeze scope and carry forward existing fixes

Goal: Establish the complete finding-to-owner map and preserve work already in the checkout.

Acceptance Criteria

- The executing agent has stated the workflow before code edits. All 190 ERR-02 occurrences and 52 ERR-01 allowances have owner, milestone, and disposition entries in ignored `.quality/reports/tech-debt-batch/dispositions.json`.
- Existing logout and check-reuse changes are preserved; their prior proof is reused when still applicable, and any newly changed behavior has focused proof.

Checklist

- [x] Read this plan, `/Users/luca/.agents/skills/execution-plans/SKILL.md`, `AGENTS.md`, `docs/development.md`, and `docs/architecture.md`; briefly state milestone order, exit criteria, command order, regression-before-fix rule, self/peer review, one batch commit/push, and inherited repository constraints before code edits.
- [x] Save `git status --short` and `git diff` in `.quality/reports/tech-debt-batch/`; preserve `internal/authredirects/service.go`, its untracked `signout_test.go`, `check.sh`, `scripts/quality/gate-stages{,.test}.mjs`, `docs/development.md`, and `docs/unchecked-error-inventory.json`. Go paths in this plan are relative to `apps/go-api` unless prefixed otherwise.
- [x] Capture `git rev-parse HEAD` in `.quality/reports/tech-debt-batch/base-revision.txt` and export that revision as `VECTOR_CHECK_BASE` before focused work. Preserve it through commit and push so the full runner prioritizes this batch's changed packages even after committing.
- [x] Before Go/Docker commands, source `scripts/dev-env.sh`. Before the first database proof, start the dedicated profile with `colima start --profile vector-mcp-tests --activate=false`; export `DOCKER_HOST="unix://$HOME/.colima/vector-mcp-tests/docker.sock"`, `TESTCONTAINERS_HOST_OVERRIDE=127.0.0.1`, and `LIVE_TEST_REDIS_BIND_HOST=127.0.0.1`, then source `scripts/dev-env.sh` again. Retain these settings through Git hooks. Follow `docs/development.md#colima-test-vm`; leave the default VM and other projects untouched.
- [x] Read every `findings[]` entry in `docs/unchecked-error-inventory.json` and every `issues[]` fingerprint/count in `configs/quality/go-lint-baseline.json`. Map each occurrence to its current source location, owner, milestone 2 or 3, and fixed/retained/deferred disposition. Record original fingerprint, rationale, existing proof, and planned focused proof; do not require a new test for a proven analyzer limitation or validated invariant. Reconcile counts exactly; line movement does not constitute closure.
- [x] For affected identifiers, run `rg -n '<exact identifier>' apps/go-api --glob '*.go'` to map direct callers, test doubles, generated DTO consumers and existing assertions before edits. Record concrete package paths and named test selectors in `.quality/reports/tech-debt-batch/proof.md`. Keep generated DTO fields, status codes, nullability, pointer/slice semantics and valid payloads stable. Contract redesign is a deferred finding, not an implementation choice in this batch.
- [x] Read the pending logout and gate-reuse diffs and saved prior proof. Their focused commands are `go -C apps/go-api test ./internal/authredirects -run '^TestSignOutReportsRevocationFailuresAndRevokesDespiteMetadataFailure$' -count=1`, `bun test scripts/quality/gate-stages.test.mjs`, and `bun run oxlint --type-aware --type-check --deny-warnings scripts/quality/gate-stages.mjs scripts/quality/gate-stages.test.mjs`. Record valid existing results; rerun the affected command after further edits.
- [x] Reconcile TEST-01 against already-delivered test reductions and fixture changes using Git history and current `internal/mcpactions` tests. Update only proven stale catalog statements. Do not restart a suite-wide audit or remove remaining authorization/replay combinations based on counts alone.

### Milestone 2: Resolve errors at security, effect and resource owners

Goal: Eliminate confirmed false success and lost required effects without making optional cleanup or telemetry mandatory.

Acceptance Criteria

- Each assigned finding has an evidence-backed disposition. Confirmed defects have a focused failing regression before the fix and a passing result afterward in `proof.md`.
- Required session deletion, durable admission, transaction ownership and resource cleanup retain their documented ordering and failure semantics. Real database behavior is proven with integration tests at the owning boundary.

Checklist

- [x] Review assigned ERR-02 findings in `internal/authredirects/service.go`, `internal/mcpoauth/linking.go`, `internal/backgroundtasks`, `internal/workitems`, `internal/projectstatereads/writes.go`, `cmd/bootstrap/main.go`, and `cmd/system`. Include the remaining resource/effect owners named by the frozen inventory. Classify each operation as required, best effort, already validated, or analyzer limitation using its callers and existing contracts.
- [x] At each confirmed defect's owner, add the smallest regression demonstrating lost error/effect, failed cleanup, or false success; run its recorded exact selector before changing production code. Use existing fakes for pure error propagation. Use the existing PostgreSQL fixture for commit/rollback, durable admission, receipts and concurrency semantics; a mock cannot close those obligations.
- [x] Fix required error propagation and effect ordering within those owners. Preserve replay consumption and sanitized errors in OAuth linking, cookie expiry ordering in logout, stored task/receipt formats, and current cache/recent-visit best-effort policy. Explain retained best-effort errors at the owner; do not add blanket ignore helpers or suppressions.
- [x] Run each changed owner cluster once after its coherent edits, with `go -C apps/go-api test -tags=integration -count=1 -p 1 -parallel 2 -timeout=20m <recorded package paths> -run '<recorded test selectors>'`. Populate literal paths/selectors in `proof.md` before invocation; relevant owners include `./internal/authredirects`, `./internal/mcpoauth`, `./internal/backgroundtasks`, `./internal/workitems`, `./internal/projectstatereads`, and `./cmd/system`. Select existing boundary tests plus new regressions, not whole credential matrices for neutral logic. Re-run only after relevant edits, failures, or unresolved risk.

### Milestone 3: Consolidate projections and resolve remaining configured findings

Goal: Resolve repeated parsing/serialization errors at shared owners and retire owned lint findings without changing supported contracts.

Acceptance Criteria

- Every original inventory entry has a final disposition. Residual findings remain explicitly tracked; this batch does not claim that zero findings is necessary or that every disposition is a fix.
- Shared error projections preserve response fields, status, nullability and classification. Parsing invariants are proven at input boundaries; HTTP write failures do not trigger a second response.
- Lint and duplication allowances decrease only for reviewed, removed findings. Coverage floors and source scope do not weaken.

Checklist

- [x] Group JSON findings in root service adapters such as `activity_reads.go`, `api_cycle_reads.go`, `api_label_reads.go`, and `api_module_reads.go` with `internal/apicontract/{json.go,json_rows.go,serializer.go}` and `internal/jsonvalue/ordered.go`. Map UUID findings, including `entity_favorites.go`, to their existing input validation. Map response writes to `cmd/system` and their actual response owner. Use the frozen inventory to include all remaining files, not only these examples.
- [x] Write focused failure and valid-contract assertions before confirmed behavior fixes. Replace redundant conversion round trips and duplicated projections only where responsibility and contracts match. Reuse existing helpers; preserve domain/authentication/transport boundaries. Keep directly related DUP-01 changes limited to these edited responsibilities.
- [x] Resolve owned ERR-01 cases in `cmd/durable-metadata`, `internal/assets`, `internal/backgroundtasks`, `internal/bootstrap`, `internal/issuequery`, `internal/stickies/content`, `internal/usernotificationpreferences`, `internal/workitems`, `internal/workspaceprojectreads`, and the remaining exact baseline files. Preserve public error text unless existing contracts establish it is internal. Keep proven bodyclose, nilerr and EOF analyzer limitations linked to existing upstream reports in `docs/dependency-refresh.md`; no dependency development.
- [x] Run focused owner and direct-consumer tests immediately after each coherent cluster using milestone 2's command shape and literal commands in `proof.md`. Shared JSON owners include `./internal/apicontract` and `./internal/jsonvalue`; adapter tests belong to `.` and response tests to `./cmd/system`. Pure mapper cases run without a database; use real storage tests only for changed persistence behavior. Do not add full per-action/per-credential products around shared logic.
- [x] Refresh ERR-02 once after code settles: copy `apps/go-api/.golangci.yml` to ignored `.quality/reports/tech-debt-batch/errcheck.yml`, add `linters.settings.errcheck.check-blank: true`, and run from `apps/go-api`: `../../.quality/bin/golangci-lint run --config ../../.quality/reports/tech-debt-batch/errcheck.yml --enable-only errcheck --issues-exit-code=0 --output.json.path=../../.quality/reports/tech-debt-batch/errcheck.json`. Retain original tags, tests, exclusions and timeout. Rebuild `docs/unchecked-error-inventory.json` with current locations, config hash, totals and configured-audit overlap; reconcile removed/new findings against dispositions.
- [x] Capture configured diagnostics once with the same command using `.golangci.yml`, without `--enable-only`, and output `../../.quality/reports/tech-debt-batch/configured-lint.json`. Review every baseline delta, then use `bun scripts/quality/go-lint-ratchet.mjs write .quality/reports/tech-debt-batch/configured-lint.json configs/quality/go-lint-baseline.json apps/go-api` from the repository root. Do not increase allowances.
- [x] Prepare duplication baseline reductions before committing: reproduce `go_files()` from `scripts/quality/go.sh` into `.quality/reports/tech-debt-batch/owned-go-files.txt` (module-relative paths, including tests/untracked owned files; exclude vendor, gen, and files with a generated marker in their first ten lines). From `apps/go-api`, run `../../.quality/bin/dupl -plumbing -threshold 100 -files < ../../.quality/reports/tech-debt-batch/owned-go-files.txt > ../../.quality/reports/tech-debt-batch/dupl.txt`. Review reductions, then run from root `bun scripts/quality/go-dupl-ratchet.mjs write .quality/reports/tech-debt-batch/dupl.txt configs/quality/go-dupl-baseline.json apps/go-api`. No added clone allowances. Do not defer reductions until after the push: stale overlarge allowances fail the ratchet too. Leave unrelated baselines untouched.
- [x] Update the catalog with actual residual counts and concise retained/deferred reasons. Preserve unresolved failures until their cause or obsolescence is established. A green rerun is not closure. Record any new confirmed dependency defect against an existing upstream ticket or report the specific filing blocker; keep external communication within authorized scope.

### Milestone 4: Review and publish the complete batch once

Goal: Publish one coherent batch with the required gate, avoiding repeated broad verification.

Acceptance Criteria

- Self-review and a focused peer review have no unresolved correctness blockers; all 242 original findings are reconciled and retained debt is visible.
- The batch commit is pushed only after required fast and full hooks pass. Proof records name the commit, commands, results, reused stages and remaining debt. No test, deadline, memory limit, coverage floor or owned-source scope was weakened.

Checklist

- [x] Self-review the full diff and dispositions; request one peer review of changed behavior, boundaries, preserved coverage and baseline deltas. Address blockers with focused proof only. Do not ask the reviewer to rerun the suite.
- [x] Keep milestone 1's dedicated VM environment, tool versions and check session stable through publication. Confirm the Git hook shell retains the same exports and sources `scripts/dev-env.sh`.
- [x] Format changed Go files with `.quality/bin/goimports -w` and changed supported text files with `bun run oxfmt`; name the actual changed files explicitly. Run targeted type-aware Oxlint for changed runner files before expensive publication checks. Complete code, tests, inventory and documentation before freezing the checkout.
- [x] Stop the running Vector `pnpm dev` invocation with Ctrl-C through its launching terminal, using its process group for a detached invocation. `pnpm dev:stop` stops containers, not frontend watchers. Confirm that Vector's frontend watchers have exited before publication builds. Record the running stack's ports and environment so restoration retains `http://localhost:8000/` and matching proxy upstreams. Do not clean shared package output while the stack runs.
- [ ] Stage only reviewed batch files, commit once with `git commit` (Husky runs the fast Prek group), then push the current branch with `git push origin HEAD` (Husky runs the full Prek group). Do not run a manual full gate first. The canonical manual full command `prek run --all-files --group full` is an alternative publication workflow, not an additional scheduled pass. Never bypass hooks.
- [ ] Keep the checkout unchanged during hooks. On a required failure, save its exact stage/log, repair the responsible code with narrow proof, amend only this unpublished batch commit, and retry. Reuse eligible stage receipts automatically; do not rerun the complete suite to fish for green. Reuse requires matching source/configuration/tools/environment/artifacts and a valid eight-hour same-session receipt; frontend stages still run when required.
- [ ] Unset `DOCKER_HOST`, `TESTCONTAINERS_HOST_OVERRIDE`, and `LIVE_TEST_REDIS_BIND_HOST`, source `scripts/dev-env.sh`, then restore the Vector stack with `pnpm dev` and its recorded ports/environment. Confirm the local proxy responds and stop the dedicated test VM with `colima stop --profile vector-mcp-tests` after testing. Report only what changed, measured reduction in debt, checks passed and remaining limitations.

## Following batches

1. Remaining TEST-01 overlap and evidence-led runtime reliability, prioritizing NEW-07/09/10/12 that make publication expensive. Ten recorded failures remain open; frontend test-review work stays deferred.
2. MCP-01/02 acceptance: 83 recorded actions, including 54 mutations. Reconcile existing proof before adding cases; mutation debt overlaps acceptance debt.
3. Scoped STRUCT-01/DUP-01 work (16 maintainability exceptions; 466 directed clone reports), followed by meaningful coverage gaps and retained dependency limitations. Counts are not additive task estimates.
