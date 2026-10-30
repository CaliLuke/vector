BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Retain repaired levels and revision history when removing enforcement.
ALTER TABLE issue_types DROP CONSTRAINT issue_types_level_valid;
COMMIT;
