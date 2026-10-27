package workitems

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func (r *Repository) preparePropertyPQL(ctx context.Context, q Request) error {
	plan, err := issuequery.PreparePQL(q.Params["pql"], issuequery.PQLContext{ActorID: q.ActorID, Timezone: q.Timezone, Now: r.now()}, func(id string) (issuequery.PQLPropertyDefinition, bool, error) {
		return r.propertyDiscovery.PQLProperty(ctx, id)
	})
	if err != nil {
		return pqlRequestError(err)
	}
	r.pql = plan
	return nil
}

func (r *Repository) appendPropertyPQL(ctx context.Context, where []string, args []any) ([]string, []any, error) {
	if r.pql == nil {
		return where, args, nil
	}
	rows, err := r.propertyCandidateRows(ctx, where, args)
	if err != nil {
		return nil, args, err
	}
	values := make(map[string][]issuequery.PQLPropertyValue)
	for _, id := range r.pql.PropertyIDs() {
		values[id] = []issuequery.PQLPropertyValue{}
		for _, row := range rows {
			projected := row["property_values"].(map[string]workitemtypes.ProjectedValue)
			value := projected[id]
			if value.Status != "value" && value.Status != "empty" {
				continue
			}
			values[id] = append(values[id], issuequery.PQLPropertyValue{ItemID: stringValue(row["id"]), Value: value.Value})
		}
	}
	predicate, err := r.pql.SQL(values, &args)
	if err != nil {
		return nil, args, pqlRequestError(err)
	}
	return append(where, predicate), args, nil
}

func pqlRequestError(err error) error {
	var query *issuequery.FilterError
	if errors.As(err, &query) {
		return invalid(map[string]any{"pql": query.Error(), "code": "invalid_pql"})
	}
	return err
}

func pqlDiscoveryReadError(err error) error {
	var definition *workitemtypes.PropertyDefinitionError
	var scope *workitemtypes.Error
	if errors.As(err, &definition) || errors.As(err, &scope) && (scope.Status == 400 || scope.Status == 422) {
		return invalid(map[string]any{"pql": "Invalid PQL at byte 0: property discovery is unavailable.", "code": "invalid_pql"})
	}
	return pqlRequestError(discoveryReadError(err))
}
