BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER label_edit_revision ON labels;
DROP FUNCTION advance_label_edit_revision();
DROP TABLE label_edit_revisions;
COMMIT;
