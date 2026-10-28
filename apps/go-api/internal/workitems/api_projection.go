package workitems

import (
	"context"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

// Bound relations follow the item's ownership, including corrupt or stale
// links from historical writers. The same snapshot covers ID and object forms.
func (r *Repository) boundIssueProjection(ctx context.Context, row map[string]any, q Request, fields, expand string) (map[string]any, error) {
	owned, err := r.row(ctx, `SELECT jsonb_build_object(
 'state_id',(SELECT s.id FROM states s WHERE s.id=i.state_id AND s.project_id=i.project_id AND s.workspace_id=i.workspace_id AND s.deleted_at IS NULL),
 'type_id',(SELECT t.id FROM issue_types t WHERE t.id=i.type_id AND t.workspace_id=i.workspace_id AND t.deleted_at IS NULL),
 'estimate_point_id',(SELECT ep.id FROM estimate_points ep JOIN estimates e ON e.id=ep.estimate_id AND e.project_id=i.project_id AND e.workspace_id=i.workspace_id AND e.deleted_at IS NULL WHERE ep.id=i.estimate_point_id AND ep.project_id=i.project_id AND ep.workspace_id=i.workspace_id AND ep.deleted_at IS NULL),
 'assignees',COALESCE((SELECT jsonb_agg(u.id ORDER BY ia.created_at DESC) FROM issue_assignees ia JOIN users u ON u.id=ia.assignee_id AND u.is_active
 WHERE ia.issue_id=i.id AND ia.project_id=i.project_id AND ia.workspace_id=i.workspace_id AND ia.deleted_at IS NULL
 AND EXISTS(SELECT 1 FROM project_members pm JOIN workspace_members wm ON wm.workspace_id=pm.workspace_id AND wm.member_id=pm.member_id
 WHERE pm.project_id=i.project_id AND pm.workspace_id=i.workspace_id AND pm.member_id=u.id AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN (5,15,20) AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN (5,15,20))),'[]'::jsonb))
 FROM issues i WHERE i.id=$1 AND i.workspace_id=$2`, row["id"], q.ExpectedWorkspaceID.String())
	if err != nil {
		return nil, err
	}
	projected := cloneCollaborationRow(row)
	for key, value := range owned {
		projected[key] = value
	}
	projected["type"] = projected["type_id"]
	parts := []string{}
	expandAssignees := false
	for _, key := range strings.Split(expand, ",") {
		key = strings.TrimSpace(key)
		if key == "assignees" {
			expandAssignees = true
		} else {
			parts = append(parts, key)
		}
	}
	projected, err = djangoapi.MaskIssueParent(ctx, r.db, projected, q.ActorID)
	if err != nil {
		return nil, err
	}
	result, err := r.serializeAuthorizedIssue(ctx, projected, fields, strings.Join(parts, ","))
	if err != nil {
		return nil, err
	}
	if _, selected := result["assignees"]; expandAssignees && selected {
		values := []any{}
		if ids, ok := projected["assignees"].([]any); ok {
			for _, id := range ids {
				value, err := djangoapi.JSON(ctx, r.db, `SELECT jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'display_name',u.display_name,'email',u.email,'avatar',u.avatar)||`+djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url")+` FROM users u WHERE u.id=$1`, id)
				if err != nil {
					return nil, err
				}
				values = append(values, value)
			}
		}
		result["assignees"] = values
	}
	return result, nil
}
