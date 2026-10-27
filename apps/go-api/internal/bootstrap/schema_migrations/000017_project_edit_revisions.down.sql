BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER project_edit_revision ON projects;
DROP FUNCTION advance_project_edit_revision();
DROP TABLE project_edit_revisions;
COMMIT;
