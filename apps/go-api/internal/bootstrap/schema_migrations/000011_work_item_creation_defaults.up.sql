BEGIN;
SET LOCAL search_path = public, pg_catalog;

CREATE TABLE work_item_project_defaults (
 project_id uuid PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 type_id uuid NOT NULL REFERENCES issue_types(id) ON DELETE CASCADE
);
CREATE FUNCTION enforce_work_item_project_default() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM work_item_schema_lock(NEW.workspace_id);
 IF NOT EXISTS(SELECT 1 FROM projects p JOIN issue_types t ON t.workspace_id=p.workspace_id
  WHERE p.id=NEW.project_id AND p.workspace_id=NEW.workspace_id AND t.id=NEW.type_id AND p.deleted_at IS NULL AND t.deleted_at IS NULL) THEN
  RAISE EXCEPTION 'The project default type must belong to this workspace' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER work_item_project_default_tenant BEFORE INSERT OR UPDATE ON work_item_project_defaults FOR EACH ROW EXECUTE FUNCTION enforce_work_item_project_default();

-- Import only unambiguous preferences. An arbitrary legacy first row is not a
-- user choice, and optional-import removal must not erase a saved preference.
INSERT INTO work_item_project_defaults(project_id,workspace_id,type_id)
SELECT p.id,p.workspace_id,(array_agg(DISTINCT t.id))[1]
FROM projects p JOIN project_issue_types imported ON imported.project_id=p.id AND imported.workspace_id=p.workspace_id
JOIN issue_types t ON t.id=imported.issue_type_id AND t.workspace_id=p.workspace_id
WHERE p.deleted_at IS NULL AND imported.deleted_at IS NULL AND imported.is_default AND t.deleted_at IS NULL
GROUP BY p.id,p.workspace_id HAVING count(DISTINCT t.id)=1;

CREATE OR REPLACE FUNCTION enforce_work_item_schema() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 settings work_item_type_settings%ROWTYPE;
 current_type issue_types%ROWTYPE;
 property record;
 config jsonb;
 value jsonb;
 changing_type boolean;
BEGIN
 SELECT * INTO settings FROM work_item_type_settings WHERE workspace_id=NEW.workspace_id;
 IF NOT FOUND OR NOT settings.enabled OR NEW.deleted_at IS NOT NULL THEN RETURN NEW; END IF;
 PERFORM work_item_schema_lock(NEW.workspace_id);
 IF NEW.type_id IS NULL THEN
  RAISE EXCEPTION 'Select a work item type before saving' USING ERRCODE='23514';
 END IF;
 SELECT * INTO current_type FROM issue_types WHERE id=NEW.type_id AND workspace_id=NEW.workspace_id AND deleted_at IS NULL;
 IF NOT FOUND THEN RAISE EXCEPTION 'The work item type is not available in this workspace' USING ERRCODE='23514'; END IF;
 changing_type := TG_OP='INSERT';
 IF TG_OP='UPDATE' THEN changing_type := NEW.type_id IS DISTINCT FROM OLD.type_id OR NEW.project_id IS DISTINCT FROM OLD.project_id; END IF;
 IF changing_type AND (NOT current_type.is_active OR NOT (
  current_type.is_default OR current_type.mandatory OR EXISTS(SELECT 1 FROM project_issue_types WHERE project_id=NEW.project_id AND issue_type_id=NEW.type_id AND deleted_at IS NULL AND is_active)
 )) THEN RAISE EXCEPTION 'The work item type is inactive or has not been imported into this project' USING ERRCODE='23514'; END IF;
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

CREATE OR REPLACE FUNCTION enforce_draft_work_item_type() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 selected_type issue_types%ROWTYPE;
 changing_type boolean;
