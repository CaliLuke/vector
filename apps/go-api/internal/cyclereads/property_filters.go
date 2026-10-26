package cyclereads

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

type cyclePropertyEntry struct {
	predicate *workitemtypes.ScopedPropertyPredicate
	matches   map[string]bool
}

// Bindings belong to one authorized snapshot. Validation may add entries only
// during preparation; final query compilation cannot introduce unevaluated leaves.
type cyclePropertyBindings struct {
	entries  map[string]*cyclePropertyEntry
	validate func(string, any) (*workitemtypes.ScopedPropertyPredicate, bool, error)
}

func (bindings *cyclePropertyBindings) field(field string, value any, args *[]any) (string, bool, error) {
	if bindings == nil || !strings.HasPrefix(field, "customproperty_") {
		return "", false, nil
	}
	operand, err := json.Marshal(value)
	if err != nil {
		return "", true, err
	}
	key := field + "\x00" + string(operand)
	entry := bindings.entries[key]
	if entry == nil && bindings.validate != nil {
		predicate, handled, validationErr := bindings.validate(field, value)
		if validationErr != nil {
			return "", true, cycleDiscoveryError(validationErr)
		}
		if handled {
			entry = &cyclePropertyEntry{predicate: predicate, matches: map[string]bool{}}
			bindings.entries[key] = entry
		}
	}
	if entry == nil {
		return "", true, newFilterError("Property filter was not prepared.", "invalid_filter_reference")
	}
	*args = append(*args, entry.matches)
	// Cycle negation is plain SQL NOT. Keep unknown values NULL here, including
	// under mixed legacy/custom groups, instead of coercing them to false.
	return fmt.Sprintf("($%d::jsonb ->> issue.id::text)::boolean", len(*args)), true, nil
}

func (repository *Repository) propertyIssues(ctx context.Context, actor uuid.UUID, slug string, project, cycle uuid.UUID, options IssueListOptions) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, actor, slug, project, false); err != nil {
		return nil, err
	}
	readSnapshot := workitemtypes.WithPropertyDiscoveryRead[json.RawMessage]
	if !issuequery.UsesPropertyFilters(options.ComplexFilter) && !issuequery.UsesTypeFilters(options.ComplexFilter) {
		readSnapshot = workitemtypes.WithPropertyProjectionRead[json.RawMessage]
	}
	result, err := readSnapshot(ctx, repository.database, actor.String(), slug, project.String(), func(tx pgx.Tx, snapshot *workitemtypes.PropertyDiscoverySnapshot) (json.RawMessage, error) {
		reader := *repository
		reader.database = tx
		if accessErr := reader.AuthorizeProject(ctx, actor, slug, project, false); accessErr != nil {
			return nil, accessErr
		}
		options.typeIDs, options.snapshot = snapshot.Types.TypeIDs(), snapshot
		if issuequery.UsesPropertyFilters(options.ComplexFilter) {
			bindings := &cyclePropertyBindings{entries: map[string]*cyclePropertyEntry{}, validate: func(field string, value any) (*workitemtypes.ScopedPropertyPredicate, bool, error) {
				return snapshot.FilterPredicate(ctx, tx, field, value)
			}}
			options.properties = bindings
			scratch := []any{}
			if _, prepareErr := (issueFilterCompiler{typeIDs: options.typeIDs, properties: bindings}).build(options.ComplexFilter, &scratch); prepareErr != nil {
				return nil, prepareErr
			}
			bindings.validate = nil
			if bindErr := reader.bindPropertyFilters(ctx, actor, slug, project, cycle, options, snapshot); bindErr != nil {
				return nil, bindErr
			}

		}
		return reader.issueResults(ctx, actor, slug, project, cycle, options)
	})
	return result, cycleDiscoveryError(err)
}

func (repository *Repository) bindPropertyFilters(ctx context.Context, actor uuid.UUID, slug string, project, cycle uuid.UUID, options IssueListOptions, snapshot *workitemtypes.PropertyDiscoverySnapshot) error {
	args := []any{actor, slug, project, cycle}
	joins, predicates, err := issueFilterSource(options, &args, false)
	if err != nil {
		return err
	}
	query := issueRequestScopeSQL + ` SELECT COALESCE(jsonb_agg(candidate),'[]'::jsonb) FROM (
 SELECT issue.id AS "ID",issue.project_id AS "ProjectID",issue.type_id AS "TypeID",
 issue.estimate_point_id AS "EstimatePointID",issue.custom_properties AS "Saved",
 jsonb_build_object('Name',issue.name,'Priority',issue.priority,'StartDate',issue.start_date,'DueDate',issue.target_date) AS "Standard"
 ` + fmt.Sprintf(issueFilterFromSQL, strings.Join(joins, "\n")) + ` WHERE ` + strings.Join(predicates, " AND ") + `) candidate`
	var raw []byte
	if err = repository.database.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return err
	}
	var items []workitemtypes.DiscoveryItem
	if err = json.Unmarshal(raw, &items); err != nil {
		return err
	}
	values, err := snapshot.ProjectItems(ctx, items)
	if err != nil {
		return err
	}
	for _, entry := range options.properties.entries {
		for index, item := range items {
			match, known := entry.predicate.Match(values[index].Values)
			if known {
				entry.matches[item.ID] = match
			}
		}
	}
	return nil
}
