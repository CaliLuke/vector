BEGIN;
SET LOCAL search_path = public, pg_catalog;

-- The hierarchy editor accepts integer levels from 0 through 20. Take the
-- existing workspace locks before reading repair candidates, so a concurrent
-- reviewed hierarchy change cannot be overwritten with an older copy.
CREATE TEMP TABLE vector_invalid_level_workspaces ON COMMIT DROP AS
SELECT DISTINCT workspace_id FROM issue_types
WHERE NOT (level BETWEEN 0 AND 20 AND level = floor(level));
DO $$
DECLARE affected uuid;
BEGIN
 FOR affected IN SELECT workspace_id FROM vector_invalid_level_workspaces ORDER BY workspace_id LOOP
  PERFORM work_item_schema_lock(affected);
 END LOOP;
END $$;

-- Project copies are updated with every reviewed hierarchy change. Recover only
-- when they agree. With no copy, use the documented unassigned level of zero.
CREATE TEMP TABLE vector_type_level_repairs ON COMMIT DROP AS
SELECT t.id,t.workspace_id,
 CASE WHEN count(p.id)=0 THEN 0
      WHEN min(p.level)=max(p.level) AND min(p.level) BETWEEN 0 AND 20 THEN min(p.level)
      ELSE NULL END AS repaired_level
FROM issue_types t
JOIN vector_invalid_level_workspaces affected ON affected.workspace_id=t.workspace_id
LEFT JOIN project_issue_types p ON p.issue_type_id=t.id AND p.workspace_id=t.workspace_id
WHERE NOT (t.level BETWEEN 0 AND 20 AND t.level = floor(t.level))
GROUP BY t.id,t.workspace_id;
DO $$
BEGIN
 IF EXISTS(SELECT 1 FROM vector_type_level_repairs WHERE repaired_level IS NULL) THEN
  RAISE EXCEPTION 'Cannot repair work item type levels: project copies disagree or are outside the supported range'
   USING ERRCODE='23514',HINT='Restore reviewed type levels and follow the migration 46 journal recovery procedure in docs/development.md. No parent relationships have been changed.';
 END IF;
END $$;

SET CONSTRAINTS type_hierarchy_guard DEFERRED;
UPDATE issue_types t SET level=r.repaired_level
FROM vector_type_level_repairs r WHERE t.id=r.id AND t.workspace_id=r.workspace_id;
-- Invalidate native reviews as well as the MCP revisions maintained by triggers.
UPDATE work_item_type_settings SET revision=revision+1
WHERE workspace_id IN(SELECT workspace_id FROM vector_type_level_repairs);
DO $$
DECLARE affected uuid;
BEGIN
 FOR affected IN SELECT DISTINCT workspace_id FROM vector_type_level_repairs ORDER BY workspace_id LOOP
  BEGIN
   PERFORM validate_workspace_work_item_hierarchy(affected);
  EXCEPTION WHEN check_violation THEN
   RAISE EXCEPTION 'Cannot repair work item type levels without changing parent relationships'
    USING ERRCODE='23514',HINT='Restore reviewed type levels and follow the migration 46 journal recovery procedure in docs/development.md. This migration does not unlink work items or drafts.';
  END;
 END LOOP;
END $$;
SET CONSTRAINTS type_hierarchy_guard IMMEDIATE;
ALTER TABLE issue_types ADD CONSTRAINT issue_types_level_valid
CHECK (level BETWEEN 0 AND 20 AND level = floor(level));
COMMIT;
