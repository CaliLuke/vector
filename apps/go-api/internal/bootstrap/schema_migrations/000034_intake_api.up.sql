BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- The intake token covers both the submission metadata and its underlying item.
CREATE TRIGGER intake_issue_edit_revision
AFTER INSERT OR UPDATE OR DELETE ON intake_issues
FOR EACH ROW EXECUTE FUNCTION advance_issue_assignment_revision();
COMMIT;
