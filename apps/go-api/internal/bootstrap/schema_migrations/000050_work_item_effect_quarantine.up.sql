BEGIN;
SET LOCAL search_path = public, pg_catalog;

ALTER TABLE work_item_effects ADD COLUMN quarantined_at timestamptz;
DROP INDEX work_item_effects_pending;
CREATE INDEX work_item_effects_pending ON work_item_effects(next_attempt_at,created_at,id)
 WHERE admitted_at IS NULL AND quarantined_at IS NULL;
CREATE INDEX work_item_effects_oldest_pending ON work_item_effects(created_at)
 WHERE admitted_at IS NULL AND quarantined_at IS NULL;

COMMIT;
