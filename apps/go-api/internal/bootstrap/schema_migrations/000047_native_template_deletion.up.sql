BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- Older deletion workers did not register native templates. Recover their
-- inherited tombstones without changing definitions, audit or prior deletions.
UPDATE resource_templates AS template
SET deleted_at = LEAST(workspace.deleted_at, project.deleted_at),
    updated_at = clock_timestamp()
FROM projects AS project
JOIN workspaces AS workspace ON workspace.id = project.workspace_id
WHERE template.workspace_id = workspace.id
  AND template.project_id = project.id
  AND template.deleted_at IS NULL
  AND (workspace.deleted_at IS NOT NULL OR project.deleted_at IS NOT NULL);

-- Workspace templates have no project join and remain independent of project
-- references inside their opaque JSON definitions.
UPDATE resource_templates AS template
SET deleted_at = workspace.deleted_at,
    updated_at = clock_timestamp()
FROM workspaces AS workspace
WHERE template.workspace_id = workspace.id
  AND template.project_id IS NULL
  AND template.deleted_at IS NULL
  AND workspace.deleted_at IS NOT NULL;

COMMIT;
