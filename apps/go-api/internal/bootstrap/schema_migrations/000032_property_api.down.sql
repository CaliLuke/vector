BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER property_api_revision ON work_item_properties;
DROP TRIGGER property_link_api_revision ON work_item_type_properties;
DROP FUNCTION advance_property_api_revision();
ALTER TABLE work_item_properties DROP COLUMN api_metadata;
COMMIT;
