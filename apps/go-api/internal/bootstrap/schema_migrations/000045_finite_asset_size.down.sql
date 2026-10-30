BEGIN;
SET LOCAL search_path = public, pg_catalog;
ALTER TABLE file_assets DROP CONSTRAINT file_assets_size_finite;
COMMIT;
