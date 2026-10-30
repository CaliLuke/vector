BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Removing enforcement does not recreate invalid historical timestamps.
ALTER TABLE issue_activities DROP CONSTRAINT issue_activities_epoch_finite;
COMMIT;
