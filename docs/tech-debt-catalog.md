# Technical debt catalog

This is the working catalog for the requested retirement of all known technical
debt. It consolidates the quality baselines, MCP acceptance tracker, test review,
and previously local work log. It does not claim an exhaustive audit of every
possible defect.

Snapshot: `1be8fcd5e84318be223af9ea49022d70248a80b8`, 2026-10-30.
Both full quality gates passed at this published revision, including 13,567 Go test events
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

| ID        | Work                                      | Current evidence and scope                                                                                                                                                                                                                                                                                                                            | Closure evidence                                                                                                                                                                                                                                                |
| --------- | ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| MCP-01    | Complete MCP acceptance                   | [Action-by-action tracker](mcp-parity.md) covers 208 actions across 30 tools, including `workspace.list`. Workflow wiring is implemented; compatibility, scope, concurrency, replay, revisions, and durable-effect acceptance remain to be reconciled.                                                                                                | Each action has inspected contract evidence and the applicable generated-server, authorization, tenant, receipt/CAS, effect, and storage proofs. Update stale tracker statements against current tests; do not count a green full gate as missing domain proof. |
| MCP-02    | Complete mutation guarantees              | [Write contract and recovery requirements](mcp-writes.md), plus mutation rows in MCP-01. This overlaps MCP-01.                                                                                                                                                                                                                                        | Current authorization on replay, request-hash conflicts, relevant native-writer revisions, contention/cancellation/uncertain commits, and durable effects are proven for the documented contracts.                                                              |
| ERR-01    | Resolve configured Go findings            | [Exact file/message inventory](../configs/quality/go-lint-baseline.json): **162** occurrences: 58 errcheck, 78 staticcheck, 13 errorlint, 7 bodyclose, 6 nilerr. Some analyzer findings have documented upstream limitations.                                                                                                                         | Fix the underlying behavior or remove an obsolete finding with evidence. No increased allowances, blanket suppressions, or source exclusions. Retained upstream limitations stay in EXT-01.                                                                     |
| ERR-02    | Retire stricter unchecked-error inventory | [Exact file/line inventory](unchecked-error-inventory.json), refreshed at `264d505f`: **945** findings with `errcheck.check-blank: true`; **58 overlap ERR-01**, leaving **887 additional findings**. The report contains 551 test-file and 394 non-test-file findings.                                                                               | Resolve each retained error according to resource/effect ownership and refresh the inventory. Findings are not automatically defects; no blanket suppression or unchecked-error wrapper. Do not count the 58 configured findings twice.                         |
| TS-01     | Resolve TypeScript findings               | [Full inventory](../.quality/oxlint-full-baseline.json): **1,941** occurrences. [Fast inventory](../.quality/oxlint-fast-baseline.json): 1,062 overlapping occurrences, not additional debt.                                                                                                                                                          | Resolve each recorded finding while preserving contracts. Compiler errors cannot be baselined. Prioritize unsafe data and asynchronous behavior before mechanical style cleanup.                                                                                |
| STRUCT-01 | Retire Go maintainability exceptions      | [Named functions and scores](../configs/quality/go-maintainability-baseline.json): **17** exceptions, including server assembly and large test functions.                                                                                                                                                                                             | Coherent responsibilities, effect ordering, and narrower interfaces; no score regressions or movement into unmeasured code. Remove allowances only after review.                                                                                                |
| DUP-01    | Consolidate Go duplication                | [Fingerprint inventory](../configs/quality/go-dupl-baseline.json): **474 directed reports**, represented by 365 fingerprints.                                                                                                                                                                                                                         | Consolidate duplicated responsibilities with behavior checks; reduce the reviewed baseline without adding allowances. Directed reports are not 474 independent refactors.                                                                                       |
| DUP-02    | Consolidate frontend duplication          | [Fingerprint inventory](../.quality/jscpd-baseline.json): **263 clone pairs**, represented by 262 fingerprints.                                                                                                                                                                                                                                       | Shared behavior/components replace duplicated ownership, with focused consumer tests and no new clone allowances.                                                                                                                                               |
| COV-01    | Retire Go coverage gaps                   | [Package and file floors](../configs/quality/go-coverage-baseline.json): 165 package floors, **40 at zero**, and 7 file floors. Package names are enumerated in that file. The number of measured packages is not a debt count.                                                                                                                       | Add meaningful behavior tests for unproven owned behavior and tighten floors. A zero floor or lack of tests must be resolved explicitly; do not inflate coverage with implementation-mirroring assertions.                                                      |
| COV-02    | Measure remaining frontend scopes         | [Coverage scope inventory](../.quality/coverage-scope.json): **14 unmeasured workspaces**, including web, admin, space, and shared packages. Existing behavior tests do not provide reliable source coverage for every scope.                                                                                                                         | Reliable attribution of source coverage, meaningful tests, and enforced floors for the recorded scopes. Preserve existing measured-suite floors.                                                                                                                |
| UI-01     | Finish recorded browser verification      | [Test-suite review](test-suite-review.md) and the existing local browser evidence. Legacy-comment tooltip verification remains pending. Clipboard toast behavior was observed; the automation clipboard could not verify a paste round-trip.                                                                                                          | Verify the specific remaining user-visible behaviors and record the result. Do not treat automation limitations as proven application defects.                                                                                                                  |
| EXT-01    | Review retained dependency limitations    | [Development limitations and removal conditions](development.md#dependency-limitations), plus the Testcontainers/Colima guidance in [test commands](development.md#tests). Includes Ryuk reuse, Colima forwarding, bodyclose analysis, errorlint EOF handling, Go overlay coverage, Lagotto loading, and scoped dependency compatibility workarounds. | Use upstream reports and pinned consumer reproductions. Remove a workaround only when the consumer passes its removal condition. No dependency edits, forks, or publication without separate authorization.                                                     |
| EXT-02    | Revalidate recorded tooling warnings      | Previously observed next-themes warning and an unconfirmed router CLI circular-dependency warning. Historical observations need current reproduction before changes.                                                                                                                                                                                  | Reproduce and locate the cause, then fix owned behavior or retain a reported upstream limitation with a removal condition. Close obsolete observations with evidence.                                                                                           |

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
records the source contract and checks. The `264d505f` batch adds current
authorization after waiting for workspace/project writes and replays,
private-project permissions, project-bound receipts and native revisions, and
atomic default-intake creation. This batch is published with the archive-read fix
at `87a791d9`; both complete quality gates passed. The earlier unsuccessful push
recorded the intermittent UI timeout in NEW-02 below.

The published feature batch closes the foreign-workspace and archive gaps. It fixes
archived settings reads to match native project retrieval while preserving write
and replay rejection until restore. Its 102 focused leaf cases also cover current
membership and competing project-feature writes. Its complete publication gates
passed at `87a791d9`. The five state actions also completed their local
browser/proxy check against that rebuilt application, supplementing fifteen
reviewed passing server, storage and durable-effect test roots. The
[state acceptance record](mcp-parity.md#state-acceptance-mcp-01-mcp-02) closes
that domain's recorded gap: edits in both directions, stale-revision rejection,
deletion/replay and exact preservation of the five initial states.
Other action rows and inventory categories remain open.

The subsequent [label acceptance batch](mcp-modules-labels.md#label-mutation-acceptance)
has 75 passing label cases. Transport cases cover both credentials and protocols;
repository cases cover concurrent requests and revisions. Four related feature
cases also pass, giving 79 focused cases with no failures or skips. These cover
receipts, native-writer revisions, atomic cascade effects, current roles/lifecycle,
authorization changes after lock waits, tenant/resource binding, request contention
and competing revisions. Browser/proxy checks also passed in both edit directions,
including stale-revision rejection and deletion/replay.
Both full publication gates passed at `1be8fcd5`, including 13,567 Go test events
with no failures or skips. Remote `main` matches that commit. Duplication remains
at 474 reports, with no new allowances. The five label action rows are closed.

The active MCP-01/MCP-02 batch is cycle/module acceptance. Existing read and
lifecycle workflow tests cover both credentials and protocols. The next required
proof is explicit:

| Batch                                        | Required evidence                                                                                                                                                                     | Status                                                                                                                                                                                                      |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Planning receipts and rollback               | Changed input conflicts; other writers invalidate old revisions; replay preserves later edits; failed association/effect writes roll back relations, revisions and receipts.          | 52 MCP cases pass across both credentials, protocols and domains, including native relation-writer revisions and lifecycle replay. Three deletion-worker cases also pass. Complete batch gates remain open. |
| Planning authorization and contention        | Current roles/lifecycle and credentials govern fresh writes and replay; workspace/project/resource bindings hold; competing requests and revisions admit only the documented commits. | 72 role/lifecycle and 40 lock-wait cases pass across both credentials/protocols and domains. Eight native module-delete authorization cases pass. Resource-binding and contention proof remain open.        |
| Planning browser and contract reconciliation | Native/MCP edits and lifecycle changes agree through the local proxy; stale cycle documentation matches implemented actions.                                                          | Existing partial browser evidence; remaining checks open.                                                                                                                                                   |

The planning batch fixes a reproduced MCP-02 gap: an administrator could delete
another user's cycle/module, lose that role, and still replay the delete receipt.
Current owner/admin authority is now checked before replay. Current administrators
can also replay after the target disappears through a native owner cascade.
Scheduled purge retains owner authorization rows during the receipt's retry window,
then releases them after expiry. The focused run passes 52 MCP cases plus two purge
cases and the existing durable cascade/retention regression, with no failures or
skips. The first full publication attempt passed all 13,660 Go test events and the
TypeScript gate, but stopped at the module coverage floor (65.28% versus 65.51%).
The fix remains local. The next MCP-01/MCP-02/COV-01 batch adds current-role and
lock-wait authorization proofs, plus direct native module-delete authorization
coverage now that MCP uses the shared transactional authorization path. No floor
is lowered; complete publication gates remain required.

These batches refine existing catalog entries. They do not introduce new product
scope or replace the remaining inventory.

The published `1be8fcd5` full profile also supports stricter COV-01 floors:
`internal/mcpactions` rises from 84.24% to 85.05%, and
`internal/projectconfiguration` from 55.16% to 60.72%. Only these two measured
package floors change. File floors and other package allowances are preserved.

COV-01 floor review uses the completed full profile at `17d1afdb`:
`internal/features` increases from 80.92% to 83.21%, and `internal/mcpactions`
from 83.82% to 84.24%. Only these affected package floors change; no allowance is
increased. The authorization-wait batch at `264d505f` changed tests and documentation;
the following archive-read fix changes the owned feature scope query.

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

The earlier publication check at `264d505f` recorded **NEW-02**: the UI
color-picker pending-save regression timed out at its existing five-second limit;
the coverage summary was then absent. The unchanged isolated test and all 73 UI
tests under the same resource limits pass on recheck, as does the UI coverage
ratchet. The cause is not established, so no application defect or dependency bug
is claimed. Keep this observation open for reproduction; no test deadline or
coverage floor was relaxed. The complete retry at `87a791d9` passed both full
gates and all 13,482 Go test events and was published. This removes the publication
block, but does not establish the intermittent timeout's cause; keep the
observation open for reproduction.

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
