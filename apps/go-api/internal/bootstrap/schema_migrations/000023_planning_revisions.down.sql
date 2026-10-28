BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER module_member_edit_revision ON module_members;
DROP TRIGGER module_issue_edit_revision ON module_issues;
DROP TRIGGER cycle_issue_edit_revision ON cycle_issues;
DROP FUNCTION advance_planning_edit_revision();
COMMIT;
