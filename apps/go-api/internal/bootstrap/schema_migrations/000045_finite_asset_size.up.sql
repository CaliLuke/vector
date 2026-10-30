BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- Zero is the existing fallback when a legacy upload supplies no usable size.
-- Preserve the object's identity, attributes and storage metadata.
UPDATE file_assets SET size = 0
WHERE NOT (size > '-Infinity'::double precision AND size < 'Infinity'::double precision);
ALTER TABLE file_assets ADD CONSTRAINT file_assets_size_finite
CHECK (size > '-Infinity'::double precision AND size < 'Infinity'::double precision);
COMMIT;
