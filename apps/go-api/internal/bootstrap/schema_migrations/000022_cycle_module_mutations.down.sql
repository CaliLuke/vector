BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER cycle_edit_revision ON cycles;
DROP FUNCTION advance_cycle_edit_revision();
DROP TABLE cycle_edit_revisions;
DROP TRIGGER module_edit_revision ON modules;
DROP FUNCTION advance_module_edit_revision();
DROP TABLE module_edit_revisions;
COMMIT;
