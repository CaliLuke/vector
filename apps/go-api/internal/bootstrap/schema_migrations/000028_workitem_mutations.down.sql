BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER issue_label_edit_revision ON issue_labels;
DROP TRIGGER issue_assignee_edit_revision ON issue_assignees;
DROP FUNCTION advance_issue_assignment_revision();
DROP TRIGGER issue_edit_revision ON issues;
DROP FUNCTION advance_issue_edit_revision();
DROP TABLE issue_edit_revisions;
COMMIT;
