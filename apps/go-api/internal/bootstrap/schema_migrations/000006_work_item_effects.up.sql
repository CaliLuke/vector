BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- A consumed draft cannot be republished even after the draft cleanup job runs.
CREATE TABLE work_item_draft_publications (
 draft_id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 issue_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

-- Request transactions record effects here. The native durable worker admits
-- each effect with this stable identity before retiring its private payload.
CREATE TABLE work_item_effects (
 id uuid PRIMARY KEY,
 kind text NOT NULL CHECK (kind IN ('activity','description_version','soft_delete')),
 payload jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 next_attempt_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 attempts integer NOT NULL DEFAULT 0,
 admitted_at timestamptz,
 last_error text NOT NULL DEFAULT ''
);
CREATE INDEX work_item_effects_pending ON work_item_effects(next_attempt_at,created_at,id) WHERE admitted_at IS NULL;
COMMIT;
