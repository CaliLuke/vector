BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TABLE work_log_edit_revisions;
DROP TRIGGER work_log_revision ON work_logs;
DROP FUNCTION advance_work_log_revision();
DROP TABLE work_logs;
COMMIT;
