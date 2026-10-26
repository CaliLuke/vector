BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Project creation and rollout edits must observe one another, even when concurrent.
CREATE OR REPLACE FUNCTION import_required_work_item_types() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM work_item_schema_lock(NEW.workspace_id);
 IF EXISTS(SELECT 1 FROM work_item_type_settings WHERE workspace_id=NEW.workspace_id AND enabled) THEN
  INSERT INTO project_issue_types(id,created_at,updated_at,workspace_id,project_id,issue_type_id,is_default,level,is_active)
  SELECT gen_random_uuid(),clock_timestamp(),clock_timestamp(),NEW.workspace_id,NEW.id,id,is_default,level::integer,true
  FROM issue_types WHERE workspace_id=NEW.workspace_id AND deleted_at IS NULL AND (is_default OR mandatory);
 END IF;
 RETURN NEW;
END;
$$;
-- Draft assignments share the type lock and availability rules with published work.
CREATE FUNCTION enforce_draft_work_item_type() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 selected_type issue_types%ROWTYPE;
 changing_type boolean;
BEGIN
 PERFORM work_item_schema_lock(NEW.workspace_id);
 IF NOT EXISTS(SELECT 1 FROM work_item_type_settings WHERE workspace_id=NEW.workspace_id AND enabled) THEN RETURN NEW; END IF;
 IF NEW.type_id IS NULL THEN
  SELECT id INTO NEW.type_id FROM issue_types WHERE workspace_id=NEW.workspace_id AND is_default AND deleted_at IS NULL ORDER BY created_at LIMIT 1;
 END IF;
 SELECT * INTO selected_type FROM issue_types WHERE id=NEW.type_id AND workspace_id=NEW.workspace_id AND deleted_at IS NULL;
 IF NOT FOUND THEN RAISE EXCEPTION 'The work item type is not available in this workspace' USING ERRCODE='23514'; END IF;
 changing_type := TG_OP='INSERT';
 IF TG_OP='UPDATE' THEN changing_type := NEW.type_id IS DISTINCT FROM OLD.type_id OR NEW.project_id IS DISTINCT FROM OLD.project_id; END IF;
 IF changing_type AND (NOT selected_type.is_active OR NOT (
  selected_type.is_default OR selected_type.mandatory OR EXISTS(SELECT 1 FROM project_issue_types WHERE project_id=NEW.project_id AND issue_type_id=NEW.type_id AND workspace_id=NEW.workspace_id AND deleted_at IS NULL AND is_active)
 )) THEN RAISE EXCEPTION 'The work item type is inactive or has not been imported into this project' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER draft_work_item_type_guard BEFORE INSERT OR UPDATE ON draft_issues FOR EACH ROW EXECUTE FUNCTION enforce_draft_work_item_type();
COMMIT;
