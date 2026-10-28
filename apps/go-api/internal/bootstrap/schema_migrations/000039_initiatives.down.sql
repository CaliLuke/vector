BEGIN;
SET LOCAL search_path = public, pg_catalog;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM initiatives) THEN RAISE EXCEPTION 'cannot discard initiative data';END IF;END; $$;
DROP TABLE initiative_events,initiative_projects,initiative_work_items,initiatives,initiative_revisions;
DROP TRIGGER initiative_workspace_revision ON workspaces;
DROP FUNCTION seed_initiative_revision(),advance_initiative_revision();
COMMIT;
