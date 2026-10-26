BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE work_item_schema_reviews (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 command jsonb NOT NULL,
 fingerprint text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 committed_at timestamptz
);
CREATE INDEX work_item_schema_reviews_workspace ON work_item_schema_reviews(workspace_id,created_at);
COMMIT;
