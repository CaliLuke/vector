package workitems

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
)

// projectLockKey must agree with Django's convert_uuid_to_integer. Both
// runtimes can create work items against the same sequence ledger during rollout.
func projectLockKey(project string) int64 {
	hash := sha256.Sum256([]byte(project))
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

func defaultIssue(project string) map[string]any {
	return map[string]any{
		"id": uuid.NewString(), "project_id": project, "workspace_id": nil,
		"name": "", "description_html": "<p></p>", "description_json": map[string]any{},
		"custom_properties":    map[string]any{},
		"description_stripped": "", "description_binary": nil, "parent_id": nil, "state_id": nil,
		"estimate_point_id": nil, "type_id": nil, "point": nil, "priority": "none",
		"start_date": nil, "target_date": nil, "sequence_id": 1, "sort_order": 65535,
		"completed_at": nil, "archived_at": nil, "is_draft": false, "external_id": nil, "external_source": nil,
		"created_at": nil, "updated_at": nil, "created_by_id": nil, "updated_by_id": nil, "deleted_at": nil,
	}
}

func timestamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000Z") }

func stripTags(value string) any {
	if value == "" {
		return nil
	}
	return content.StripTags(value)
}

// saveIssue applies Issue.save and ProjectBaseModel.save. Callers retain the
// serializer's operation-specific validation and the timing of durable effects.
func (r *Repository) saveIssueRow(ctx context.Context, actor string, row map[string]any, old map[string]any) (map[string]any, error) {
	project, err := r.project(ctx, row["project_id"].(string))
	if err != nil {
		return nil, err
	}
	row["workspace_id"] = project["workspace_id"]
	if row["state_id"] == nil {
		var state *string
		err = r.db.QueryRow(ctx, `SELECT (SELECT id::text FROM states WHERE project_id=$1 AND deleted_at IS NULL AND NOT is_triage ORDER BY "default" DESC,sequence ASC LIMIT 1)`, row["project_id"]).Scan(&state)
		if err != nil {
			return nil, err
		}
		if state != nil {
			row["state_id"] = *state
		}
	}
	now := timestamp(r.now())
	if old == nil || !reflect.DeepEqual(old["state_id"], row["state_id"]) {
		if row["state_id"] != nil {
			var completed bool
			err = r.db.QueryRow(ctx, `SELECT "group"='completed' FROM states WHERE id=$1`, row["state_id"]).Scan(&completed)
			if err != nil {
				return nil, err
			}
			row["completed_at"] = nil
			if completed {
				row["completed_at"] = now
			}
		}
	}
	row["description_stripped"] = stripTags(row["description_html"].(string))
	row["updated_at"] = now
	if old == nil {
		row["created_at"], row["created_by_id"], row["updated_by_id"] = now, actor, nil
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		// All schema-aware creation paths take locks in the same order.
		if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, row["workspace_id"]); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, projectLockKey(row["project_id"].(string))); err != nil {
			return nil, err
		}
		var sequence int64
		var order *float64
		err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM issue_sequences WHERE project_id=$1 AND deleted_at IS NULL`, row["project_id"]).Scan(&sequence)
		if err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx, `SELECT MAX(sort_order) FROM issues WHERE project_id=$1 AND state_id IS NOT DISTINCT FROM $2::uuid AND deleted_at IS NULL`, row["project_id"], row["state_id"]).Scan(&order)
		if err != nil {
			return nil, err
		}
		row["sequence_id"] = sequence
		if order != nil {
			row["sort_order"] = *order + 10000
		}
		encoded, err := issueRecordJSON(row)
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO issues SELECT (jsonb_populate_record(NULL::issues,$1::jsonb)).*`, encoded); err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO issue_sequences (id,issue_id,project_id,workspace_id,sequence,deleted,created_at,updated_at,created_by_id,updated_by_id)
 VALUES ($1,$2,$3,$4,$5,false,$6,$6,$7,NULL)`, uuid.NewString(), row["id"], row["project_id"], row["workspace_id"], sequence, now, actor)
		if err != nil {
			return nil, err
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
	} else {
		row["updated_by_id"] = actor
		encoded, err := issueRecordJSON(row)
		if err != nil {
			return nil, err
		}
		// Only model columns are assigned; annotations and many-to-many arrays
		// are never trusted as database column names.
		_, err = r.db.Exec(ctx, `UPDATE issues i SET
 (name,description_html,description_json,description_binary,description_stripped,parent_id,state_id,
 estimate_point_id,type_id,point,priority,start_date,target_date,sequence_id,sort_order,completed_at,
 archived_at,is_draft,external_id,external_source,created_at,created_by_id,updated_at,updated_by_id,deleted_at,project_id,workspace_id,custom_properties)=
 (v.name,v.description_html,v.description_json,v.description_binary,v.description_stripped,v.parent_id,v.state_id,
 v.estimate_point_id,v.type_id,v.point,v.priority,v.start_date,v.target_date,v.sequence_id,v.sort_order,v.completed_at,
 v.archived_at,v.is_draft,v.external_id,v.external_source,v.created_at,v.created_by_id,v.updated_at,v.updated_by_id,v.deleted_at,v.project_id,v.workspace_id,v.custom_properties)
 FROM jsonb_populate_record(NULL::issues,$1::jsonb) v WHERE i.id=v.id`, encoded)
		if err != nil {
			return nil, err
		}
	}
	return r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE id=$1`, row["id"])
}

func issueRecordJSON(row map[string]any) ([]byte, error) {
	copy := make(map[string]any, len(row))
	for k, v := range row {
		copy[k] = v
	}
	// jsonb_populate_record expects PostgreSQL bytea text rather than the
	// serializer's base64 representation. Binary conversion happens at validation.
	if value, ok := copy["description_binary"].([]byte); ok {
		copy["description_binary"] = `\x` + hexBytes(value)
	}
	if value, ok := copy["description_binary"].(string); ok {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, err
		}
		copy["description_binary"] = `\x` + hexBytes(decoded)
	}
	return json.Marshal(copy)
}
func hexBytes(data []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for i, b := range data {
		out[i*2] = digits[b>>4]
		out[i*2+1] = digits[b&15]
	}
	return string(out)
}

func (r *Repository) deleteIssue(ctx context.Context, actor string, row map[string]any) error {
	old := make(map[string]any, len(row))
	for k, v := range row {
		old[k] = v
	}
	row["deleted_at"] = timestamp(r.now())
	if _, err := r.saveIssue(ctx, actor, row, old); err != nil {
		return err
	}
	return r.softDelete(ctx, "issue", row["id"].(string))
}
