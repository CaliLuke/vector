BEGIN;
SET LOCAL search_path = public, pg_catalog;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM resource_templates) THEN RAISE EXCEPTION 'cannot roll back template storage containing data';END IF;END; $$;
DROP TABLE template_events,resource_templates;
DROP TRIGGER workspace_template_settings ON workspaces;
DROP FUNCTION seed_template_settings(),advance_template_revision();
DROP TABLE workspace_template_settings;
DROP FUNCTION template_governance_revision();
COMMIT;
