package modulereads

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

type modulePropertyFilter struct {
	snapshot   *workitemtypes.PropertyDiscoverySnapshot
	plan       *issuequery.PreparedPropertyFilter
	predicates map[int]*workitemtypes.ScopedPropertyPredicate
}

func (repository *Repository) propertyIssues(ctx context.Context, actor uuid.UUID, slug string, project, module uuid.UUID, options IssueListOptions) (IssuePage, error) {
	// The module endpoint requires an elevated project role before schema reads;
	// issueResults rechecks it inside the repeatable-read transaction.
	if err := repository.authorizeProject(ctx, actor, slug, project, true); err != nil {
		return IssuePage{}, err
	}
	readSnapshot := workitemtypes.WithPropertyDiscoveryRead[IssuePage]
	if !issuequery.UsesPropertyFilters(options.Filters) && !issuequery.UsesTypeFilters(options.Filters) {
		readSnapshot = workitemtypes.WithPropertyProjectionRead[IssuePage]
	}
	page, err := readSnapshot(ctx, repository.database, actor.String(), slug, project.String(), func(tx pgx.Tx, snapshot *workitemtypes.PropertyDiscoverySnapshot) (IssuePage, error) {
		reader := *repository
		reader.database = tx
		if accessErr := reader.authorizeProject(ctx, actor, slug, project, true); accessErr != nil {
			return IssuePage{}, accessErr
		}
		if issuequery.UsesPropertyFilters(options.Filters) {
			filters := &modulePropertyFilter{snapshot: snapshot, predicates: map[int]*workitemtypes.ScopedPropertyPredicate{}}
			plan, validationErr := issuequery.PreparePropertyFilter(options.Filters, snapshot.Types.TypeIDs(), func(field string, value any) (bool, error) {
				predicate, handled, fieldErr := snapshot.FilterPredicate(ctx, tx, field, value)
				if fieldErr == nil && handled {
					filters.predicates[len(filters.predicates)+1] = predicate
				}
				return handled, fieldErr
			})
			if validationErr != nil {
				return IssuePage{}, validationErr
			}
			filters.plan, reader.propertyFilters = plan, filters

		}
		page, readErr := reader.issueResults(ctx, actor, slug, project, module, options, snapshot.Types.TypeIDs())
		if readErr != nil || !options.IncludeProperties {
			return page, readErr
		}
		var rows []map[string]any
		decoder := json.NewDecoder(bytes.NewReader(page.Items))
		decoder.UseNumber()
		if decodeErr := decoder.Decode(&rows); decodeErr != nil {
			return IssuePage{}, decodeErr
		}
		if projectionErr := snapshot.ProjectStoredRows(ctx, rows, workitemtypes.DiscoveryElevatedProjectVisibility); projectionErr != nil {
			return IssuePage{}, projectionErr
		}
		page.Items, readErr = json.Marshal(rows)
		return page, readErr
	})
	return page, moduleDiscoveryError(err)
}

// Candidates use exactly the final query's route/flat predicates and joins, but
// omit the entire structured expression. All candidates are projected before
// binding the expression, counting, ordering, grouping or taking a page.
func (repository *Repository) bindPropertyFilters(ctx context.Context, where []string, args []any) (*issuequery.BoundPropertyFilter, error) {
	query := `WITH request_actor AS (SELECT $1::uuid AS id)
SELECT COALESCE(jsonb_agg(candidate), '[]'::jsonb) FROM (
 SELECT issue.id AS "ID", issue.project_id AS "ProjectID", issue.type_id AS "TypeID",
 issue.estimate_point_id AS "EstimatePointID", issue.custom_properties AS "Saved",
 jsonb_build_object('Name',issue.name,'Priority',issue.priority,'StartDate',issue.start_date,'DueDate',issue.target_date) AS "Standard"
 ` + moduleIssueFromSQL + ` CROSS JOIN request_actor
 WHERE ` + strings.Join(where, " AND ") + `) candidate`
	var raw []byte
	if err := repository.database.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return nil, err
	}
	var items []workitemtypes.DiscoveryItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	values, err := repository.propertyFilters.snapshot.ProjectItems(ctx, items)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(items))
	for index, item := range items {
		ids[index] = item.ID
	}
	return repository.propertyFilters.plan.BindEvaluatedItems(ids, func(leafID, index int) (bool, bool) {
		return repository.propertyFilters.predicates[leafID].Match(values[index].Values)
	})
}

const moduleIssueFromSQL = `FROM issues issue
  JOIN module_issues module_issue ON module_issue.issue_id = issue.id
  JOIN modules module
    ON module.id = module_issue.module_id
    AND module.project_id = module_issue.project_id
    AND module.workspace_id = module_issue.workspace_id
  JOIN projects project
    ON project.id = issue.project_id
    AND project.id = module_issue.project_id
    AND project.workspace_id = issue.workspace_id
  JOIN workspaces workspace
    ON workspace.id = issue.workspace_id
    AND workspace.id = module_issue.workspace_id`
