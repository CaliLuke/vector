-- The baseline migration clears the session search path. Scope these objects
-- explicitly without changing the connection state for later migrations.
BEGIN;
SET LOCAL search_path = public, pg_catalog;

DROP TRIGGER project_required_work_item_types ON projects;
DROP FUNCTION import_required_work_item_types();
DROP TRIGGER work_item_schema_guard ON issues;
DROP FUNCTION enforce_work_item_schema();
DROP FUNCTION effective_work_item_property(jsonb,uuid,uuid);
DROP FUNCTION work_item_schema_lock(uuid);
DROP TABLE work_item_releases;
DROP TABLE work_item_property_versions;
DROP TABLE work_item_type_properties;
DROP TABLE work_item_properties;
ALTER TABLE draft_issues DROP COLUMN custom_properties;
ALTER TABLE issues DROP COLUMN custom_properties;
ALTER TABLE project_issue_types DROP COLUMN is_active;
ALTER TABLE issue_types DROP COLUMN mandatory;
DROP TABLE work_item_type_settings;

COMMIT;
