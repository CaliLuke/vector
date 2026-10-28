BEGIN;
SET LOCAL search_path = public, pg_catalog;
-- Fail atomically if newer data cannot fit the older one-relation-per-pair model.
CREATE UNIQUE INDEX issue_relation_unique_issue_related_issue_when_deleted_at_null ON issue_relations(issue_id,related_issue_id) WHERE deleted_at IS NULL;
DROP INDEX issue_relation_unique_kind_live;
DROP TRIGGER dependency_relation_revision ON issue_relations;
DROP TRIGGER workspace_relation_revision ON workspaces;
DROP TABLE work_item_custom_relations;
DROP TABLE work_item_relation_definitions;
DROP FUNCTION advance_work_item_relation_revision();
DROP FUNCTION seed_work_item_relation_revision();
DROP TABLE work_item_relation_revisions;
DROP INDEX issues_relation_owner_key;
COMMIT;
