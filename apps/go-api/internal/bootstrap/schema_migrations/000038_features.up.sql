BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE workspace_features(
 workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 project_grouping boolean NOT NULL DEFAULT false,initiatives boolean NOT NULL DEFAULT false,teams boolean NOT NULL DEFAULT false,
 customers boolean NOT NULL DEFAULT false,wiki boolean NOT NULL DEFAULT false,pi boolean NOT NULL DEFAULT false,
 revision bigint NOT NULL DEFAULT 1 CHECK(revision>0)
);
INSERT INTO workspace_features(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION seed_workspace_features() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO workspace_features(workspace_id) VALUES(NEW.id);RETURN NEW;END; $$;
CREATE TRIGGER workspace_features AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION seed_workspace_features();
ALTER TABLE projects ADD COLUMN epics_enabled boolean NOT NULL DEFAULT false,ADD COLUMN parallel_cycles_enabled boolean NOT NULL DEFAULT false,ADD COLUMN project_updates_enabled boolean NOT NULL DEFAULT false,ADD COLUMN workflows_enabled boolean NOT NULL DEFAULT false;
CREATE FUNCTION advance_workspace_features() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.revision:=OLD.revision+1;RETURN NEW;END; $$;
CREATE TRIGGER workspace_features_revision BEFORE UPDATE ON workspace_features FOR EACH ROW EXECUTE FUNCTION advance_workspace_features();
CREATE FUNCTION advance_project_features() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN UPDATE workspace_features SET revision=revision WHERE workspace_id IN(OLD.workspace_id,NEW.workspace_id);RETURN NULL;END; $$;
CREATE TRIGGER project_features_revision AFTER UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION advance_project_features();
CREATE TABLE feature_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,project_id uuid,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,before_value jsonb NOT NULL,after_value jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
COMMIT;
