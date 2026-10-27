BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- Keep identities even when the affected entity is deleted. Returning a receipt
-- always requires current authorization; these identifiers never grant access.
CREATE TABLE mutation_receipts (
 actor_id uuid NOT NULL,
 workspace_id uuid NOT NULL,
 project_id uuid,
 tool text NOT NULL CHECK (tool <> ''),
 action text NOT NULL CHECK (action <> ''),
 request_id uuid NOT NULL,
 input_hash bytea NOT NULL CHECK (octet_length(input_hash) = 32),
 result jsonb NOT NULL,
 created_at timestamptz NOT NULL,
 replay_expires_at timestamptz NOT NULL,
 UNIQUE NULLS NOT DISTINCT (actor_id, workspace_id, project_id, tool, action, request_id),
 CHECK (replay_expires_at >= created_at + INTERVAL '24 hours')
);
CREATE INDEX mutation_receipts_expiry ON mutation_receipts(replay_expires_at);
COMMIT;
