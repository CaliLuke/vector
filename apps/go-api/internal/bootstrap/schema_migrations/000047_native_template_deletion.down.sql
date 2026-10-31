BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Keep recovered tombstones: rolling back must not restore templates whose
-- parents were deleted or overwrite their retained deletion timestamps.
COMMIT;
