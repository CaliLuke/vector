package modulereads

import (
	"context"
	"errors"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) Issues(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID, options IssueListOptions) (IssuePage, error) {
	if options.IncludeProperties || issuequery.UsesPropertyFilters(options.Filters) {
		return repository.propertyIssues(ctx, userID, slug, projectID, moduleID, options)
	}
	if !issuequery.UsesTypeFilters(options.Filters) {
		return repository.issueResults(ctx, userID, slug, projectID, moduleID, options, nil)
	}
	result, err := workitemtypes.WithTypeDiscoveryRead(ctx, repository.database, userID.String(), slug, projectID.String(), func(tx pgx.Tx, catalog workitemtypes.TypeIdentityCatalog) (IssuePage, error) {
		if !catalog.Enabled {
			return IssuePage{}, &FilterError{Code: "invalid_filter_reference", Message: "Work item types are disabled. Update this filter."}
		}
		reader := *repository
		reader.database = tx
		return reader.issueResults(ctx, userID, slug, projectID, moduleID, options, catalog.TypeIDs())
	})
	return result, moduleDiscoveryError(err)
}

func moduleDiscoveryError(err error) error {
	var access *workitemtypes.Error
	if errors.As(err, &access) {
		if access.Status == 403 {
			return ErrForbidden
		}
		return &FilterError{Code: "invalid_filter_reference", Message: access.Message}
	}
	return err
}
