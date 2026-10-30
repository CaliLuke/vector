BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- Repair only non-finite ordering values, using the existing allocation base
-- of 65535 because NaN and infinities do not represent a valid position.
-- Keep finite orders and unrelated metadata, including soft-deleted rows.
-- Ordinary UPDATEs retain the existing revision triggers for repaired records.

UPDATE cycles SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE cycles ADD CONSTRAINT cycles_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE draft_issues SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE draft_issues ADD CONSTRAINT draft_issues_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE issue_versions SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE issue_versions ADD CONSTRAINT issue_versions_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE issue_views SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE issue_views ADD CONSTRAINT issue_views_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE issues SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE issues ADD CONSTRAINT issues_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE modules SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE modules ADD CONSTRAINT modules_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE pages SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE pages ADD CONSTRAINT pages_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE project_members SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE project_members ADD CONSTRAINT project_members_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE project_user_properties SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE project_user_properties ADD CONSTRAINT project_user_properties_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE states SET sequence = 65535
WHERE NOT (sequence > '-Infinity'::double precision
           AND sequence < 'Infinity'::double precision);
ALTER TABLE states ADD CONSTRAINT states_sequence_finite
CHECK (sequence > '-Infinity'::double precision
       AND sequence < 'Infinity'::double precision);

UPDATE stickies SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE stickies ADD CONSTRAINT stickies_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE user_favorites SET sequence = 65535
WHERE NOT (sequence > '-Infinity'::double precision
           AND sequence < 'Infinity'::double precision);
ALTER TABLE user_favorites ADD CONSTRAINT user_favorites_sequence_finite
CHECK (sequence > '-Infinity'::double precision
       AND sequence < 'Infinity'::double precision);

UPDATE workspace_home_preferences SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE workspace_home_preferences ADD CONSTRAINT workspace_home_preferences_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE workspace_user_preferences SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE workspace_user_preferences ADD CONSTRAINT workspace_user_preferences_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE work_item_relation_definitions SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE work_item_relation_definitions ADD CONSTRAINT work_item_relation_definitions_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE release_labels SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE release_labels ADD CONSTRAINT release_labels_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE page_collections SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE page_collections ADD CONSTRAINT page_collections_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE page_collection_pages SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE page_collection_pages ADD CONSTRAINT page_collection_pages_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE customer_properties SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE customer_properties ADD CONSTRAINT customer_properties_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);

UPDATE customer_property_options SET sort_order = 65535
WHERE NOT (sort_order > '-Infinity'::double precision
           AND sort_order < 'Infinity'::double precision);
ALTER TABLE customer_property_options ADD CONSTRAINT customer_property_options_sort_order_finite
CHECK (sort_order > '-Infinity'::double precision
       AND sort_order < 'Infinity'::double precision);
COMMIT;
