BEGIN;
SET LOCAL search_path = public, pg_catalog;

ALTER TABLE work_item_property_versions
 ADD COLUMN property_title text NOT NULL DEFAULT 'Rich-text field',
 ADD COLUMN source text NOT NULL DEFAULT 'preservation'
 CHECK (source IN ('change','baseline','preservation','restore'));
UPDATE work_item_property_versions v SET property_title=p.definition->>'title'
 FROM work_item_properties p WHERE p.id=v.property_id AND p.workspace_id=v.workspace_id
 AND COALESCE(p.definition->>'title','')<>'';
ALTER TABLE work_item_property_versions ALTER COLUMN source SET DEFAULT 'change';
CREATE INDEX work_item_property_versions_order ON work_item_property_versions(issue_id,property_id,created_at DESC,id DESC);

COMMIT;
