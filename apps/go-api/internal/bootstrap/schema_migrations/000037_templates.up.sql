BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE workspace_template_settings(
 workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 owned_by_workspace boolean NOT NULL DEFAULT false,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0)
);
INSERT INTO workspace_template_settings(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION seed_template_settings() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO workspace_template_settings(workspace_id) VALUES(NEW.id);RETURN NEW;END; $$;
CREATE TRIGGER workspace_template_settings AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION seed_template_settings();
CREATE TABLE resource_templates(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,project_id uuid,
 kind text NOT NULL CHECK(kind IN('workitem','page','project')),name varchar(255) NOT NULL CHECK(length(btrim(name))>0),
 summary text NOT NULL DEFAULT '',template_data jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(template_data)='object'),logo_props jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(project_id,workspace_id) REFERENCES projects(id,workspace_id) ON DELETE CASCADE,
 CHECK(kind<>'project' OR project_id IS NULL),UNIQUE(id,workspace_id)
);
CREATE INDEX resource_templates_scope ON resource_templates(workspace_id,kind,project_id,created_at,id) WHERE deleted_at IS NULL;
CREATE TABLE template_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,template_id uuid NOT NULL,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,action text NOT NULL,before_value jsonb,after_value jsonb,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION advance_template_revision() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE old_workspace uuid;new_workspace uuid;
BEGIN
 IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id;END IF;
 IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id;END IF;
 PERFORM workspace_id FROM workspace_template_settings WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE workspace_template_settings SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);
 RETURN NULL;
END; $$;
CREATE TRIGGER resource_template_revision AFTER INSERT OR UPDATE OR DELETE ON resource_templates FOR EACH ROW EXECUTE FUNCTION advance_template_revision();
CREATE FUNCTION template_governance_revision() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.owned_by_workspace IS DISTINCT FROM OLD.owned_by_workspace THEN NEW.revision:=OLD.revision+1;END IF;RETURN NEW;END; $$;
CREATE TRIGGER template_governance_revision BEFORE UPDATE ON workspace_template_settings FOR EACH ROW EXECUTE FUNCTION template_governance_revision();
COMMIT;
