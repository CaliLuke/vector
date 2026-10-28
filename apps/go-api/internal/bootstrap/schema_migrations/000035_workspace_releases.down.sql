BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Refuse rollback if workspace releases cannot fit the original ownership rule.
ALTER TABLE work_item_releases ALTER COLUMN project_id SET NOT NULL;
DROP TRIGGER release_revision ON work_item_releases;
DROP TRIGGER workspace_release_revision ON workspaces;
DROP TABLE release_work_items,release_item_labels,release_changelogs,release_events;
ALTER TABLE work_item_releases DROP CONSTRAINT release_tag_owner;
DROP TABLE release_tags,release_labels;
DROP FUNCTION advance_release_revision(),seed_release_revision();
DROP TABLE release_revisions;
DROP INDEX workspace_release_name_live,work_item_release_owner;
ALTER TABLE work_item_releases DROP COLUMN description_id,DROP COLUMN description_html,DROP COLUMN description_json,DROP COLUMN status,DROP COLUMN release_date,DROP COLUMN lead_id,DROP COLUMN tag_id,DROP COLUMN is_prerelease,DROP COLUMN external_source,DROP COLUMN external_id,DROP COLUMN updated_at,DROP COLUMN deleted_at,DROP COLUMN created_by_id,DROP COLUMN updated_by_id;
COMMIT;
