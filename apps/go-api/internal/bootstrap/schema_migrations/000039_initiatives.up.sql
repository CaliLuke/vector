BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE initiatives(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name varchar(255) NOT NULL CHECK(length(btrim(name))>0),description_html text,description_stripped text,description text,description_binary bytea,
 start_date date,end_date date,logo_props jsonb NOT NULL DEFAULT '{}',state text NOT NULL DEFAULT 'DRAFT' CHECK(state IN('DRAFT','PLANNED','ACTIVE','COMPLETED','CLOSED')),
 lead_id uuid REFERENCES users(id) ON DELETE SET NULL,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 UNIQUE(id,workspace_id),CHECK(start_date IS NULL OR end_date IS NULL OR start_date<=end_date)
);
CREATE TABLE initiative_projects(
 initiative_id uuid NOT NULL,project_id uuid NOT NULL,workspace_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,PRIMARY KEY(initiative_id,project_id),
 FOREIGN KEY(initiative_id,workspace_id) REFERENCES initiatives(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(project_id,workspace_id) REFERENCES projects(id,workspace_id) ON DELETE CASCADE
);
CREATE TABLE initiative_work_items(
 initiative_id uuid NOT NULL,issue_id uuid NOT NULL,project_id uuid NOT NULL,workspace_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,PRIMARY KEY(initiative_id,issue_id),
 FOREIGN KEY(initiative_id,workspace_id) REFERENCES initiatives(id,workspace_id) ON DELETE CASCADE,
 FOREIGN KEY(issue_id,project_id,workspace_id) REFERENCES issues(id,project_id,workspace_id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE TABLE initiative_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,initiative_id uuid NOT NULL,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,action text NOT NULL,input jsonb NOT NULL,before_value jsonb,after_value jsonb,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE initiative_revisions(workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO initiative_revisions(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION seed_initiative_revision() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO initiative_revisions(workspace_id) VALUES(NEW.id);RETURN NEW;END; $$;
CREATE TRIGGER initiative_workspace_revision AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION seed_initiative_revision();
CREATE FUNCTION advance_initiative_revision() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE old_workspace uuid;new_workspace uuid;
BEGIN IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id;END IF;IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id;END IF;
 PERFORM workspace_id FROM initiative_revisions WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE initiative_revisions SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);RETURN NULL;END; $$;
CREATE TRIGGER initiative_revision AFTER INSERT OR UPDATE OR DELETE ON initiatives FOR EACH ROW EXECUTE FUNCTION advance_initiative_revision();
CREATE TRIGGER initiative_project_revision AFTER INSERT OR UPDATE OR DELETE ON initiative_projects FOR EACH ROW EXECUTE FUNCTION advance_initiative_revision();
CREATE TRIGGER initiative_item_revision AFTER INSERT OR UPDATE OR DELETE ON initiative_work_items FOR EACH ROW EXECUTE FUNCTION advance_initiative_revision();
CREATE TRIGGER initiative_item_edit_revision AFTER INSERT OR UPDATE OR DELETE ON initiative_work_items FOR EACH ROW EXECUTE FUNCTION advance_issue_assignment_revision();
COMMIT;
