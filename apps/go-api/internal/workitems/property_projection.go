package workitems

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

// Rows have already passed the route's complete item visibility predicate. This
// opt-in projection never runs on public/API-token serializers. Future custom
// predicates use this same snapshot plan before count/pagination, not this page.
func (r *Repository) projectDiscoveryRows(ctx context.Context, rows []map[string]any) error {
	if r.propertyDiscovery == nil {
		return nil
	}
	items := make([]workitemtypes.DiscoveryItem, 0, len(rows))
	for _, row := range rows {
		saved, _ := row["custom_properties"].(map[string]any)
		items = append(items, workitemtypes.DiscoveryItem{
			ID: stringValue(row["id"]), ProjectID: stringValue(row["project_id"]), TypeID: stringValue(row["type_id"]),
			EstimatePointID: stringValue(row["estimate_point_id"]), Saved: saved,
			Standard: workitemtypes.ItemFormulaInputs{Name: stringValue(row["name"]), Priority: stringValue(row["priority"]), StartDate: stringValue(row["start_date"]), DueDate: stringValue(row["target_date"])},
		})
	}
	projected, err := r.propertyDiscovery.ProjectItems(ctx, items)
	if err != nil {
		if r.pql != nil {
			return pqlDiscoveryReadError(err)
		}
		return discoveryReadError(err)
	}
	for index, value := range projected {
		rows[index]["property_values"], rows[index]["properties_evaluated_at"] = value.Values, value.EvaluatedAt
		rows[index]["properties_revision"] = r.propertyDiscovery.Types.Revision
	}
	return nil
}
