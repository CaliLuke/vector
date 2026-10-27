package workspaceexports

import "github.com/CaliLuke/go-plane/internal/djangoapi"

// Custom exports are private snapshots. Rechecking the recorded rows prevents
// history from handing out a download after the actor loses any source access.
// Already downloaded files and issued presigned URLs retain snapshot semantics.
var customExportHistoryVisibleSQL = `(COALESCE(e.rich_filters->>'include_custom_properties','false') <> 'true' OR (
 e.initiated_by_id=$4::uuid AND w.deleted_at IS NULL
 AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=e.workspace_id
  AND wm.member_id=$4::uuid AND wm.role IN (15,20) AND wm.is_active AND wm.deleted_at IS NULL)
 AND NOT EXISTS(SELECT 1 FROM unnest(e.project) selected(id) WHERE NOT EXISTS(
  SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id
  WHERE p.id=selected.id AND p.workspace_id=e.workspace_id AND p.deleted_at IS NULL
   AND pm.member_id=$4::uuid AND pm.role IN (5,15,20) AND pm.is_active AND pm.deleted_at IS NULL))
 AND (e.status<>'completed' OR jsonb_typeof(e.rich_filters->'exported_item_ids')='array')
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements_text(
  CASE WHEN jsonb_typeof(e.rich_filters->'exported_item_ids')='array' THEN e.rich_filters->'exported_item_ids' ELSE '[]'::jsonb END
 ) exported(id) WHERE NOT EXISTS(SELECT 1 FROM issues i WHERE i.id::text=exported.id AND ` + djangoapi.IssueVisibleSQL("i", "$4::uuid", "e.workspace_id") + `))))`
