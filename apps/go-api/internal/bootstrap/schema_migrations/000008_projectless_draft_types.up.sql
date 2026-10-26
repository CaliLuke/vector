BEGIN;
SET LOCAL search_path = public, pg_catalog;
CREATE OR REPLACE FUNCTION enforce_draft_work_item_type() RETURNS trigger LANGUAGE plpgsql AS $$
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
  NEW.project_id IS NULL OR selected_type.is_default OR selected_type.mandatory OR EXISTS(SELECT 1 FROM project_issue_types WHERE project_id=NEW.project_id AND issue_type_id=NEW.type_id AND workspace_id=NEW.workspace_id AND deleted_at IS NULL AND is_active)
 )) THEN RAISE EXCEPTION 'The work item type is inactive or has not been imported into this project' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END;
$$;
COMMIT;
