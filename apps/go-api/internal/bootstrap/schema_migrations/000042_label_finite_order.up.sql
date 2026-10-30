BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- Earlier writers accepted NaN and infinities, which cannot be serialized as
-- JSON numbers. Use the initial label order for those invalid values only.
-- Keep all other fields and finite orders; the existing revision trigger
-- records each repair for clients that use optimistic concurrency.
UPDATE labels SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);

ALTER TABLE labels ADD CONSTRAINT labels_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);
COMMIT;
