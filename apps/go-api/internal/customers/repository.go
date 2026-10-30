package customers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrDisabled = errors.New("customers feature is disabled")

type Repository struct{ db mutationreceipts.Database }

func NewRepository(db mutationreceipts.Database) *Repository { return &Repository{db: db} }

type Query struct {
	ActorID, WorkspaceID, ID, CustomerID, RequestID uuid.UUID
	WorkspaceSlug, Tool, Action, Search             string
	Params                                          map[string]string
	Present                                         map[string]bool
	IncludeRevision                                 bool
}
type Command struct {
	Query
	Fields           map[string]any
	Options          []map[string]any
	Values           map[string][]string
	Add, Remove      []uuid.UUID
	RetryID          *uuid.UUID
	ExpectedRevision *string
	Revalidate       func(context.Context) error
}

func token(workspace uuid.UUID, rev int64) string {
	return fmt.Sprintf("customer:%s:%d", workspace, rev)
}
func table(tool string) string {
	switch tool {
	case "customer":
		return "customers"
	case "customer_request":
		return "customer_requests"
	case "customer_property":
		return "customer_properties"
	}
	return ""
}
func optionalID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
func scope(ctx context.Context, tx pgx.Tx, q Query, write bool) error {
	if table(q.Tool) == "" {
		return projectmutations.ErrInvalid
	}
	role, err := projectmutations.WorkspaceRole(ctx, tx, q.ActorID, q.WorkspaceID, q.WorkspaceSlug, write)
	if err != nil {
		return err
	}
	if write && role != 20 {
		return projectmutations.ErrForbidden
	}
	suffix := ""
	if write {
		suffix = " FOR SHARE"
	}
	var enabled bool
	if err = tx.QueryRow(ctx, `SELECT customers FROM workspace_features WHERE workspace_id=$1`+suffix, q.WorkspaceID).Scan(&enabled); err != nil {
		return err
	}
	if !enabled {
		return ErrDisabled
	}
	return nil
}
func rows(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]map[string]any, error) {
	return djangoapi.ReadJSONRows(ctx, tx, query, args...)
}
func record(ctx context.Context, tx pgx.Tx, q Query, id uuid.UUID) (map[string]any, error) {
	query := `SELECT to_jsonb(t) FROM ` + table(q.Tool) + ` t WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`
	args := []any{id, q.WorkspaceID}
	if q.Tool == "customer_request" {
		query += ` AND customer_id=$3`
		args = append(args, q.CustomerID)
	}
	values, err := rows(ctx, tx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, projectmutations.ErrNotFound
	}
	return values[0], nil
}
func customerExists(ctx context.Context, tx pgx.Tx, q Query) error {
	if q.CustomerID == uuid.Nil {
		return projectmutations.ErrInvalid
	}
	selected := q
	selected.Tool = "customer"
	_, err := record(ctx, tx, selected, q.CustomerID)
	return err
}
func requestExists(ctx context.Context, tx pgx.Tx, q Query) error {
	selected := q
	selected.Tool = "customer_request"
	_, err := record(ctx, tx, selected, q.RequestID)
	return err
}
func (r *Repository) Read(ctx context.Context, q Query) (_ json.RawMessage, resultErr error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if err = scope(ctx, tx, q, false); err != nil {
		return nil, err
	}
	if q.Tool == "customer_request" || q.Action == "get_values" || q.Action == "list_workitems" {
		if err = customerExists(ctx, tx, q); err != nil {
			return nil, err
		}
	}
	if q.Tool == "customer" && q.RequestID != uuid.Nil {
		if err = requestExists(ctx, tx, q); err != nil {
			return nil, err
		}
	}
	var value any
	switch q.Action {
	case "list":
		value, err = page(ctx, tx, q)
	case "retrieve":
		var row map[string]any
		row, err = record(ctx, tx, q, q.ID)
		if err == nil {
			value, err = projection(ctx, tx, q, row)
		}
	case "list_workitems":
		value, err = linkedItems(ctx, tx, q)
	case "get_values":
		value, err = readValues(ctx, tx, q)
	default:
		return nil, projectmutations.ErrInvalid
	}
	if err != nil {
		return nil, err
	}
	if q.IncludeRevision && q.Action != "get_values" {
		var rev int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM customer_revisions WHERE workspace_id=$1`, q.WorkspaceID).Scan(&rev); err != nil {
			return nil, err
		}
		switch v := value.(type) {
		case map[string]any:
			v["revision"] = token(q.WorkspaceID, rev)
		case []map[string]any:
			for _, row := range v {
				row["revision"] = token(q.WorkspaceID, rev)
			}
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, tx.Commit(ctx)
}
func page(ctx context.Context, tx pgx.Tx, q Query) (map[string]any, error) {
	limit, p, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return nil, err
	}
	from := ` FROM ` + table(q.Tool) + ` t WHERE workspace_id=$1 AND deleted_at IS NULL`
	args := []any{q.WorkspaceID}
	if q.Tool == "customer_request" {
		from += ` AND customer_id=$2`
		args = append(args, q.CustomerID)
	}
	if q.Tool != "customer_property" {
		args = append(args, q.Search)
		from += fmt.Sprintf(` AND strpos(lower(name),lower($%d))>0`, len(args))
	}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*)`+from, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, limit, offset)
	list, err := rows(ctx, tx, `SELECT to_jsonb(t)`+from+fmt.Sprintf(` ORDER BY created_at DESC,id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	values := []any{}
	for _, row := range list {
		value, err := projection(ctx, tx, q, row)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return djangoapi.Envelope(values, total, limit, p), nil
}
func projection(ctx context.Context, tx pgx.Tx, q Query, row map[string]any) (map[string]any, error) {
	keys := []string{"id", "name", "deleted_at", "created_at", "updated_at", "description", "description_html", "description_stripped", "description_binary", "email", "website_url", "logo_props", "domain", "employees", "stage", "contract_status", "revenue", "external_source", "external_id", "archived_at"}
	if q.Tool == "customer_request" {
		keys = []string{"id", "name", "description", "description_html", "link", "created_at"}
	}
	if q.Tool == "customer_property" {
		keys = []string{"id", "deleted_at", "created_at", "updated_at", "name", "display_name", "description", "logo_props", "sort_order", "property_type", "relation_type", "is_required", "default_value", "settings", "is_active", "is_multi", "validation_rules", "external_source", "external_id"}
	}
	out := map[string]any{}
	for _, key := range keys {
		out[key] = row[key]
	}
	if q.Tool == "customer_request" {
		out["work_item_ids"], out["attachment_count"] = nil, 0
		return out, nil
	}
	out["workspace"], out["created_by"], out["updated_by"] = row["workspace_id"], row["created_by_id"], row["updated_by_id"]
	if q.Tool == "customer" {
		var count int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM customer_requests WHERE customer_id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, row["id"], q.WorkspaceID).Scan(&count)
		out["customer_request_count"], out["logo_url"], out["logo_asset"] = count, nil, nil
		return out, err
	}
	options, err := rows(ctx, tx, `SELECT jsonb_build_object('id',id,'deleted_at',deleted_at,'created_at',created_at,'updated_at',updated_at,'name',name,'sort_order',sort_order,'description',description,'logo_props','{}'::jsonb,'is_active',is_active,'is_default',is_default,'external_source',NULL,'external_id',NULL,'created_by',created_by_id,'updated_by',updated_by_id,'workspace',workspace_id,'property',property_id,'parent',NULL) FROM customer_property_options WHERE property_id=$1 AND workspace_id=$2 AND deleted_at IS NULL ORDER BY sort_order,id`, row["id"], q.WorkspaceID)
	if err != nil {
		return nil, err
	}
	out["options"] = options
	if row["property_type"] == "RELATION" {
		defaults := stringValues(row["default_value"])
		out["default_value"], err = visibleValues(ctx, tx, q, row, defaults)
	}
	return out, err
}
func linkedItems(ctx context.Context, tx pgx.Tx, q Query) ([]map[string]any, error) {
	return rows(ctx, tx, `SELECT jsonb_build_object('id',i.id,'name',i.name,'state_id',(SELECT s.id FROM states s WHERE s.id=i.state_id AND s.project_id=i.project_id AND s.workspace_id=i.workspace_id AND s.deleted_at IS NULL),'priority',i.priority,'sequence_id',i.sequence_id,'project_id',i.project_id,'project_identifier',p.identifier,'created_at',i.created_at,'updated_at',i.updated_at) FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE EXISTS(SELECT 1 FROM customer_work_items rel WHERE rel.issue_id=i.id AND rel.project_id=i.project_id AND rel.workspace_id=$1 AND rel.customer_id=$2 AND ($3::uuid IS NULL OR rel.request_id=$3)) AND p.archived_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$4::uuid", "$1::uuid")+` AND (strpos(lower(i.name),lower($5))>0 OR strpos(i.sequence_id::text,$5)>0 OR strpos(lower(p.identifier),lower($5))>0) ORDER BY i.created_at DESC,i.id`, q.WorkspaceID, q.CustomerID, optionalID(q.RequestID), q.ActorID, q.Search)
}
