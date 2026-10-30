BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- An invalid historical timestamp is unknown. Preserve the activity and all
-- its other metadata instead of inventing an occurrence time.
UPDATE issue_activities SET epoch = NULL
WHERE NOT (epoch > '-Infinity'::double precision AND epoch < 'Infinity'::double precision);
ALTER TABLE issue_activities ADD CONSTRAINT issue_activities_epoch_finite
CHECK (epoch > '-Infinity'::double precision AND epoch < 'Infinity'::double precision);
COMMIT;
