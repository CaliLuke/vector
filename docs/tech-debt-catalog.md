# Technical debt catalog

This is the authoritative queue for the requested retirement of all known debt.
It links the exact finding inventories, states the next acceptance work, and
records the evidence needed to close each category. It does not claim an
exhaustive audit of every possible defect. Counts overlap; they are not an
estimate of equally sized tasks or time remaining.

## Current state

Last verified published revision: `3f044c7a3915c86c54f93041631af73fd143d860`,
2026-10-30. Both full quality gates passed, including all 13,960 Go test events
across 118 test-bearing packages, with no failures or skipped tests. Remote
`main` was verified at that revision.

| Catalog IDs                     | Published work                                            | Evidence                                                                               | Status                                                                            |
| ------------------------------- | --------------------------------------------------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| MCP-01, MCP-02                  | Cycle/module scope, receipts, contention and cancellation | Authorization batch at `97660842`; 31 additional focused cases at `6ba02d1d`           | Published; all 20 planning actions and the PQL reference meet recorded acceptance |
| MCP-01, MCP-02 / NEW-03, NEW-04 | Calendar boundaries and transfer snapshot isolation       | `a2e44d6f`: 34 focused cases, fast gates and both full gates pass                      | Both reproduced defects fixed and published                                       |
| ERR-01, ERR-02                  | Worker and migration output/cleanup errors                | `6ba02d1d`: 35 focused cases; configured findings 162 → 156; strict findings 945 → 933 | This bounded batch is published; remaining inventories stay open                  |
| COV-01                          | Worker CLI and module coverage floors                     | Worker floors 55% and 30.58%; module floor 65.51% → 67.84%                             | Published and enforced; no floor lowered                                          |

