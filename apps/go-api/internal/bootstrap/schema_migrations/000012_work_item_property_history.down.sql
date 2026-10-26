BEGIN;
SET LOCAL search_path = public, pg_catalog;

DROP INDEX work_item_property_versions_order;
ALTER TABLE work_item_property_versions DROP COLUMN source, DROP COLUMN property_title;

COMMIT;
