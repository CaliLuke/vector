BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER draft_work_item_type_guard ON draft_issues;
DROP FUNCTION enforce_draft_work_item_type();
CREATE OR REPLACE FUNCTION import_required_work_item_types() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM work_item_type_settings WHERE workspace_id=NEW.workspace_id AND enabled) THEN
  INSERT INTO project_issue_types(id,created_at,updated_at,workspace_id,project_id,issue_type_id,is_default,level,is_active)
  SELECT gen_random_uuid(),clock_timestamp(),clock_timestamp(),NEW.workspace_id,NEW.id,id,is_default,level::integer,true
  FROM issue_types WHERE workspace_id=NEW.workspace_id AND deleted_at IS NULL AND (is_default OR mandatory);
 END IF;
 RETURN NEW;
END;
$$;
COMMIT;
