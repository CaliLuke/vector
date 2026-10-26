package cyclereads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) Issues(ctx context.Context, userID uuid.UUID, slug string, projectID, cycleID uuid.UUID, options IssueListOptions) (json.RawMessage, error) {
	if options.IncludeProperties || issuequery.UsesPropertyFilters(options.ComplexFilter) {
		return repository.propertyIssues(ctx, userID, slug, projectID, cycleID, options)
	}
	if !issuequery.UsesTypeFilters(options.ComplexFilter) {
		return repository.issueResults(ctx, userID, slug, projectID, cycleID, options)
	}
	result, err := workitemtypes.WithTypeDiscoveryRead(ctx, repository.database, userID.String(), slug, projectID.String(), func(tx pgx.Tx, catalog workitemtypes.TypeIdentityCatalog) (json.RawMessage, error) {
		if !catalog.Enabled {
			return nil, newFilterError("Work item types are disabled. Update this filter.", "invalid_filter_reference")
		}
		reader := *repository
		reader.database = tx
		options.typeIDs = catalog.TypeIDs()
		return reader.issueResults(ctx, userID, slug, projectID, cycleID, options)
	})
	return result, cycleDiscoveryError(err)
}

func cycleDiscoveryError(err error) error {
	var access *workitemtypes.Error
	if errors.As(err, &access) {
		if access.Status == 403 {
			return ErrForbidden
		}
		return newFilterError(access.Message, "invalid_filter_reference")
	}
	return err
}

type issueFilterCompiler struct {
	typeIDs    map[string]bool
	properties *cyclePropertyBindings
}

func buildIssueComplexFilter(raw string, args *[]any) (string, error) {
	return (issueFilterCompiler{}).build(raw, args)
}

func (compiler issueFilterCompiler) typeField(field string, value any, args *[]any) (string, bool, error) {
	clause, handled, err := issuequery.BuildTypeFilterField(field, value, args, compiler.typeIDs)
	var invalid *issuequery.FilterError
	if errors.As(err, &invalid) {
		err = &FilterError{Code: invalid.Code, Message: invalid.Message, Errors: invalid.Errors}
	}
	return clause, handled, err
}

func issueLogicalValue(object map[string]any, operator string) (any, bool) {
	for key, value := range object {
		if strings.EqualFold(key, operator) {
			return value, true
		}
	}
	return nil, false
}
