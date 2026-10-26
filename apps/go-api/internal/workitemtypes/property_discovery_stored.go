package workitemtypes

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

// DiscoveryItemReference identifies a published item already admitted by the
// caller's route-specific visibility query in this same transaction.
type DiscoveryItemReference struct{ ID, ProjectID string }

// DiscoveryVisibility preserves the admitting route's permission contract.
// Elevated routes allow active workspace admins through a guest project role.
type DiscoveryVisibility uint8

const (
	DiscoveryItemVisibility DiscoveryVisibility = iota
	DiscoveryElevatedProjectVisibility
)

// ProjectStoredItems loads canonical inputs in one batch without extending a
// legacy serializer with raw property storage. It rechecks item visibility and
// rejects missing, draft or mismatched references; it is not a discovery query.
// Input order and duplicate cardinality are retained, with no partial result.
func (snapshot *PropertyDiscoverySnapshot) ProjectStoredItems(ctx context.Context, references []DiscoveryItemReference, visibility DiscoveryVisibility) ([]ItemProjection, error) {
	if visibility != DiscoveryItemVisibility && visibility != DiscoveryElevatedProjectVisibility {
		return nil, reject(403, "Invalid property projection scope.")
	}
	ids := make([]string, 0, len(references))
	seen := map[string]bool{}
	projects := make([]string, 0, len(snapshot.projects))
	for _, reference := range references {
		if !snapshot.projects[reference.ProjectID] {
			return nil, reject(403, "Work item project access changed.")
		}
		if !seen[reference.ID] {
			ids = append(ids, reference.ID)
			seen[reference.ID] = true
		}
	}
	if len(ids) == 0 {
		return []ItemProjection{}, nil
	}
	for project := range snapshot.projects {
		projects = append(projects, project)
	}
	rows, err := snapshot.tx.Query(ctx, `SELECT i.id::text,i.project_id::text,COALESCE(i.type_id::text,''),COALESCE(i.estimate_point_id::text,''),
 i.name,i.priority,COALESCE(i.start_date::text,''),COALESCE(i.target_date::text,''),i.custom_properties
 FROM issues i WHERE i.id=ANY($1::uuid[]) AND i.workspace_id=$2::uuid AND i.project_id=ANY($3::uuid[])
 AND i.deleted_at IS NULL AND NOT i.is_draft AND (`+djangoapi.IssueVisibleSQL("i", "$4::uuid", "$2::uuid")+`
 OR ($5::boolean AND EXISTS (
 SELECT 1 FROM workspace_members wm
 JOIN project_members pm ON pm.workspace_id=wm.workspace_id AND pm.member_id=wm.member_id
 JOIN projects p ON p.id=pm.project_id AND p.workspace_id=pm.workspace_id
 JOIN workspaces w ON w.id=p.workspace_id
 WHERE wm.workspace_id=$2::uuid AND wm.member_id=$4::uuid AND wm.role=20
 AND wm.is_active AND wm.deleted_at IS NULL AND pm.project_id=i.project_id
 AND pm.is_active AND pm.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL
 )))`, ids, snapshot.workspace, projects, snapshot.actor, visibility == DiscoveryElevatedProjectVisibility)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]DiscoveryItem, len(ids))
	for rows.Next() {
		var item DiscoveryItem
		if err = rows.Scan(&item.ID, &item.ProjectID, &item.TypeID, &item.EstimatePointID, &item.Standard.Name, &item.Standard.Priority, &item.Standard.StartDate, &item.Standard.DueDate, &item.Saved); err != nil {
			rows.Close()
			return nil, err
		}
		byID[item.ID] = item
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	items := make([]DiscoveryItem, 0, len(references))
	for _, reference := range references {
		item, exists := byID[reference.ID]
		if !exists || item.ProjectID != reference.ProjectID {
			return nil, reject(403, "Work item access changed.")
		}
		items = append(items, item)
	}
	return snapshot.ProjectItems(ctx, items)
}

// ProjectStoredRows enriches an already-admitted page without changing its
// ordering, grouping metadata, or existing serialized values. All projections
// succeed before any row is changed.
func (snapshot *PropertyDiscoverySnapshot) ProjectStoredRows(ctx context.Context, rows []map[string]any, visibility DiscoveryVisibility) error {
	references := make([]DiscoveryItemReference, len(rows))
	for index, row := range rows {
		id, _ := row["id"].(string)
		project, _ := row["project_id"].(string)
		references[index] = DiscoveryItemReference{ID: id, ProjectID: project}
	}
	projected, err := snapshot.ProjectStoredItems(ctx, references, visibility)
	if err != nil {
		return err
	}
	for index, value := range projected {
		rows[index]["property_values"], rows[index]["properties_evaluated_at"] = value.Values, value.EvaluatedAt
		rows[index]["properties_revision"] = snapshot.Types.Revision
	}
	return nil
}
