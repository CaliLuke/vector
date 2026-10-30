# Technical debt catalog

This is the working catalog for the requested retirement of all known technical
debt. It consolidates the quality baselines, MCP acceptance tracker, test review,
and previously local work log. It does not claim an exhaustive audit of every
possible defect.

Snapshot: `17d1afdbf4fd4bcf298f460574f14063712052b6`, 2026-10-30.
Both full quality gates passed at this revision, including 13,381 Go test events
with no failed or skipped cases. Passing gates allow the recorded debt below;
they do not mean this catalog is complete.

## Work selection

The next task must name a catalog ID, an observable problem or missing proof, and
the evidence needed to close it. After a verified change, update its status and
counts. Do not start another cleanup merely because it is adjacent to an edit.
Record newly discovered concerns in the intake section first, with their evidence
and impact. A source observation is not a reproduced defect.

Priority order:

1. Reconcile and complete the original MCP acceptance requirements, prioritizing
   authorization, tenant isolation, data integrity, and durable effects.
2. Correct known runtime failures and unsafe asynchronous behavior, with focused
   regressions. Reduce coverage gaps in the affected behavior at the same time.
3. Retire remaining coverage, maintainability, and duplication debt in bounded
   components with clear ownership and behavior checks.
4. Retire the remaining lower-impact lint and style findings.

Dependency limitations and product follow-ups remain visible separately. They
must not silently become dependency development or new product scope.

## Remaining catalog

Counts are findings or allowances, not equally sized tasks. Do not add overlapping
counts to produce a single "tasks remaining" total.

