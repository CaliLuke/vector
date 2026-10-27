# MCP mutation recovery

The accepted [master contract](https://vector.tail7064c5.ts.net/vector/projects/4b4a3685-542a-4ed9-bef5-8fa9bec196ae/pages/f29707ed-60af-42cb-a173-cb69d43d9ed8?paneTab=info) and [VEC-4](https://vector.tail7064c5.ts.net/vector/browse/VEC-4/) require optional write receipts and revision checks. Ordinary Plane-compatible calls remain valid without either extension.

`internal/mutationreceipts` provides the receipt transaction foundation. It is not yet connected to a resource action. No MCP mutation is advertised or counted as implemented. Revision coverage, domain integration, receipt cleanup, generated extension fields, and mutation telemetry remain pending.

## Transaction boundary

A domain application operation calls `Execute` on the primary database. Its authorization and mutation callbacks receive the same read-committed PostgreSQL transaction. The mutation must write its domain changes, activity, required outbox rows, and safe result before this transaction commits. It must not dispatch external effects or open a second write transaction. Existing domain authorization and effect behavior must be retained when adapting a writer.

The optional UUID `request_id` is scoped by actor, workspace, project (when applicable), tool, and action. A transaction-scoped advisory lock serializes concurrent calls with the same key. A lock-hash collision can delay an unrelated call but cannot make it match another receipt: the stored key comparison includes every scope field.

After acquiring that lock, the authorization callback revalidates the credential, checks current account, workspace, project, and action permissions, and retains the relevant domain authorization locks through commit. Credential validation before entering `Execute` is insufficient: a call may have waited behind another transaction. The receipt helper cannot supply those resource-specific rules. A destructive replay checks the surviving scope and action permissions without requiring the deleted target to exist.

An existing, unexpired receipt is examined only after authorization succeeds. Matching input returns its original safe result and expiry; changed input returns a conflict. Replay skips the mutation callback, so it precedes the optional `expected_revision` check. An expired key can be used for a new mutation. Receipt identities have no entity foreign keys, so target deletion does not erase retry protection or force entity lookup during replay.

## Input and result handling

The caller supplies the normalized, typed domain input after defaults, preserving omitted versus cleared fields and including `expected_revision` when present. It excludes `request_id` and all credentials. The helper normalizes JSON object ordering and whitespace, preserves integer precision, and stores only a SHA-256 digest. Domain-specific normalization, including numeric semantics, remains the caller's responsibility.

The result must be valid JSON containing only data safe for an authorized replay. PostgreSQL stores that result with the mutation and returns the same JSON representation on the initial call and replay. The receipt advertises its database-generated `replay_expires_at`, currently 48 hours after the receipt is written, above the required 24-hour minimum. Cleanup must never remove an unexpired receipt.

Without `request_id`, the operation uses the same transaction and authorization path but stores no receipt and returns no replay expiry. It is never retried automatically.

## Cancellation and acknowledgement

An authorization, mutation, serialization, or receipt-write failure rolls back the domain changes and durable effects. Cancellation detected before commit also rolls back. A successful commit is returned as success even if the caller's context is cancelled afterward.

A commit error returns `ErrIndeterminate` and no result. A network failure can hide a successful commit, so callers must not report rollback or retry an unkeyed mutation automatically. A keyed retry revalidates credentials and permissions, then recovers the stored result if the commit succeeded. An unkeyed caller must reconcile using reads. The public commit error omits database details; its underlying cause remains available for internal classification. This does not promise exactly-once delivery of external effects.

## Evidence and remaining integration

Real PostgreSQL tests cover every receipt key dimension, canonical input, large integer conflicts, duplicate commits and rollback recovery, cancellation while waiting, late authorization revocation, deleted-target replay, expiration, and atomic rollback of receipt/domain/outbox failures. A TCP fixture drops PostgreSQL's successful COMMIT response before the driver receives it, proving both keyed recovery and truthful unkeyed uncertainty. Bootstrap tests install the migration in a fresh database and repeat the complete migration sequence.

Before advertising any mutation, its domain adapter still needs generated optional extension schemas, safe typed result/error mappings, both authentication modes and protocol revisions, revision checks covering every relevant writer, real domain/outbox integration, and correlated sanitized telemetry. The receipt foundation's test tables establish transaction behavior, not resource parity. Browser conflict checks and the complete VEC-4 acceptance evidence remain open.
