package workitemtypes

import "context"

// DiscoveryItem must come from the callback transaction's fully authorized item
// query, including guest visibility and route restrictions. Project membership
// alone does not authorize an item. EstimatePointID is only a lookup hint;
// estimates are resolved from the actual issue relation in the same snapshot.
type DiscoveryItem struct {
	ID              string
	ProjectID       string
	TypeID          string
	EstimatePointID string
	Standard        ItemFormulaInputs
	Saved           map[string]any
}

// ProjectItems shares estimate hydration and effective formula/property values
// across list adapters. Use only inside WithPropertyDiscoveryRead's callback.
// Input order and duplicate rows are retained; errors return no partial result.
func (snapshot *PropertyDiscoverySnapshot) ProjectItems(ctx context.Context, items []DiscoveryItem) ([]ItemProjection, error) {
	for _, item := range items {
		if err := snapshot.validateItemScope(item.ProjectID, item.TypeID); err != nil {
			return nil, err
		}
	}
	estimates := map[string]discoveryEstimate{}
	if snapshot.Types.Enabled {
		var err error
		estimates, err = snapshot.itemEstimates(ctx, items)
		if err != nil {
			return nil, err
		}
	}
	out := make([]ItemProjection, 0, len(items))
	for _, item := range items {
		standard := item.Standard
		standard.Estimate = nil
		if estimate, exists := estimates[item.ID]; exists && estimate.projectID == item.ProjectID {
			standard.Estimate = &estimate.value
		}
		projected, projectionErr := snapshot.Project(item.ProjectID, item.TypeID, standard, item.Saved)
		if projectionErr != nil {
			return nil, projectionErr
		}
		out = append(out, projected)
	}
	return out, nil
}

type discoveryEstimate struct{ projectID, value string }

func (snapshot *PropertyDiscoverySnapshot) itemEstimates(ctx context.Context, items []DiscoveryItem) (map[string]discoveryEstimate, error) {
	out := map[string]discoveryEstimate{}
	ids := []string{}
	seen := map[string]bool{}
	for _, item := range items {
		if item.EstimatePointID != "" && !seen[item.ID] {
			ids = append(ids, item.ID)
			seen[item.ID] = true
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	projects := make([]string, 0, len(snapshot.projects))
	for project := range snapshot.projects {
		projects = append(projects, project)
	}
	rows, err := snapshot.tx.Query(ctx, `SELECT i.id::text,i.project_id::text,ep.value FROM issues i JOIN estimate_points ep ON ep.id=i.estimate_point_id AND ep.workspace_id=i.workspace_id AND ep.project_id=i.project_id AND ep.deleted_at IS NULL JOIN estimates e ON e.id=ep.estimate_id AND e.workspace_id=ep.workspace_id AND e.project_id=ep.project_id AND e.deleted_at IS NULL WHERE i.id=ANY($1::uuid[]) AND i.workspace_id=$2 AND i.project_id=ANY($3::uuid[]) AND i.deleted_at IS NULL`, ids, snapshot.workspace, projects)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var value discoveryEstimate
		if err = rows.Scan(&id, &value.projectID, &value.value); err != nil {
			return nil, err
		}
		out[id] = value
	}
	return out, rows.Err()
}
