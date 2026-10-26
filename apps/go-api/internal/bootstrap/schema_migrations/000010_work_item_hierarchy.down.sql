BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER aa_work_item_hierarchy_lock ON issues;
DROP TRIGGER aa_draft_hierarchy_lock ON draft_issues;
DROP TRIGGER aa_type_hierarchy_lock ON issue_types;
DROP TRIGGER aa_settings_hierarchy_lock ON work_item_type_settings;
DROP TRIGGER work_item_hierarchy_guard ON issues;
DROP TRIGGER draft_work_item_hierarchy_guard ON draft_issues;
DROP TRIGGER type_hierarchy_guard ON issue_types;
DROP TRIGGER settings_hierarchy_guard ON work_item_type_settings;
DROP TRIGGER work_item_hierarchy_settings_delete ON work_item_type_settings;
DROP FUNCTION protect_work_item_hierarchy_settings();
DROP FUNCTION enforce_work_item_hierarchy();
DROP FUNCTION enforce_type_work_item_hierarchy();
DROP FUNCTION enforce_workspace_work_item_hierarchy();
DROP FUNCTION lock_work_item_hierarchy();
DROP FUNCTION validate_workspace_work_item_hierarchy(uuid);
DROP FUNCTION validate_work_item_hierarchy(uuid,boolean);

CREATE OR REPLACE FUNCTION enforce_work_item_schema() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 settings work_item_type_settings%ROWTYPE;
 current_type issue_types%ROWTYPE;
 parent_type_level double precision;
 property record;
 config jsonb;
 value jsonb;
 changing_type boolean;
BEGIN
 SELECT * INTO settings FROM work_item_type_settings WHERE workspace_id=NEW.workspace_id;
 IF NOT FOUND OR NOT settings.enabled OR NEW.deleted_at IS NOT NULL THEN RETURN NEW; END IF;
 PERFORM work_item_schema_lock(NEW.workspace_id);
 IF NEW.type_id IS NULL THEN
  SELECT id INTO NEW.type_id FROM issue_types WHERE workspace_id=NEW.workspace_id AND is_default AND deleted_at IS NULL ORDER BY created_at LIMIT 1;
 END IF;
 SELECT * INTO current_type FROM issue_types WHERE id=NEW.type_id AND workspace_id=NEW.workspace_id AND deleted_at IS NULL;
 IF NOT FOUND THEN RAISE EXCEPTION 'The work item type is not available in this workspace' USING ERRCODE='23514'; END IF;
 changing_type := TG_OP='INSERT';
 IF TG_OP='UPDATE' THEN changing_type := NEW.type_id IS DISTINCT FROM OLD.type_id OR NEW.project_id IS DISTINCT FROM OLD.project_id; END IF;
 IF changing_type AND (NOT current_type.is_active OR NOT (
  current_type.is_default OR current_type.mandatory OR EXISTS(SELECT 1 FROM project_issue_types WHERE project_id=NEW.project_id AND issue_type_id=NEW.type_id AND deleted_at IS NULL AND is_active)
 )) THEN RAISE EXCEPTION 'The work item type is inactive or has not been imported into this project' USING ERRCODE='23514'; END IF;
 IF settings.hierarchy_enabled THEN
  IF NEW.parent_id IS NOT NULL THEN
   SELECT t.level INTO parent_type_level FROM issues i JOIN issue_types t ON t.id=i.type_id
   WHERE i.id=NEW.parent_id AND i.workspace_id=NEW.workspace_id AND i.project_id=NEW.project_id AND i.deleted_at IS NULL;
   IF NOT FOUND OR parent_type_level<>current_type.level+1 THEN
    RAISE EXCEPTION 'The parent must be from the immediately higher hierarchy level' USING ERRCODE='23514';
   END IF;
  END IF;
  IF EXISTS(SELECT 1 FROM issues child JOIN issue_types t ON t.id=child.type_id WHERE child.parent_id=NEW.id AND child.deleted_at IS NULL AND t.level<>current_type.level-1) THEN
   RAISE EXCEPTION 'The type does not match the existing child hierarchy' USING ERRCODE='23514';
  END IF;
 END IF;
 IF COALESCE(current_setting('vector.schema_migration',true),'')<>'on' THEN
  FOR property IN SELECT p.id,p.definition FROM work_item_properties p JOIN work_item_type_properties link ON link.property_id=p.id WHERE link.type_id=NEW.type_id AND p.workspace_id=NEW.workspace_id LOOP
   config := effective_work_item_property(property.definition,NEW.project_id,NEW.type_id);
   value := NEW.custom_properties->property.id::text;
   IF (config->>'active')::boolean AND (config->>'required')::boolean AND (value IS NULL OR value='null'::jsonb OR value='""'::jsonb OR value='[]'::jsonb) THEN
    RAISE EXCEPTION 'Required custom property is missing: %',property.definition->>'title' USING ERRCODE='23514';
   END IF;
  END LOOP;
 END IF;
 RETURN NEW;
END;
$$;

COMMIT;
