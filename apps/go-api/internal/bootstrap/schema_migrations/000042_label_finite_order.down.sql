BEGIN;
SET LOCAL search_path = public, pg_catalog;
ALTER TABLE labels DROP CONSTRAINT labels_sort_order_finite;
-- Repairs remain finite: reverting the constraint does not recreate invalid data.
COMMIT;
