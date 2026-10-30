# MCP mutation recovery

MCP mutations support optional write receipts and revision checks. Ordinary Plane-compatible calls remain valid without either extension.

`internal/mutationreceipts` provides the receipt transaction foundation. Archive and restore connect it to `project.archive` and `project.unarchive`, with generated optional extensions, project-row revision coverage, and sanitized mutation telemetry. The seven milestone actions also use the shared receipt boundary, with native association effects and revision guards. State create/update/delete also use this boundary, with all-writer revision triggers and atomic deletion outbox entries. Focused PostgreSQL and race tests cover these integrations. All catalogue mutations are wired. Receipt cleanup runs through the durable scheduler; local browser conflict checks passed. Full frontend and Go gates passed at `17d1afdb`; remaining resource acceptance and coverage work is recorded in the technical debt catalog.

## Transaction boundary

A domain application operation calls `Execute` on the primary database. Its authorization and mutation callbacks receive the same read-committed PostgreSQL transaction. The mutation must write its domain changes, activity, required outbox rows, and safe result before this transaction commits. It must not dispatch external effects or open a second write transaction. Existing domain authorization and effect behavior must be retained when adapting a writer.

The optional UUID `request_id` is scoped by actor, workspace, project (when applicable), tool, and action. A transaction-scoped advisory lock serializes concurrent calls with the same key. A lock-hash collision can delay an unrelated call but cannot make it match another receipt: the stored key comparison includes every scope field.

After acquiring that lock, the authorization callback revalidates the credential, checks current account, workspace, project, and action permissions, and retains the relevant domain authorization locks through commit. Credential validation before entering `Execute` is insufficient: a call may have waited behind another transaction. The receipt helper cannot supply those resource-specific rules. A destructive replay checks the surviving scope and action permissions without requiring a live target. Domains that need retained ownership information must preserve that authorization scope throughout the receipt lifetime.

An existing, unexpired receipt is examined only after authorization succeeds. Matching input returns its original safe result and expiry; changed input returns a conflict. Replay skips the mutation callback, so it precedes the optional `expected_revision` check. An expired key can be used for a new mutation. Receipt identities have no entity foreign keys, so target deletion does not erase retry protection or force a live entity lookup during replay.

## Input and result handling

The caller supplies the normalized, typed domain input after defaults, preserving omitted versus cleared fields and including `expected_revision` when present. It excludes `request_id` and all credentials. The helper normalizes JSON object ordering and whitespace, preserves integer precision, and stores only a SHA-256 digest. Domain-specific normalization, including numeric semantics, remains the caller's responsibility.

The result must be valid JSON containing only data safe for an authorized replay. PostgreSQL stores that result with the mutation and returns the same JSON representation on the initial call and replay. The receipt advertises its database-generated `replay_expires_at`, currently 48 hours after the receipt is written, above the required 24-hour minimum. Cleanup runs daily at 03:45 UTC through the existing durable scheduler, in batches of at most 500 rows. It compares expiry against both the fixed run cutoff and current database time, so even a future cutoff cannot remove an unexpired receipt. Row locks are skipped while another transaction replays or renews a receipt; renewed receipts remain eligible for their full advertised lifetime.

Without `request_id`, the operation uses the same transaction and authorization path but stores no receipt and returns no replay expiry. It is never retried automatically.

## Cancellation and acknowledgement

An authorization, mutation, serialization, or receipt-write failure rolls back the domain changes and durable effects. Cancellation detected before commit also rolls back. A successful commit is returned as success even if the caller's context is cancelled afterward.

A commit error returns `ErrIndeterminate` and no result. A network failure can hide a successful commit, so callers must not report rollback or retry an unkeyed mutation automatically. A keyed retry revalidates credentials and permissions, then recovers the stored result if the commit succeeded. An unkeyed caller must reconcile using reads. The public commit error omits database details; its underlying cause remains available for internal classification. This does not promise exactly-once delivery of external effects.

## Project archive and restore

Both actions require `project_id` and accept optional `request_id` and `expected_revision`. With neither extension, they retain the pinned handler's empty content result. With either extension, the result contains `project_id` and `revision`. A keyed call additionally returns its canonical `request_id`, `replay_expires_at`, and `replayed` flag. These fields are Vector extensions, not changes to ordinary SDK results.

