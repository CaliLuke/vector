BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE work_item_property_search (
 issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
 property_id uuid NOT NULL REFERENCES work_item_properties(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 source_hash text NOT NULL CHECK(length(source_hash)=64),
 search_text text NOT NULL,
 PRIMARY KEY(issue_id,property_id)
);
CREATE INDEX work_item_property_search_workspace ON work_item_property_search(workspace_id,issue_id);
CREATE TABLE work_item_search_backfills (
 workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 completed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
COMMIT;
