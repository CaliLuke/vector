package workitems

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) listIssues(ctx context.Context, q Request) (Result, error) {
	if q.API && usesDiscoveryGrouping(q) {
		return Result{}, failure(400, "error", "Work item type grouping requires an authenticated application read.")
	}
	propertyPQL := false
	if q.API {
		var err error
		propertyPQL, err = issuequery.PQLUsesProperties(q.Params["pql"])
		if err != nil {
			return Result{}, pqlRequestError(err)
		}
	}
	if propertyPQL || !q.API && (q.Params["include_properties"] == "true" || issuequery.UsesPropertyFilters(q.Params["filters"]) || len(propertyGroupingKeys(q)) > 0) {
		readSnapshot := workitemtypes.WithPropertyDiscoveryRead[Result]
		if !propertyPQL && !issuequery.UsesPropertyFilters(q.Params["filters"]) && !issuequery.UsesTypeFilters(q.Params["filters"]) && !usesDiscoveryGrouping(q) {
			readSnapshot = workitemtypes.WithPropertyProjectionRead[Result]
		}
		result, err := readSnapshot(ctx, r.db, q.ActorID, q.Slug, q.ProjectID, func(tx pgx.Tx, snapshot *workitemtypes.PropertyDiscoverySnapshot) (Result, error) {
			reader := *r
			reader.db, reader.propertyDiscovery, reader.typeDiscovery = tx, snapshot, &snapshot.Types
			if propertyPQL {
				if err := reader.preparePropertyPQL(ctx, q); err != nil {
					return Result{}, err
				}
			}
			if err := reader.preparePropertyGrouping(ctx, q); err != nil {
				return Result{}, err
			}
			if !q.API && issuequery.UsesPropertyFilters(q.Params["filters"]) {
				if err := reader.preparePropertyFilters(ctx, q, tx); err != nil {
					return Result{}, err
				}
			}
			if err := reader.prepareParentGrouping(ctx, q); err != nil {
				return Result{}, err
			}
			return reader.listIssueResults(ctx, q, snapshot.Types.TypeIDs())
		})
		if propertyPQL {
			return result, pqlDiscoveryReadError(err)
		}
		return result, discoveryReadError(err)
	}

	if q.API || !issuequery.UsesTypeFilters(q.Params["filters"]) && !usesDiscoveryGrouping(q) {
		return r.listIssueResults(ctx, q, nil)
	}
	result, err := workitemtypes.WithTypeDiscoveryRead(ctx, r.db, q.ActorID, q.Slug, q.ProjectID, func(tx pgx.Tx, catalog workitemtypes.TypeIdentityCatalog) (Result, error) {
		if !catalog.Enabled {
			return Result{}, failure(400, "error", "Work item types are disabled. Update this filter.")
		}
		reader := *r
		reader.db, reader.typeDiscovery = tx, &catalog
		if err := reader.prepareParentGrouping(ctx, q); err != nil {
			return Result{}, err
		}
		return reader.listIssueResults(ctx, q, catalog.TypeIDs())
	})
	return result, discoveryReadError(err)
}

func discoveryReadError(err error) error {
	var access *workitemtypes.Error
	if errors.As(err, &access) {
		status := access.Status
		if status == 422 {
			status = 400
		}
		return failure(status, "error", access.Message)
	}
	return err
}
