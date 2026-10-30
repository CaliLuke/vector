package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PropertyValueAPIQuery struct {
	APIScope
	IssueID, PropertyID uuid.UUID
	IncludeRevision     bool
}
type PropertyValueScope struct {
	Definition    property.Definition
	Configuration property.Configuration
	Value         any
	Present       bool
	typeID        string
	stored        map[string]any
}

// LoadPropertyValueScope is shared by authorized reads and native item writes.
// The caller holds any required schema lock before asking for a write lock.
func LoadPropertyValueScope(ctx context.Context, tx pgx.Tx, q PropertyValueAPIQuery, write bool) (PropertyValueScope, error) {
	var out PropertyValueScope
	if _, err := authorizeAPI(ctx, tx, q.APIScope, write); err != nil {
		return out, err
	}
	suffix := ""
	if write {
		suffix = " FOR UPDATE OF i"
	}
	var definition, values []byte
	var typeID string
	err := tx.QueryRow(ctx, `SELECT p.definition,i.custom_properties,i.type_id::text FROM issues i JOIN issue_types t ON t.id=i.type_id AND t.workspace_id=i.workspace_id AND t.deleted_at IS NULL JOIN work_item_type_settings s ON s.workspace_id=i.workspace_id AND s.enabled JOIN work_item_type_properties l ON l.type_id=i.type_id AND l.property_id=$2 JOIN work_item_properties p ON p.id=l.property_id AND p.workspace_id=i.workspace_id WHERE i.id=$1 AND i.project_id=$3 AND `+djangoapi.IssueVisibleSQL("i", "$4::uuid", "$5::uuid")+suffix, q.IssueID, q.PropertyID, q.ProjectID, q.ActorID, q.WorkspaceID).Scan(&definition, &values, &typeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, projectmutations.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(definition, &out.Definition); err != nil {
		return out, err
	}
	out.Configuration, _, err = property.Effective(out.Definition, q.ProjectID.String(), typeID)
	if err != nil {
		return out, err
	}
	if !out.Configuration.Active {
		return out, projectmutations.ErrNotFound
	}
	var stored map[string]any
	if err = json.Unmarshal(values, &stored); err != nil {
		return out, err
	}
	out.Value, out.Present = stored[q.PropertyID.String()]
	out.typeID, out.stored = typeID, stored
	return out, nil
}
func (r *Repository) ReadPropertyValueAPI(ctx context.Context, q PropertyValueAPIQuery) (_ json.RawMessage, resultErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return nil, err
	}
	raw, err := ProjectPropertyValueAPI(ctx, tx, q)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return raw, nil
}
func ProjectPropertyValueAPI(ctx context.Context, tx pgx.Tx, q PropertyValueAPIQuery) (json.RawMessage, error) {
	scope, err := LoadPropertyValueScope(ctx, tx, q, false)
	if err != nil {
		return nil, err
	}
	if scope.Definition.Kind == property.Formula || scope.Definition.Kind == property.Text && scope.Configuration.Format == "Read-only" {
		return projectComputedPropertyValueAPI(ctx, tx, q, scope)
	}
	if !scope.Present {
		return nil, projectmutations.ErrNotFound
	}
	canonical, err := property.ReadValue(scope.Definition.Kind, scope.Configuration, scope.Value)
	if err != nil {
		return nil, err
	}
	kind, _ := apiKind(scope.Definition.Kind)
	rows, err := tx.Query(ctx, `SELECT id::text,value_key,external_source,external_id,created_at,updated_at FROM work_item_property_value_identities WHERE issue_id=$1 AND property_id=$2 AND workspace_id=$3 AND project_id=$4 AND deleted_at IS NULL ORDER BY created_at,id`, q.IssueID, q.PropertyID, q.WorkspaceID, q.ProjectID)
	if err != nil {
		return nil, err
	}
	records := map[string]map[string]any{}
	for rows.Next() {
		var id, key string
		var source, external *string
		var created, updated any
		if err = rows.Scan(&id, &key, &source, &external, &created, &updated); err != nil {
			rows.Close()
			return nil, err
		}
		records[key] = map[string]any{"id": id, "property_id": q.PropertyID.String(), "issue_id": q.IssueID.String(), "value_type": kind, "external_source": source, "external_id": external, "created_at": created, "updated_at": updated, "value_detail": nil}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	revision := ""
	if q.IncludeRevision {
		var rev int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM issue_edit_revisions WHERE issue_id=$1`, q.IssueID).Scan(&rev); err != nil {
			return nil, err
		}
		revision = fmt.Sprintf("workitem_property_value:%s:%d", q.IssueID, rev)
	}
	if scope.Configuration.Multiple || scope.Definition.Kind == property.Cascading {
		out := []map[string]any{}
		for _, id := range memberValueIDs(canonical) {
			record := records[id]
			if record == nil {
				return nil, projectmutations.ErrNotFound
			}
			record["value"] = id
			if revision != "" {
				record["revision"] = revision
			}
			out = append(out, record)
		}
		return json.Marshal(out)
	}
	record := records[""]
	if record == nil {
		return nil, projectmutations.ErrNotFound
	}
	record["value"] = canonical
	if scope.Definition.Kind == property.RichText {
		html, _ := canonical.(string)
		record["value"] = record["id"]
		record["value_detail"] = map[string]any{"id": record["id"], "description_html": html, "description_stripped": content.StripTags(html)}
	}
	if revision != "" {
		record["revision"] = revision
	}
	return json.Marshal(record)
}
