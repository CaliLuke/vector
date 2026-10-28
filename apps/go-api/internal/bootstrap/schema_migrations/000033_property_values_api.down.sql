BEGIN;
SET LOCAL search_path = public, pg_catalog;
DROP TRIGGER item_property_value_identities ON issues;
DROP FUNCTION sync_item_property_value_identities();
DROP FUNCTION sync_property_value_identities(public.issues);
DROP TABLE work_item_property_value_identities;
DROP FUNCTION advance_property_value_identity_revision();
DROP INDEX work_item_properties_api_owner;
COMMIT;