Read the current opaque token with `project.retrieve` and `include_revision: true`. Normal detail results omit it. The token covers the project row; it does not version per-user favorites, member counts, or other derived detail fields. A PostgreSQL trigger maintains a separate counter for every project insert or update, including HTTP, worker, and collaboration writers. An update that leaves values unchanged can conservatively advance the token. The authorized detail and token use the same database snapshot. Archive locks the project, reads the latest counter, and compares before updating. Stale input returns `revision_conflict`; a matching stored receipt is replayed before that comparison.

Archive and favorite cleanup commit in the same transaction as the revision and optional receipt. Restore clears the archive timestamp and does not restore favorites. These operations retain the existing domain effect behavior: neither dispatches external activity or webhook work. Both HTTP routes use the same application writer while retaining their distinct existing member permissions and response shapes.

Discovery filters fresh copies of generated action schemas using current OAuth scopes and project-write permissions. Read discovery applies current workspace/project membership; guests cannot discover `member.list_workspace`. Milestone discovery also applies its guest-feature and archived-project policy. Read-only callers do not see write action branches or their extension inputs. Invocation still checks the specific target and permission in the transaction, including on replay. No principal-specific schema is written back to the shared SDK catalogue.

Mutation logs and spans contain the validated action, authenticated actor/client, selected workspace, safe outcome, duration, replay status, and committed project ID on success. They share request and trace correlation. They exclude credentials, argument/result bodies, receipt keys, revision inputs, and raw errors. An uncertain commit is recorded as `indeterminate`, without claiming a committed project ID.

## Evidence and remaining integration

Real PostgreSQL tests cover every receipt key dimension, canonical input, large integer conflicts, duplicate commits and rollback recovery, cancellation while waiting, late authorization revocation, deleted-target replay, expiration, and atomic rollback of receipt/domain/outbox failures. A TCP fixture drops PostgreSQL's successful COMMIT response before the driver receives it, proving both keyed recovery and truthful unkeyed uncertainty. Bootstrap tests install the migration in a fresh database and repeat the complete migration sequence.

Generated-server archive tests exercise real PostgreSQL with PAT and OAuth under both revisions: empty default results, opt-in revisions, replay before CAS, changed-input conflicts, stale-write rejection, permission loss on replay, and discovery isolation. Domain tests additionally prove authorization policies, favorite rollback, concurrent CAS, and revision advancement by other writers. Telemetry tests verify matching log/span correlation and redaction.

The receipt foundation alone does not establish resource parity. The [cleanup regression](../apps/go-api/internal/backgroundtasks/mutation_receipt_cleanup_integration_test.go) covers bounded deletion, future-cutoff protection, concurrent renewal, durable admission and timer registration against PostgreSQL. Local browser checks passed for work-item revision conflicts and page collaboration replacement. The [parity tracker](mcp-parity.md) records those checks and the six-domain receipt batch. Full gates passed at `17d1afdb`; the remaining coverage inventory and resource acceptance still require separate evidence.

Page and collection writes support the same optional receipt/revision extensions. Workspace revisions include all native page, project-link, collection, membership and item-page-link writers. Page body changes enqueue native transaction/version work atomically. See [page and collection contracts](mcp-pages-collections.md). Content workflows and the six-domain receipt batch pass under both authentication modes and protocol revisions; remaining acceptance is tracked in [MCP parity](mcp-parity.md).

Template, initiative and feature-setting mutations support optional receipt/revision extensions, current authorization before replay and transactional audit events. Initiative work-item link changes enqueue native association history. Catalogue workflows and receipt checks pass under both authentication modes and protocol revisions; the parity tracker records remaining acceptance.

Cycle and module delete receipts check current owner-or-project-administrator
authority before replay. Active project members retain owner-based delete rights;
losing administrator status removes rights over another user's resource.
Current administrators can replay even after a native cascade removes the target.
For owner-based replay, scheduled purge retains the deleted cycle/module while
an unexpired delete receipt references it. Expired receipts do not extend retention.
This protects the 48-hour receipt window even with a one-day purge policy.
The [planning delete checks](../apps/go-api/internal/mcpactions/planning_delete_authorization_integration_test.go)
and [purge regression](../apps/go-api/internal/backgroundtasks/planning_deletion_receipt_integration_test.go)
cover these boundaries. Complete publication gates for this batch remain pending.
