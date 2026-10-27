package workitems

import (
	"context"
	"strings"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) preparePropertyFilters(ctx context.Context, q Request, tx pgx.Tx) error {
	predicates := map[int]*workitemtypes.ScopedPropertyPredicate{}
	plan, err := issuequery.PreparePropertyFilter(q.Params["filters"], r.propertyDiscovery.Types.TypeIDs(), func(field string, value any) (bool, error) {
		predicate, handled, validationErr := r.propertyDiscovery.FilterPredicate(ctx, tx, field, value)
		if validationErr != nil {
			return handled, discoveryReadError(validationErr)
		}
		if handled {
			predicates[len(predicates)+1] = predicate
		}
		return handled, nil
	})
	if err != nil {
		return err
	}
	r.propertyFilter, r.propertyPredicates = plan, predicates
	return nil
}

// The candidate query contains every route visibility/flat restriction but none
// of the structured expression. Removing only custom terms would change OR/NOT.
// This is deliberately before counts and windows; it never reads the first page
// and infers the remaining matches from it.
func (r *Repository) bindPropertyFilters(ctx context.Context, where []string, args []any) (*issuequery.BoundPropertyFilter, error) {
	rows, err := r.propertyCandidateRows(ctx, where, args)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for index, row := range rows {
		ids[index] = stringValue(row["id"])
	}
	return r.propertyFilter.BindEvaluatedItems(ids, func(leafID, index int) (bool, bool) {
		values := rows[index]["property_values"].(map[string]workitemtypes.ProjectedValue)
		return r.propertyPredicates[leafID].Match(values)
	})
}

func (r *Repository) propertyCandidateRows(ctx context.Context, where []string, args []any) ([]map[string]any, error) {
	rows, err := r.rows(ctx, `SELECT jsonb_build_object('id',issue.id,'project_id',issue.project_id,'type_id',issue.type_id,'name',issue.name,'priority',issue.priority,'start_date',issue.start_date,'target_date',issue.target_date,'estimate_point_id',issue.estimate_point_id,'custom_properties',issue.custom_properties) FROM issues issue JOIN projects project ON project.id=issue.project_id JOIN workspaces workspace ON workspace.id=issue.workspace_id WHERE `+strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, err
	}
	if err = r.projectDiscoveryRows(ctx, rows); err != nil {
		return nil, err
	}
	return rows, nil
}
