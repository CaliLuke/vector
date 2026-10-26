BEGIN;
SET LOCAL search_path = public, pg_catalog;

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
  SELECT id INTO NEW.type_id FROM issue_types WHERE workspace_id=NEW.workspace_id AND is_default AND deleted_at IS NULL ORDER BY created_at LIMIT 1;
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

-- Hierarchy constraints are separate from required-field validation. Only these
-- named constraints may be deferred by a reviewed graph/configuration change.
CREATE FUNCTION validate_work_item_hierarchy(item uuid, is_draft boolean)
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
  IF configured AND parent_node.level<>node_level+1 THEN
   RAISE EXCEPTION 'The parent must be from the immediately higher hierarchy level' USING ERRCODE='23514';
  END IF;
 END IF;
 IF NOT is_draft AND EXISTS(
  SELECT 1 FROM (
   SELECT workspace_id,type_id FROM issues WHERE parent_id=node.id
   UNION ALL SELECT workspace_id,type_id FROM draft_issues WHERE parent_id=node.id
  ) child LEFT JOIN issue_types t ON t.id=child.type_id AND t.workspace_id=child.workspace_id
  WHERE child.workspace_id IS DISTINCT FROM node.workspace_id OR (configured AND COALESCE(t.level,0)<>node_level-1)
 ) THEN
  RAISE EXCEPTION 'The type does not match the existing child hierarchy' USING ERRCODE='23514';
 END IF;
END;
$$;

CREATE FUNCTION validate_workspace_work_item_hierarchy(workspace uuid)
RETURNS void LANGUAGE plpgsql VOLATILE AS $$
DECLARE node record;
BEGIN
 FOR node IN SELECT id,false AS draft FROM issues WHERE workspace_id=workspace
 UNION ALL SELECT id,true FROM draft_issues WHERE workspace_id=workspace LOOP
  PERFORM validate_work_item_hierarchy(node.id,node.draft);
 END LOOP;
END;
$$;

CREATE FUNCTION lock_work_item_hierarchy() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF current_setting('transaction_isolation')<>'read committed' THEN
  RAISE EXCEPTION 'Work item graph writes require READ COMMITTED isolation' USING ERRCODE='25000';
 END IF;
 -- Application writers take this lock before row locks. This guard also covers
 -- direct SQL writers; graph checks execute after the statement with fresh state.
 IF TG_OP='UPDATE' AND NEW.workspace_id IS DISTINCT FROM OLD.workspace_id THEN
  RAISE EXCEPTION 'A work item schema record cannot move between workspaces' USING ERRCODE='23514';
 END IF;
 IF TG_TABLE_NAME='work_item_type_settings' AND TG_OP='UPDATE' THEN
  IF OLD.hierarchy_enabled AND NOT NEW.hierarchy_enabled THEN
   RAISE EXCEPTION 'An activated hierarchy cannot be disabled' USING ERRCODE='23514';
  END IF;
 END IF;
 PERFORM work_item_schema_lock(NEW.workspace_id);
 RETURN NEW;
END;
$$;

CREATE FUNCTION protect_work_item_hierarchy_settings() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM work_item_schema_lock(OLD.workspace_id);
 IF OLD.hierarchy_enabled AND EXISTS(SELECT 1 FROM workspaces WHERE id=OLD.workspace_id) THEN
  RAISE EXCEPTION 'An activated hierarchy cannot be disabled' USING ERRCODE='23514';
 END IF;
 RETURN OLD;
END;
$$;
CREATE TRIGGER work_item_hierarchy_settings_delete BEFORE DELETE ON work_item_type_settings FOR EACH ROW EXECUTE FUNCTION protect_work_item_hierarchy_settings();

CREATE FUNCTION enforce_work_item_hierarchy() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM validate_work_item_hierarchy(NEW.id,TG_TABLE_NAME='draft_issues');
 RETURN NULL;
END;
$$;
CREATE FUNCTION enforce_type_work_item_hierarchy() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE node record;
BEGIN
 IF TG_OP='UPDATE' THEN
  IF OLD.level IS NOT DISTINCT FROM NEW.level THEN RETURN NULL; END IF;
 END IF;
 FOR node IN SELECT id,false AS draft FROM issues WHERE type_id=NEW.id
 UNION ALL SELECT id,true FROM draft_issues WHERE type_id=NEW.id LOOP
  PERFORM validate_work_item_hierarchy(node.id,node.draft);
 END LOOP;
 RETURN NULL;
END;
$$;
CREATE FUNCTION enforce_workspace_work_item_hierarchy() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM validate_workspace_work_item_hierarchy(NEW.workspace_id);
 RETURN NULL;
END;
$$;

CREATE TRIGGER aa_work_item_hierarchy_lock BEFORE INSERT OR UPDATE ON issues FOR EACH ROW EXECUTE FUNCTION lock_work_item_hierarchy();
CREATE TRIGGER aa_draft_hierarchy_lock BEFORE INSERT OR UPDATE ON draft_issues FOR EACH ROW EXECUTE FUNCTION lock_work_item_hierarchy();
CREATE TRIGGER aa_type_hierarchy_lock BEFORE INSERT OR UPDATE ON issue_types FOR EACH ROW EXECUTE FUNCTION lock_work_item_hierarchy();
CREATE TRIGGER aa_settings_hierarchy_lock BEFORE INSERT OR UPDATE ON work_item_type_settings FOR EACH ROW EXECUTE FUNCTION lock_work_item_hierarchy();
CREATE CONSTRAINT TRIGGER work_item_hierarchy_guard AFTER INSERT OR UPDATE ON issues DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION enforce_work_item_hierarchy();
CREATE CONSTRAINT TRIGGER draft_work_item_hierarchy_guard AFTER INSERT OR UPDATE ON draft_issues DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION enforce_work_item_hierarchy();
CREATE CONSTRAINT TRIGGER type_hierarchy_guard AFTER INSERT OR UPDATE OF level ON issue_types DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION enforce_type_work_item_hierarchy();
CREATE CONSTRAINT TRIGGER settings_hierarchy_guard AFTER INSERT OR UPDATE OF hierarchy_enabled ON work_item_type_settings DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION enforce_workspace_work_item_hierarchy();

COMMIT;
