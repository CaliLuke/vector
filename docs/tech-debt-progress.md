# Technical debt progress log

The [catalog](tech-debt-catalog.md) owns the current queue and remaining scope.
These notes preserve dated batch evidence; historical pending statuses do not
override the catalog. Git history preserves the source and exact earlier reports.

## Earlier acceptance and runtime batches

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

| Batch                                        | Required evidence                                                                                                                                                                     | Status                                                                                                                                                                                                                                                                                                                   |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Planning receipts and rollback               | Changed input conflicts; other writers invalidate old revisions; replay preserves later edits; failed association/effect writes roll back relations, revisions and receipts.          | 52 MCP cases pass across both credentials, protocols and domains, including native relation-writer revisions and lifecycle replay. Three deletion-worker cases also pass. Published with complete gates at `97660842`.                                                                                                   |
| Planning authorization and contention        | Current roles/lifecycle and credentials govern fresh writes and replay; workspace/project/resource bindings hold; competing requests and revisions admit only the documented commits. | 72 role/lifecycle and 40 lock-wait cases pass across both credentials/protocols and domains. Eight native module-delete authorization cases pass. These are published at `97660842`. Another 16 isolation, six contention and two cancellation cases pass against applied source; full publication gates remain pending. |
| Planning browser and contract reconciliation | Native/MCP edits and lifecycle changes agree through the local proxy; stale cycle documentation matches implemented actions.                                                          | Existing partial browser evidence; remaining checks open.                                                                                                                                                                                                                                                                |

The planning batch fixes a reproduced MCP-02 gap: an administrator could delete
another user's cycle/module, lose that role, and still replay the delete receipt.
Current owner/admin authority is now checked before replay. Current administrators
can also replay after the target disappears through a native owner cascade.
Scheduled purge retains owner authorization rows during the receipt's retry window,
then releases them after expiry. The focused run passes 52 MCP cases plus two purge
cases and the existing durable cascade/retention regression, with no failures or
skips. The first full publication attempt passed all 13,660 Go test events and the
TypeScript gate, but stopped at the module coverage floor (65.28% versus 65.51%).
Direct native module-delete authorization coverage and the current-role/lock-wait
batch resolved that gap without lowering a floor. Both full gates passed at
`97660842`, including all 13,814 Go test events; remote `main` was verified there.

The next MCP-01/MCP-02 batch is applied locally. Sixteen transport cases cover
workspace/project/resource and receipt binding, plus preservation of foreign
work-item associations under both credentials and protocols. Six repository cases
cover identical and conflicting request keys and competing revisions. Two more
prove cancellation during a lock wait leaves no mutation or receipt, permits a
changed request with the same key, and preserves later replay. Three existing
label contention cases and four milestone fixture cases also pass through the
applied source: 31 leaf cases in total. Full publication gates remain required. The browser/contract gap and NEW-03 remain
open.

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
Their fixes are applied and focused tests pass; full gates remain pending:

| Item                               | Source                                                                                                                                      | Reason and required proof                                                                                                                                                                 | Status                                                                                                                             |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Worker readiness and shutdown      | [`cmd/background-worker/main.go`](../apps/go-api/cmd/background-worker/main.go)                                                             | Readiness output is unchecked; startup failure discards shutdown errors. Prove one shutdown, preserved primary/cleanup causes, usable bounded cleanup context, and graceful cancellation. | Two failures reproduced; five lifecycle cases pass against applied source. Full gates pending.                                     |
| Metadata worker output and cleanup | [`cmd/durable-metadata/main.go`](../apps/go-api/cmd/durable-metadata/main.go)                                                               | Admission/readiness output and application shutdown errors can be discarded. Prove that output failure never repeats an admitted durable workflow.                                        | Output, workflow and cleanup causes covered by ten passing applied-source cases. Full gates pending.                               |
| Bootstrap migration cleanup        | [`internal/bootstrap/support.go`](../apps/go-api/internal/bootstrap/support.go), [`schema.go`](../apps/go-api/internal/bootstrap/schema.go) | Migration/source/connection cleanup errors can be discarded. Preserve adoption order, migration effects, and session-lock release while retaining cleanup causes.                         | Twelve cleanup cases, five PostgreSQL migration/adoption cases and URL validation pass against applied source. Full gates pending. |

The completed full profile at `1772ca79` also raises COV-01's
`internal/modulereads` floor from 65.51% to 67.84% (981 covered statements out of
1,446). The measured module source is unchanged in the calendar/snapshot batch.
No package or file floor is lowered.

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

## Planning browser acceptance, 2026-10-30

MCP-01/MCP-02: [cycle/module browser acceptance](mcp-cycles.md#native-browser-acceptance)
passes against the local proxy. Native edits advanced revisions; stale MCP writes
failed without changes; current writes and lifecycle receipt replay passed.
Both dedicated fixtures were deleted, final list responses matched their original
empty baselines, and all project feature settings were restored. The query and
reference test commit is `f47720fd`; its full publication gate remains pending.

## Planning acceptance published, 2026-10-30

`81009902` is verified on remote main. Both full gates pass, with 13,951 exact
run/pass Go test events across 118 test-bearing packages, no failures and no
skipped tests. The 20 cycle/module rows and `get_pql_reference` are reconciled
against their contract, authorization/scope, receipt/revision, effect, query and
browser evidence. Those rows are closed; the broader MCP inventory remains open.

NEW-01 now has controlled cancellation and deadline reproductions. The applied fix
uses a helper process and passes 22 race-enabled test events, with 68.07% package
coverage (113/166 statements). Its floor increases from 62.5% to 68.07%. It preserves stdin ownership, Enter/EOF behavior and read diagnostics.
The full publication gate remains pending. No dependency
is added. A cancellation-library candidate was rejected after a standalone macOS
EOF reproduction; [upstream #30](https://github.com/muesli/cancelreader/issues/30)
records that independent problem. No dependency source was changed.
