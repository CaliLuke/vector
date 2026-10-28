BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER draft_live_state ON draft_issues;
DROP TRIGGER issue_live_state ON issues;
DROP FUNCTION require_live_item_state();
DROP TRIGGER state_edit_revision ON states;
DROP FUNCTION advance_state_edit_revision();
DROP TABLE state_edit_revisions;
COMMIT;
