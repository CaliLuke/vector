BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Do not discard durable work when rolling back an application version.
ALTER TABLE work_item_effects DROP CONSTRAINT work_item_effects_kind_check;
ALTER TABLE work_item_effects ADD CONSTRAINT work_item_effects_kind_check
 CHECK (kind IN ('activity','description_version','soft_delete'));
COMMIT;
