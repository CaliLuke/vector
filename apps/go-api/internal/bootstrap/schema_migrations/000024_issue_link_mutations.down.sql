BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER issue_link_edit_revision ON issue_links;
DROP FUNCTION advance_issue_link_edit_revision();
DROP TABLE issue_link_edit_revisions;
COMMIT;