BEGIN
 PERFORM work_item_schema_lock(NEW.workspace_id);
 IF NOT EXISTS(SELECT 1 FROM work_item_type_settings WHERE workspace_id=NEW.workspace_id AND enabled) THEN RETURN NEW; END IF;
 IF NEW.type_id IS NULL THEN
  IF COALESCE(NEW.custom_properties,'{}'::jsonb)<>'{}'::jsonb THEN
   RAISE EXCEPTION 'Select a work item type before saving custom properties' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
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
CREATE OR REPLACE FUNCTION validate_work_item_hierarchy(item uuid, is_draft boolean)
RETURNS void LANGUAGE plpgsql VOLATILE AS $$
DECLARE
 node record;
 parent_node record;
 configured boolean;
 node_level double precision;
BEGIN
 IF is_draft THEN
  SELECT id,workspace_id,type_id,parent_id INTO node FROM draft_issues WHERE id=item;
 ELSE
  SELECT id,workspace_id,type_id,parent_id INTO node FROM issues WHERE id=item;
 END IF;
 -- A deferred event may refer to a row subsequently deleted in this transaction.
 IF NOT FOUND THEN RETURN; END IF;
 SELECT COALESCE(hierarchy_enabled,false) INTO configured FROM work_item_type_settings WHERE workspace_id=node.workspace_id;
 IF node.type_id IS NOT NULL AND EXISTS(SELECT 1 FROM issue_types WHERE id=node.type_id AND workspace_id IS DISTINCT FROM node.workspace_id) THEN
  RAISE EXCEPTION 'The work item type must belong to the same workspace' USING ERRCODE='23514';
 END IF;
 SELECT COALESCE((SELECT level FROM issue_types WHERE id=node.type_id AND workspace_id=node.workspace_id),0) INTO node_level;
 IF node.parent_id IS NOT NULL THEN
  SELECT i.id,i.workspace_id,COALESCE(t.level,0) AS level INTO parent_node
  FROM issues i LEFT JOIN issue_types t ON t.id=i.type_id AND t.workspace_id=i.workspace_id WHERE i.id=node.parent_id;
  IF NOT FOUND OR parent_node.workspace_id IS DISTINCT FROM node.workspace_id THEN
   RAISE EXCEPTION 'The parent must belong to the same workspace' USING ERRCODE='23514';
  END IF;
  IF NOT is_draft AND node.parent_id=node.id THEN
   RAISE EXCEPTION 'A work item cannot be its own parent' USING ERRCODE='23514';
  END IF;
  IF (
   WITH RECURSIVE ancestors AS (
    SELECT i.id,i.parent_id FROM issues i WHERE i.id=node.parent_id
    UNION
    SELECT i.id,i.parent_id FROM ancestors a JOIN issues i ON i.id=a.parent_id
   ) SELECT EXISTS(SELECT 1 FROM ancestors WHERE NOT is_draft AND id=node.id)
    OR NOT EXISTS(SELECT 1 FROM ancestors a WHERE a.parent_id IS NULL OR NOT EXISTS(SELECT 1 FROM issues i WHERE i.id=a.parent_id))
  ) THEN
   RAISE EXCEPTION 'Parent relationships cannot contain a cycle' USING ERRCODE='23514';
  END IF;
  IF configured AND NOT (is_draft AND node.type_id IS NULL) AND parent_node.level<>node_level+1 THEN
   RAISE EXCEPTION 'The parent must be from the immediately higher hierarchy level' USING ERRCODE='23514';
  END IF;
 END IF;
 IF NOT is_draft AND EXISTS(
  SELECT 1 FROM (
   SELECT workspace_id,type_id,false AS draft FROM issues WHERE parent_id=node.id
   UNION ALL SELECT workspace_id,type_id,true AS draft FROM draft_issues WHERE parent_id=node.id
  ) child LEFT JOIN issue_types t ON t.id=child.type_id AND t.workspace_id=child.workspace_id
  WHERE child.workspace_id IS DISTINCT FROM node.workspace_id OR (configured AND NOT (child.draft AND child.type_id IS NULL) AND COALESCE(t.level,0)<>node_level-1)
 ) THEN
  RAISE EXCEPTION 'The type does not match the existing child hierarchy' USING ERRCODE='23514';
 END IF;
END;
$$;


COMMIT;
