-- The baseline migration clears the session search path. Scope these objects
-- explicitly without changing the connection state for later migrations.
BEGIN;
SET LOCAL search_path = public, pg_catalog;

CREATE TABLE work_item_type_settings (
 workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 enabled boolean NOT NULL DEFAULT false,
 hierarchy_enabled boolean NOT NULL DEFAULT false,
 default_level integer NOT NULL DEFAULT 0 CHECK (default_level BETWEEN 0 AND 20),
 revision bigint NOT NULL DEFAULT 0
);
ALTER TABLE issue_types ADD COLUMN mandatory boolean NOT NULL DEFAULT false;
ALTER TABLE project_issue_types ADD COLUMN is_active boolean NOT NULL DEFAULT true;
ALTER TABLE issues ADD COLUMN custom_properties jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE draft_issues ADD COLUMN custom_properties jsonb NOT NULL DEFAULT '{}'::jsonb;
CREATE TABLE work_item_properties (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 definition jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 created_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
 UNIQUE (workspace_id,id)
);
CREATE INDEX work_item_property_names ON work_item_properties(workspace_id,(definition->>'name'));
CREATE TABLE work_item_type_properties (
 type_id uuid NOT NULL REFERENCES issue_types(id) ON DELETE CASCADE,
 property_id uuid NOT NULL REFERENCES work_item_properties(id) ON DELETE CASCADE,
 sort_order integer NOT NULL DEFAULT 0,
 PRIMARY KEY(type_id,property_id)
);
CREATE TABLE work_item_property_versions (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
 property_id uuid NOT NULL,
 value jsonb NOT NULL,
 search_text text NOT NULL DEFAULT '',
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX work_item_property_versions_issue ON work_item_property_versions(issue_id,property_id,created_at DESC);
CREATE INDEX work_item_property_versions_search ON work_item_property_versions USING gin(to_tsvector('simple',search_text));
CREATE TABLE work_item_releases (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 name varchar(255) NOT NULL,
 description text NOT NULL DEFAULT '',
 target_date date,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(project_id,name)
);

-- Shared with the domain repository: schema edits and work-item writes serialize
-- per workspace so hierarchy changes cannot race parent or type changes.
CREATE FUNCTION work_item_schema_lock(workspace uuid) RETURNS void LANGUAGE sql AS $$
 SELECT pg_advisory_xact_lock(hashtextextended(workspace::text || ':work-item-schema',0));
$$;

CREATE FUNCTION effective_work_item_property(definition jsonb,project uuid,item_type uuid)
RETURNS jsonb LANGUAGE sql STABLE AS $$
 SELECT COALESCE((
  SELECT (definition->'configuration') || COALESCE((SELECT jsonb_object_agg(key,value) FROM jsonb_each(context->'configuration') WHERE context->'override_keys' ? key),'{}'::jsonb)
  FROM jsonb_array_elements(COALESCE(definition->'contexts','[]'::jsonb)) context
  WHERE (jsonb_array_length(context->'project_ids')=0 OR context->'project_ids' ? project::text)
    AND (jsonb_array_length(context->'type_ids')=0 OR context->'type_ids' ? item_type::text)
  ORDER BY ((jsonb_array_length(context->'project_ids')>0)::integer + (jsonb_array_length(context->'type_ids')>0)::integer) DESC
  LIMIT 1
 ),definition->'configuration');
$$;

CREATE FUNCTION enforce_work_item_schema() RETURNS trigger LANGUAGE plpgsql AS $$
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
CREATE TRIGGER work_item_schema_guard BEFORE INSERT OR UPDATE ON issues FOR EACH ROW EXECUTE FUNCTION enforce_work_item_schema();

CREATE FUNCTION import_required_work_item_types() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM work_item_type_settings WHERE workspace_id=NEW.workspace_id AND enabled) THEN
  INSERT INTO project_issue_types(id,created_at,updated_at,workspace_id,project_id,issue_type_id,is_default,level,is_active)
  SELECT gen_random_uuid(),clock_timestamp(),clock_timestamp(),NEW.workspace_id,NEW.id,id,is_default,level::integer,true
  FROM issue_types WHERE workspace_id=NEW.workspace_id AND deleted_at IS NULL AND (is_default OR mandatory);
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER project_required_work_item_types AFTER INSERT ON projects FOR EACH ROW EXECUTE FUNCTION import_required_work_item_types();

COMMIT;
