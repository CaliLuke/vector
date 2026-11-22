BEGIN;
SET LOCAL search_path = public, pg_catalog;

DROP INDEX work_item_effects_oldest_pending;
DROP INDEX work_item_effects_pending;
UPDATE work_item_effects SET next_attempt_at=clock_timestamp()
 WHERE admitted_at IS NULL AND quarantined_at IS NOT NULL;
ALTER TABLE work_item_effects DROP COLUMN quarantined_at;
CREATE INDEX work_item_effects_pending ON work_item_effects(next_attempt_at,created_at,id)
 WHERE admitted_at IS NULL;

COMMIT;
