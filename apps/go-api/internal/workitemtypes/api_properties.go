package workitemtypes

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PropertyAPIQuery struct {
	APIScope
	Action                       string
	PropertyID, TypeID, OptionID uuid.UUID
	Limit, Offset                int
	IncludeRevision              bool
}
type PropertyAPIChanges struct {
	DisplayName, Kind, RelationType, Description, DisplayFormat, ExternalSource, ExternalID *string
	Required, Multiple, Active                                                              *bool
	Defaults                                                                                []string
	Options                                                                                 []PropertyAPIOption
	Configuration                                                                           *PropertyAPIConfiguration
}

// PropertyAPIConfiguration exposes native kind-specific settings without
// creating a second configuration validator or overriding the SDK fields.
type PropertyAPIConfiguration struct {
	Expression *string            `json:"expression"`
	Result     *formula.Kind      `json:"result"`
	Content    *string            `json:"content"`
	Levels     *[]property.Level  `json:"levels"`
	Options    *[]property.Option `json:"options"`
}
type PropertyAPIOption struct {
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	Color          *string `json:"color"`
	ExternalSource *string `json:"external_source"`
	ExternalID     *string `json:"external_id"`
	Default        *bool   `json:"is_default"`
}
type PropertyAPIMutation struct {
	PropertyAPIQuery
	Changes          PropertyAPIChanges
	Option           PropertyAPIOption
	Attach, Detach   []uuid.UUID
	RequestID        *uuid.UUID
	ExpectedRevision *string
	Revalidate       func(context.Context) error
}
type apiPropertyRow struct {
	Definition                                 property.Definition
	Metadata                                   map[string]any
	CreatedAt, UpdatedAt, CreatedBy, UpdatedBy any
}

func propertyRevision(workspace uuid.UUID, revision int64) string {
	return fmt.Sprintf("workitem_property:%s:%d", workspace, revision)
}
func loadAPIPropertyRows(ctx context.Context, tx pgx.Tx, q PropertyAPIQuery) ([]apiPropertyRow, error) {
	rows, err := tx.Query(ctx, `SELECT definition,api_metadata,created_at,updated_at,created_by_id::text,updated_by_id::text FROM work_item_properties WHERE workspace_id=$1 ORDER BY created_at,id`, q.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []apiPropertyRow{}
	for rows.Next() {
		var row apiPropertyRow
		var raw, meta []byte
		if err = rows.Scan(&raw, &meta, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy, &row.UpdatedBy); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &row.Definition); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(meta, &row.Metadata); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
func (r *Repository) ReadPropertyAPI(ctx context.Context, q PropertyAPIQuery) (json.RawMessage, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return nil, err
	}
	if _, err = authorizeAPI(ctx, tx, q.APIScope, false); err != nil {
		return nil, err
	}
	value, err := readPropertyAPI(ctx, tx, q)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
func readPropertyAPI(ctx context.Context, tx pgx.Tx, q PropertyAPIQuery) (any, error) {
	all, err := loadAPIPropertyRows(ctx, tx, q)
	if err != nil {
		return nil, err
	}
	selected, err := selectAPIProperties(ctx, tx, q, all)
	if err != nil {
		return nil, err
	}
	if q.PropertyID != uuid.Nil && len(selected) == 0 {
		return nil, projectmutations.ErrNotFound
	}
	var revision string
	if q.IncludeRevision {
		var rev int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, q.WorkspaceID).Scan(&rev); err != nil {
			return nil, err
		}
		revision = propertyRevision(q.WorkspaceID, rev)
	}
	result := []map[string]any{}
	for _, row := range selected {
		config := row.Definition.Configuration
		if q.ProjectID != uuid.Nil || q.TypeID != uuid.Nil {
			config, _, err = property.Effective(row.Definition, q.ProjectID.String(), q.TypeID.String())
			if err != nil {
				return nil, err
			}
		}
		if q.Action == "list_options" || q.Action == "retrieve_option" {
			for index, option := range config.Options {
				if q.OptionID != uuid.Nil && option.ID != q.OptionID.String() {
					continue
				}
				record := projectAPIOption(row, config, option, index, q)
				if revision != "" {
					record["revision"] = revision
				}
				result = append(result, record)
			}
		} else {
			record := projectAPIProperty(row, config, q)
			if revision != "" {
				record["revision"] = revision
			}
			result = append(result, record)
		}
	}
	if q.Action == "retrieve" || q.Action == "retrieve_option" {
		if len(result) != 1 {
			return nil, projectmutations.ErrNotFound
		}
		return result[0], nil
	}
	if q.ProjectID != uuid.Nil && q.Limit > 0 {
		start := min(q.Offset, len(result))
		result = result[start:min(start+q.Limit, len(result))]
	}
	return result, nil
}
func selectAPIProperties(ctx context.Context, tx pgx.Tx, q PropertyAPIQuery, all []apiPropertyRow) ([]apiPropertyRow, error) {
	if q.TypeID != uuid.Nil {
		var found bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_types t WHERE t.id=$1 AND t.workspace_id=$2 AND t.deleted_at IS NULL)`, q.TypeID, q.WorkspaceID).Scan(&found)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, projectmutations.ErrNotFound
		}
	}
	if q.ProjectID == uuid.Nil && q.TypeID == uuid.Nil {
		return filterAPIPropertyID(all, q.PropertyID), nil
	}
	rows, err := tx.Query(ctx, `SELECT DISTINCT l.property_id::text FROM work_item_type_properties l JOIN issue_types t ON t.id=l.type_id WHERE t.workspace_id=$1 AND t.deleted_at IS NULL AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR t.id=$2) AND ($3::uuid='00000000-0000-0000-0000-000000000000' OR t.is_default OR t.mandatory OR EXISTS(SELECT 1 FROM project_issue_types pt WHERE pt.issue_type_id=t.id AND pt.project_id=$3 AND pt.workspace_id=$1 AND pt.is_active AND pt.deleted_at IS NULL))`, q.WorkspaceID, q.TypeID, q.ProjectID)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	selected := []apiPropertyRow{}
	for _, row := range all {
		if slices.Contains(ids, row.Definition.ID) {
			selected = append(selected, row)
		}
	}
	if q.Action == "list" && len(selected) == 0 && q.ProjectID != uuid.Nil && q.TypeID != uuid.Nil {
		flat := q
		flat.TypeID = uuid.Nil
		selected, err = selectAPIProperties(ctx, tx, flat, all)
		if err != nil {
			return nil, err
		}
		if len(selected) == 0 {
			workspace := q
			workspace.ProjectID = uuid.Nil
			selected, err = selectAPIProperties(ctx, tx, workspace, all)
		}
	}
	return filterAPIPropertyID(selected, q.PropertyID), err
}
func filterAPIPropertyID(rows []apiPropertyRow, id uuid.UUID) []apiPropertyRow {
	if id == uuid.Nil {
		return rows
	}
	result := []apiPropertyRow{}
	for _, row := range rows {
		if row.Definition.ID == id.String() {
			result = append(result, row)
		}
	}
	return result
}