| ID        | Work                                                  | Current evidence and scope                                                                                                                                                                                                                                                                                                                            | Closure evidence                                                                                                                                                                                                                                                |
| --------- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| MCP-01    | Complete MCP acceptance                               | [Action-by-action tracker](mcp-parity.md) covers 208 actions across 30 tools, including `workspace.list`. Workflow wiring is implemented; compatibility, scope, concurrency, replay, revisions, and durable-effect acceptance remain to be reconciled.                                                                                                | Each action has inspected contract evidence and the applicable generated-server, authorization, tenant, receipt/CAS, effect, and storage proofs. Update stale tracker statements against current tests; do not count a green full gate as missing domain proof. |
| MCP-02    | Complete mutation guarantees                          | [Write contract and recovery requirements](mcp-writes.md), plus mutation rows in MCP-01. This overlaps MCP-01.                                                                                                                                                                                                                                        | Current authorization on replay, request-hash conflicts, relevant native-writer revisions, contention/cancellation/uncertain commits, and durable effects are proven for the documented contracts.                                                              |
| ERR-01    | Resolve configured Go findings                        | [Exact file/message inventory](../configs/quality/go-lint-baseline.json): **162** occurrences: 58 errcheck, 78 staticcheck, 13 errorlint, 7 bodyclose, 6 nilerr. Some analyzer findings have documented upstream limitations.                                                                                                                         | Fix the underlying behavior or remove an obsolete finding with evidence. No increased allowances, blanket suppressions, or source exclusions. Retained upstream limitations stay in EXT-01.                                                                     |
| ERR-02    | Refresh and retire stricter unchecked-error inventory | A previous local `check-blank: true` audit recorded 966 findings, including 887 blank-assignment findings beyond the configured audit. These counts predate subsequent fixes and are **historical, not current totals**.                                                                                                                              | Refresh the audit once into a durable report, reconcile overlap with ERR-01, and handle each retained error according to resource/effect ownership. Do not subtract fixes from a stale count and call it a fresh audit.                                         |
| TS-01     | Resolve TypeScript findings                           | [Full inventory](../.quality/oxlint-full-baseline.json): **1,941** occurrences. [Fast inventory](../.quality/oxlint-fast-baseline.json): 1,062 overlapping occurrences, not additional debt.                                                                                                                                                          | Resolve each recorded finding while preserving contracts. Compiler errors cannot be baselined. Prioritize unsafe data and asynchronous behavior before mechanical style cleanup.                                                                                |
| STRUCT-01 | Retire Go maintainability exceptions                  | [Named functions and scores](../configs/quality/go-maintainability-baseline.json): **17** exceptions, including server assembly and large test functions.                                                                                                                                                                                             | Coherent responsibilities, effect ordering, and narrower interfaces; no score regressions or movement into unmeasured code. Remove allowances only after review.                                                                                                |
| DUP-01    | Consolidate Go duplication                            | [Fingerprint inventory](../configs/quality/go-dupl-baseline.json): **474 directed reports**, represented by 365 fingerprints.                                                                                                                                                                                                                         | Consolidate duplicated responsibilities with behavior checks; reduce the reviewed baseline without adding allowances. Directed reports are not 474 independent refactors.                                                                                       |
| DUP-02    | Consolidate frontend duplication                      | [Fingerprint inventory](../.quality/jscpd-baseline.json): **263 clone pairs**, represented by 262 fingerprints.                                                                                                                                                                                                                                       | Shared behavior/components replace duplicated ownership, with focused consumer tests and no new clone allowances.                                                                                                                                               |
| COV-01    | Retire Go coverage gaps                               | [Package and file floors](../configs/quality/go-coverage-baseline.json): 165 package floors, **40 at zero**, and 7 file floors. Package names are enumerated in that file. The number of measured packages is not a debt count.                                                                                                                       | Add meaningful behavior tests for unproven owned behavior and tighten floors. A zero floor or lack of tests must be resolved explicitly; do not inflate coverage with implementation-mirroring assertions.                                                      |
| COV-02    | Measure remaining frontend scopes                     | [Coverage scope inventory](../.quality/coverage-scope.json): **14 unmeasured workspaces**, including web, admin, space, and shared packages. Existing behavior tests do not provide reliable source coverage for every scope.                                                                                                                         | Reliable attribution of source coverage, meaningful tests, and enforced floors for the recorded scopes. Preserve existing measured-suite floors.                                                                                                                |
| UI-01     | Finish recorded browser verification                  | [Test-suite review](test-suite-review.md) and the existing local browser evidence. Legacy-comment tooltip verification remains pending. Clipboard toast behavior was observed; the automation clipboard could not verify a paste round-trip.                                                                                                          | Verify the specific remaining user-visible behaviors and record the result. Do not treat automation limitations as proven application defects.                                                                                                                  |
| EXT-01    | Review retained dependency limitations                | [Development limitations and removal conditions](development.md#dependency-limitations), plus the Testcontainers/Colima guidance in [test commands](development.md#tests). Includes Ryuk reuse, Colima forwarding, bodyclose analysis, errorlint EOF handling, Go overlay coverage, Lagotto loading, and scoped dependency compatibility workarounds. | Use upstream reports and pinned consumer reproductions. Remove a workaround only when the consumer passes its removal condition. No dependency edits, forks, or publication without separate authorization.                                                     |
| EXT-02    | Revalidate recorded tooling warnings                  | Previously observed next-themes warning and an unconfirmed router CLI circular-dependency warning. Historical observations need current reproduction before changes.                                                                                                                                                                                  | Reproduce and locate the cause, then fix owned behavior or retain a reported upstream limitation with a removal condition. Close obsolete observations with evidence.                                                                                           |

The largest TypeScript categories at this snapshot are 351 unsafe type assertions,
264 explicit `any` uses, 129 inconsistent returns, 108 floating promises, 105
complexity findings, 78 shadowing findings, 68 console findings, 60 unsafe enum
comparisons, 52 misused promises, and 51 unstable nested components. The full
baseline contains the remaining categories and exact affected files.

## Concrete queued work

The next substantive task is **MCP-01**: reconcile the existing action-level
acceptance table with current test evidence, identify the exact missing cases,
and work through those cases by risk. Some tracker prose still describes earlier
quality-gate gaps even though current full gates pass. Reconciliation must not
close unrelated domain requirements.

The first acceptance batch covers workspace listing, selection and live access.
Its 62 focused leaf cases cover both credentials and protocols, membership
revocation, preservation of another workspace, and credential changes after
transport authentication. The [workspace acceptance evidence](mcp-parity.md#workspace-read-acceptance-mcp-01)
records the exact scope. That published batch changed no application code and
closed the recorded listing read-contract gap.

Identity and feature-setting acceptance now includes exact responses and storage proof:
41 focused leaf cases cover both credentials/protocols, native storage, tenant
isolation, role restrictions, receipt conflicts, native-writer revisions,
concurrent CAS, and audit-failure rollback. It also fixes a reproduced null-flag
compatibility defect: explicit null must preserve the flag and normalize like an
omitted field. [Feature acceptance evidence](mcp-parity.md#workspace-feature-acceptance-mcp-01-mcp-02)
records the source contract and checks. The next acceptance batch proves current
authorization after waiting for both workspace/project writes and replays,
private-project permissions, project-bound receipts and native revisions, and
atomic default-intake creation. Its focused regressions pass; full gates for this
batch must still pass before publication. The next project-feature gaps are
foreign-workspace targeting and archived-project policy reconciliation.
Other action rows and inventory categories remain open.

COV-01 floor review uses the completed full profile at the snapshot above:
`internal/features` increases from 80.92% to 83.21%, and `internal/mcpactions`
from 83.82% to 84.24%. Only these affected package floors change; no allowance is
increased. The new acceptance batch changes tests and documentation, not runtime
code.

The following runtime items are already members of ERR-01/ERR-02 and COV-01.
They are prepared follow-ups, not evidence of completed fixes:

| Item                               | Source                                                                                                                                      | Reason and required proof                                                                                                                                                                 | Status                                                                  |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| Worker readiness and shutdown      | [`cmd/background-worker/main.go`](../apps/go-api/cmd/background-worker/main.go)                                                             | Readiness output is unchecked; startup failure discards shutdown errors. Prove one shutdown, preserved primary/cleanup causes, usable bounded cleanup context, and graceful cancellation. | Source reviewed; draft regressions prepared; no implementation changes. |
| Metadata worker output and cleanup | [`cmd/durable-metadata/main.go`](../apps/go-api/cmd/durable-metadata/main.go)                                                               | Admission/readiness output and application shutdown errors can be discarded. Prove that output failure never repeats an admitted durable workflow.                                        | Source reviewed; no implementation changes.                             |
| Bootstrap migration cleanup        | [`internal/bootstrap/support.go`](../apps/go-api/internal/bootstrap/support.go), [`schema.go`](../apps/go-api/internal/bootstrap/schema.go) | Migration/source/connection cleanup errors can be discarded. Preserve adoption order, migration effects, and session-lock release while retaining cleanup causes.                         | Source reviewed; no implementation changes.                             |

## New findings awaiting verification and prioritization

These are recorded to prevent silent scope growth. They are not automatically
placed ahead of the existing catalog.

| ID     | Observation                                                                                                                      | Evidence strength                                                                                                     | Next decision                                                                                                                                                                              |
| ------ | -------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| NEW-01 | `parity-diff`'s action prompt ignores its callback context while waiting for stdin, despite the complete-operation timeout flag. | Source observation in [`responseAction`](../apps/go-api/cmd/parity-diff/main.go); no cancellation regression has run. | Prove the deadline behavior with controlled blocking input, then assess priority. Any fix needs explicit input ownership and must avoid leaked blocked reads or closing an unowned stream. |

## Completed recorded work

These are completed bounded audits or batches, not claims that whole subsystems
can never contain another defect. The [test-suite review](test-suite-review.md)
records behavior and limitations; Git history preserves the implementation.

| Work                                                 | Evidence                                                                                                                                                                                                                            |
| ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Multi-workspace MCP and accessible-workspace listing | `workspace.list` and per-call workspace selection are implemented; [MCP tracker](mcp-parity.md) links authentication/protocol regressions. Full action acceptance remains MCP-01.                                                   |
| Recorded numeric storage/writer audit                | Ordering, activity epochs, asset sizes, and hierarchy levels: migrations 43–46 and the writer regressions described in the [test review](test-suite-review.md). Commits `c28cb4c1`, `621117ba`, `a9e54dd5`, `3ada7087`, `655b3e6b`. |
| Recorded malformed-input error contracts             | Cycle pagination, page ordering, and project-member validation, with transport/storage regressions. Commits `e98d8d36`, `468dc922`, `c6104fc1`.                                                                                     |
| Audited request cleanup                              | Checked transaction cleanup and failure regressions in `ed07f5be`. Other cleanup findings remain ERR-01/ERR-02.                                                                                                                     |
| Audited clipboard callers                            | Rejection, retry, fallback cleanup, and scoped feedback in `d5cdb82d`. Browser evidence has the limitation recorded in UI-01.                                                                                                       |
| Metadata/webhook error causes                        | Preserved causes with sanitized transport behavior in `7bbd4190`.                                                                                                                                                                   |
| Route identifiers and shared issue headers           | `64727007`: 142 redundant conversions removed; full TypeScript findings 2,084 → 1,941; frontend clone pairs 264 → 263.                                                                                                              |
| CLI output and migration cleanup                     | `593a73b6`: six reproduced failures fixed; 28 focused cases, including PostgreSQL; Go findings 183 → 162. Coverage floors: parity-diff 62.5%, support-migrate 86.96%, shared output helper 100%.                                    |
| Requested Testcontainers fix pin                     | Core and PostgreSQL module use `v0.44.1-0.20261029064637-ff84c26717f1`; [development instructions](development.md#tests) link the upstream fix and removal condition. Separate reaper/forwarding limitations remain EXT-01.         |
| Stale Vector worktrees                               | Rechecked on 2026-10-30: only the current `main` checkout exists. No outdated Vector worktrees remain.                                                                                                                              |

## Product follow-ups and accepted deferrals

[GitHub issue #6](https://github.com/CaliLuke/vector/issues/6), “MCP: add discoverable
management and client-setup UI,” is still open as of this snapshot. It is a
product follow-up, not a lint finding. Keep it distinct from the sophisticated
permissions UI explicitly deferred by the user.

Accepted deferrals remain teamspaces, production OAuth issuer selection, and
sophisticated MCP permissions management UI. Do not silently reactivate them as
technical-debt cleanup.

## Completion rule

The goal remains open while catalog work lacks its required proof. A passing gate,
a reduced baseline, an implemented tool name, or an unexecuted test draft does not
close an item. Retire items only with evidence of the requested behavior and the
applicable checks. Keep stale measurements, upstream waits, product follow-ups,
and newly observed concerns explicitly labeled.