The query batch published at `81009902` adds 32 planning query cases and four PQL reference
cases. All 36 cases pass against the applied source (57 test events), with no failures
or skipped cases. The Go fast gate passes; duplication remains at 474 reports and maintainability
at 17 allowances, with no added baseline entries. Both full gates pass at `81009902`. They cover both credentials
and protocols, ordinary and custom-property filtering, pagination, projection,
parent/workspace binding, current membership, typed errors and reference defaults.
The native browser acceptance also passes: creation visibility, native edits,
revision conflicts, MCP edits, archive/restore/delete replay and list visibility.
Both fixture lists exactly match their original empty baselines, and all project
feature settings are restored. See [the browser evidence](mcp-cycles.md#native-browser-acceptance).

NEW-01 is fixed and published at `3f044c7a`. Its 22 prompt tests pass in both the
focused race run and the full suite; the enforced package floor is 68.07%.

Keep the checkout unchanged while publication hooks run. Earlier full commands
passed at `1772ca79`, but that push was rejected because tracked documentation
changed during hooks. Its changes are now included in published `a2e44d6f`.

## Ordered execution queue

1. **ERR-01/ERR-02: preserve eight API cache cleanup errors.** The exact
   `cmd/system/main.go` entries are state writes, public metadata throttling,
   assets, label writes, invitations, authentication Redis, authentication
   throttling, and account writes. These discarded errors can hide a cleanup
   failure or lose it behind the original startup/serving error. Preserve both
   causes and resource identity, keep reverse cleanup order, and prove that one
   failed close does not prevent the others. The five regression cases and all
   2,326 affected-package race test events pass.
   The configured inventory falls 156 → 155 and the strict inventory 933 → 925.
   COV-01 raises the package floor 73.73% → 74.67%. Finish fast gates and publication
   before closing this batch.
2. **MCP-01: reconcile five member reads and workspace listing.** The recorded
   read contract now has 198 passing test events across 32 relevant test roots,
   including native HTTP adapter checks. Live proxy comparisons cover all six
   actions, seven role details, filters, pagination and foreign-role rejection.
   Publish the [acceptance record](mcp-members.md#member-read-acceptance-mcp-01)
   and six tracker updates with this batch.
3. **MCP-01/MCP-02: select the next documented domain gap** from
   [the tracker](mcp-parity.md). Prioritize current authorization, tenant isolation
   and durable mutation guarantees, and record the bounded batch before edits.
4. **STRUCT-01, DUP-01/DUP-02, COV-02 and TS-01:** work through the remaining
   bounded components after the higher-risk acceptance and runtime gaps. Do not
   raise baselines or treat a changed fingerprint as retirement.

Completed-batch details are in the [progress log](tech-debt-progress.md).
The inventory below remains open even when a current batch passes its gates.

## Work selection

The next task must name a catalog ID, an observable problem or missing proof, and
the evidence needed to close it. After a verified change, update its status and
counts. Do not start another cleanup merely because it is adjacent to an edit.
Record newly discovered concerns in the intake section first, with their evidence
and impact. A source observation is not a reproduced defect.

Dependency limitations and product follow-ups remain visible separately. They
must not silently become dependency development or new product scope.

## Remaining catalog

Counts are findings or allowances, not equally sized tasks. Do not add overlapping
counts to produce a single "tasks remaining" total.

| ID        | Work                                      | Current evidence and scope                                                                                                                                                                                                                                                                                                                            | Closure evidence                                                                                                                                                                                                                                                |
| --------- | ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| MCP-01    | Complete MCP acceptance                   | [Action-by-action tracker](mcp-parity.md) covers 208 actions across 30 tools, including `workspace.list`. Workflow wiring is implemented; compatibility, scope, concurrency, replay, revisions, and durable-effect acceptance remain to be reconciled.                                                                                                | Each action has inspected contract evidence and the applicable generated-server, authorization, tenant, receipt/CAS, effect, and storage proofs. Update stale tracker statements against current tests; do not count a green full gate as missing domain proof. |
| MCP-02    | Complete mutation guarantees              | [Write contract and recovery requirements](mcp-writes.md), plus mutation rows in MCP-01. This overlaps MCP-01.                                                                                                                                                                                                                                        | Current authorization on replay, request-hash conflicts, relevant native-writer revisions, contention/cancellation/uncertain commits, and durable effects are proven for the documented contracts.                                                              |
| ERR-01    | Resolve configured Go findings            | [Exact file/message inventory](../configs/quality/go-lint-baseline.json): **155** occurrences: 51 errcheck, 78 staticcheck, 13 errorlint, 7 bodyclose, 6 nilerr. Some analyzer findings have documented upstream limitations.                                                                                                                         | Fix the underlying behavior or remove an obsolete finding with evidence. No increased allowances, blanket suppressions, or source exclusions. Retained upstream limitations stay in EXT-01.                                                                     |
| ERR-02    | Retire stricter unchecked-error inventory | [Exact file/line inventory](unchecked-error-inventory.json), refreshed for the API cache-cleanup batch based on `3f044c7a`: **925** findings with `errcheck.check-blank: true`; **51 overlap ERR-01**, leaving **874 additional findings**. The report contains 551 test-file and 374 non-test-file findings.                                         | Resolve each retained error according to resource/effect ownership and refresh the inventory. Findings are not automatically defects; no blanket suppression or unchecked-error wrapper. Do not count the 51 configured findings twice.                         |
| TS-01     | Resolve TypeScript findings               | [Full inventory](../.quality/oxlint-full-baseline.json): **1,941** occurrences. [Fast inventory](../.quality/oxlint-fast-baseline.json): 1,062 overlapping occurrences, not additional debt.                                                                                                                                                          | Resolve each recorded finding while preserving contracts. Compiler errors cannot be baselined. Prioritize unsafe data and asynchronous behavior before mechanical style cleanup.                                                                                |
| STRUCT-01 | Retire Go maintainability exceptions      | [Named functions and scores](../configs/quality/go-maintainability-baseline.json): **17** exceptions, including server assembly and large test functions.                                                                                                                                                                                             | Coherent responsibilities, effect ordering, and narrower interfaces; no score regressions or movement into unmeasured code. Remove allowances only after review.                                                                                                |
| DUP-01    | Consolidate Go duplication                | [Fingerprint inventory](../configs/quality/go-dupl-baseline.json): **474 directed reports**, represented by 365 fingerprints.                                                                                                                                                                                                                         | Consolidate duplicated responsibilities with behavior checks; reduce the reviewed baseline without adding allowances. Directed reports are not 474 independent refactors.                                                                                       |
| DUP-02    | Consolidate frontend duplication          | [Fingerprint inventory](../.quality/jscpd-baseline.json): **263 clone pairs**, represented by 262 fingerprints.                                                                                                                                                                                                                                       | Shared behavior/components replace duplicated ownership, with focused consumer tests and no new clone allowances.                                                                                                                                               |
| COV-01    | Retire Go coverage gaps                   | [Package and file floors](../configs/quality/go-coverage-baseline.json): 165 package floors, **38 at zero**, and 7 file floors. Package names are enumerated in that file. The number of measured packages is not a debt count.                                                                                                                       | Add meaningful behavior tests for unproven owned behavior and tighten floors. A zero floor or lack of tests must be resolved explicitly; do not inflate coverage with implementation-mirroring assertions.                                                      |
| COV-02    | Measure remaining frontend scopes         | [Coverage scope inventory](../.quality/coverage-scope.json): **14 unmeasured workspaces**, including web, admin, space, and shared packages. Existing behavior tests do not provide reliable source coverage for every scope.                                                                                                                         | Reliable attribution of source coverage, meaningful tests, and enforced floors for the recorded scopes. Preserve existing measured-suite floors.                                                                                                                |
| UI-01     | Finish recorded browser verification      | [Test-suite review](test-suite-review.md) and the existing local browser evidence. Legacy-comment tooltip verification remains pending. Clipboard toast behavior was observed; the automation clipboard could not verify a paste round-trip.                                                                                                          | Verify the specific remaining user-visible behaviors and record the result. Do not treat automation limitations as proven application defects.                                                                                                                  |
| EXT-01    | Review retained dependency limitations    | [Development limitations and removal conditions](development.md#dependency-limitations), plus the Testcontainers/Colima guidance in [test commands](development.md#tests). Includes Ryuk reuse, Colima forwarding, bodyclose analysis, errorlint EOF handling, Go overlay coverage, Lagotto loading, and scoped dependency compatibility workarounds. | Use upstream reports and pinned consumer reproductions. Remove a workaround only when the consumer passes its removal condition. No dependency edits, forks, or publication without separate authorization.                                                     |
| EXT-02    | Revalidate recorded tooling warnings      | Previously observed next-themes warning and an unconfirmed router CLI circular-dependency warning. Historical observations need current reproduction before changes.                                                                                                                                                                                  | Reproduce and locate the cause, then fix owned behavior or retain a reported upstream limitation with a removal condition. Close obsolete observations with evidence.                                                                                           |

The largest TypeScript categories at this snapshot are 351 unsafe type assertions,
264 explicit `any` uses, 129 inconsistent returns, 108 floating promises, 105
complexity findings, 78 shadowing findings, 68 console findings, 60 unsafe enum
comparisons, 52 misused promises, and 51 unstable nested components. The full
baseline contains the remaining categories and exact affected files.

## Intake and resolved findings

These records preserve the intake decision and closure evidence. NEW-02 remains
open. NEW-01, NEW-03 and NEW-04 are fixed and published. New observations
are not automatically placed ahead of the existing catalog.

| ID     | Observation                                                                                                                      | Evidence strength                                                                                                                                                          | Next decision                                                                                                                               |
| ------ | -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| NEW-01 | `parity-diff`'s action prompt ignores its callback context while waiting for stdin, despite the complete-operation timeout flag. | Both controlled cancellation and deadline cases reproduce a blocked read. The applied helper-process fix passes 22 race-enabled test events, with 68.07% package coverage. | Enter/EOF, cancellation/deadline, input ownership and error reporting checks pass. Fixed and published at `3f044c7a`; both full gates pass. |

**NEW-03 — cycle completion/archive calendar boundaries:**
Before the fix, [`MutateResource`](../apps/go-api/internal/cyclereads/mutations.go)
synthesized a UTC date while the native parser used the current actor's timezone.
Archive compared date strings when deciding whether to end a running cycle.
Four of six initial frozen-clock cases reproduced the defect. The published fix
uses the actor's calendar day and compares end timestamps; seven new calendar
cases and seven existing native date cases pass against the applied source. The
fix is published at `a2e44d6f`: 34 focused cases and both full gates pass. [Upstream #266](https://github.com/makeplane/plane-mcp-server/issues/266)
records the date-only running check with a standalone helper reproduction.
Keep the corrected instant comparison when reconciling a later upstream handler;
only remove a compatibility adjustment when the pinned handler and local
same-day/DST/replay regressions agree. This belongs to MCP-01/MCP-02's existing
cycle contract acceptance.

**NEW-04 — cycle transfer snapshot tenant isolation:**
Before the fix, [`transferSnapshot`](../apps/go-api/internal/workitems/cycle_transfer.go)
counted foreign work items and incorrectly scoped relation rows in a cycle's
saved progress snapshot. Four generated-server regressions (both credentials and
protocols) reproduced a total of seven for five valid local associations. The
writer transferred the expected three unfinished local items and preserved the
foreign records, but the saved and returned snapshot was incorrect. The fix
binds snapshot summary, distribution and chart scope to the selected project
and workspace while preserving native transfer behavior.
This is part of MCP-01/MCP-02's existing planning isolation acceptance. The
regression and fix are published at `a2e44d6f`. Four transport cases pass for both issue and estimate totals, distributions and
charts, retained associations and receipt replay. The combined calendar, native
date and existing core workflow run also passes. The 34 focused cases and both full gates pass at `a2e44d6f`.

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
