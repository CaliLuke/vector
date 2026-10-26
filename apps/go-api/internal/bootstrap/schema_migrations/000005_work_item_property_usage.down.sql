BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER draft_property_usage ON draft_issues;
DROP TRIGGER issue_property_usage ON issues;
DROP FUNCTION record_work_item_property_usage();
ALTER TABLE work_item_properties DROP COLUMN used;
COMMIT;
