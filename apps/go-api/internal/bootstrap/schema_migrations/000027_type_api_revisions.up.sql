BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE TABLE work_item_type_api_revisions(workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,revision bigint NOT NULL DEFAULT 1 CHECK(revision>0));
INSERT INTO work_item_type_api_revisions(workspace_id) SELECT id FROM workspaces;
CREATE FUNCTION initialize_work_item_type_api_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
BEGIN INSERT INTO work_item_type_api_revisions(workspace_id) VALUES(NEW.id); RETURN NEW; END;
$$;
CREATE TRIGGER workspace_type_api_revision AFTER INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION initialize_work_item_type_api_revision();
CREATE FUNCTION advance_work_item_type_api_revision() RETURNS trigger LANGUAGE plpgsql SET search_path=public,pg_catalog AS $$
DECLARE old_workspace uuid; new_workspace uuid;
BEGIN
 IF TG_OP<>'INSERT' THEN old_workspace:=OLD.workspace_id; END IF;
 IF TG_OP<>'DELETE' THEN new_workspace:=NEW.workspace_id; END IF;
 PERFORM workspace_id FROM work_item_type_api_revisions WHERE workspace_id IN(old_workspace,new_workspace) ORDER BY workspace_id FOR UPDATE;
 UPDATE work_item_type_api_revisions SET revision=revision+1 WHERE workspace_id IN(old_workspace,new_workspace);
 RETURN NULL;
END;
$$;
CREATE TRIGGER type_api_revision AFTER INSERT OR UPDATE OR DELETE ON issue_types FOR EACH ROW EXECUTE FUNCTION advance_work_item_type_api_revision();
CREATE TRIGGER project_type_api_revision AFTER INSERT OR UPDATE OR DELETE ON project_issue_types FOR EACH ROW EXECUTE FUNCTION advance_work_item_type_api_revision();
CREATE TRIGGER type_settings_api_revision AFTER INSERT OR UPDATE OR DELETE ON work_item_type_settings FOR EACH ROW EXECUTE FUNCTION advance_work_item_type_api_revision();
COMMIT;
