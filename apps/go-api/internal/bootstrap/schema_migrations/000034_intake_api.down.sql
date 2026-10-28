BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER intake_issue_edit_revision ON intake_issues;
COMMIT;
