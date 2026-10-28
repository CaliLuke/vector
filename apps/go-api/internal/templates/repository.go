// Package templates owns reusable resource definitions without executing them.
package templates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/djangoapi"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrWorkspaceOwned = errors.New("templates are managed by the workspace")

type Repository struct{ db mutationreceipts.Database }

func NewRepository(db mutationreceipts.Database) *Repository { return &Repository{db: db} }

type Query struct {
	ActorID, WorkspaceID, ProjectID, ID uuid.UUID
	WorkspaceSlug, Kind                 string
	IncludeRevision                     bool
}
type Command struct {
	Query
	Action            string
	Name, Description *string
	Data              map[string]any
	RequestID         *uuid.UUID
	ExpectedRevision  *string
	Revalidate        func(context.Context) error
}

func token(workspace uuid.UUID, revision int64) string {
	return fmt.Sprintf("template:%s:%d", workspace, revision)
}
func scope(ctx context.Context, tx pgx.Tx, q Query, write bool) error {
	if q.ActorID == uuid.Nil || q.WorkspaceID == uuid.Nil || q.WorkspaceSlug == "" || q.Kind != "workitem" && q.Kind != "page" && q.Kind != "project" || q.Kind == "project" && q.ProjectID != uuid.Nil {
		return projectmutations.ErrInvalid
	}
	suffix := ""
	if write {
		suffix = " FOR SHARE OF u,wm,w"
	}
	var role int
	err := tx.QueryRow(ctx, `SELECT wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)`+suffix, q.ActorID, q.WorkspaceID, q.WorkspaceSlug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectmutations.ErrForbidden
	}
	if err != nil {
		return err
	}
	if q.ProjectID != uuid.Nil {
		suffix = ""
		if write {
			suffix = " FOR SHARE OF p,pm"
		}
		err = tx.QueryRow(ctx, `SELECT pm.role FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20)`+suffix, q.ProjectID, q.WorkspaceID, q.ActorID).Scan(&role)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrForbidden
		}
		if err != nil {
			return err
		}
	}
	if write && role != 20 {
		return projectmutations.ErrForbidden
	}
	if write {
		var owned bool
		if err = tx.QueryRow(ctx, `SELECT owned_by_workspace FROM workspace_template_settings WHERE workspace_id=$1 FOR UPDATE`, q.WorkspaceID).Scan(&owned); err != nil {
			return err
		}
		if owned && q.ProjectID != uuid.Nil {
			return ErrWorkspaceOwned
		}
	}
	return nil
}
func projectID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
func projection(row map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"id", "name", "template_data", "created_at", "updated_at"} {
		out[key] = row[key]
	}
	out["workspace"] = row["workspace_id"]
	if row["project_id"] != nil {
		out["project"], out["short_description"], out["template_type"] = row["project_id"], row["summary"], row["kind"]
	} else {
		out["description"], out["logo_props"] = map[string]any{}, row["logo_props"]
		if row["kind"] == "project" {
			out["description"] = row["summary"]
		} else {
			out["description_html"] = row["summary"]
		}
		out["project"] = nil
	}
	return out
}
func (r *Repository) List(ctx context.Context, q Query) (json.RawMessage, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err = scope(ctx, tx, q, false); err != nil {
		return nil, err
	}
	var revision int64
	if q.IncludeRevision {
		if err = tx.QueryRow(ctx, `SELECT revision FROM workspace_template_settings WHERE workspace_id=$1`, q.WorkspaceID).Scan(&revision); err != nil {
			return nil, err
		}
	}
	rows, err := tx.Query(ctx, `SELECT to_jsonb(t) FROM resource_templates t WHERE t.workspace_id=$1 AND t.kind=$2 AND t.deleted_at IS NULL AND ($3::uuid IS NULL OR t.project_id=$3) AND (t.project_id IS NULL OR EXISTS(SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id WHERE p.id=t.project_id AND p.workspace_id=t.workspace_id AND p.deleted_at IS NULL AND p.archived_at IS NULL AND pm.member_id=$4 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20))) ORDER BY t.created_at DESC,t.id`, q.WorkspaceID, q.Kind, projectID(q.ProjectID), q.ActorID)
	if err != nil {
		return nil, err
	}
	values := []map[string]any{}
	for rows.Next() {
		var raw []byte
		var row map[string]any
		if err = rows.Scan(&raw); err != nil {
			break
		}
		if err = djangoapi.DecodeJSON(raw, &row); err != nil {
			break
		}
		value := projection(row)
		if q.IncludeRevision {
			value["revision"] = token(q.WorkspaceID, revision)
		}
		values = append(values, value)
	}
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return raw, tx.Commit(ctx)
}
func normalize(in *Command) error {
	if in.Action != "create" && in.Action != "update" && in.Action != "delete" || in.Action == "create" && in.ID != uuid.Nil || in.Action != "create" && in.ID == uuid.Nil || in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return projectmutations.ErrInvalid
	}
	if in.Name != nil {
		value := strings.TrimSpace(*in.Name)
		if value == "" || utf8.RuneCountInString(value) > 255 || strings.ContainsRune(value, 0) {
			return projectmutations.ErrInvalid
		}
		in.Name = &value
	}
	if in.Description != nil {
		value := *in.Description
		if strings.ContainsRune(value, 0) {
			return projectmutations.ErrInvalid
		}
		if in.ProjectID == uuid.Nil && in.Kind != "project" {
			clean, _, err := content.ValidateHTML(value)
			if err != nil {
				return projectmutations.ErrInvalid
			}
			value = clean
		}
		in.Description = &value
	}
	if in.Data != nil {
		raw, err := json.Marshal(in.Data)
		if err != nil || len(raw) > 1<<20 {
			return projectmutations.ErrInvalid
		}
		var copy map[string]any
		if djangoapi.DecodeJSON(raw, &copy) != nil {
			return projectmutations.ErrInvalid
		}
		in.Data = copy
	}
	if in.Action == "create" && (in.Name == nil || in.Data == nil) || in.Action == "update" && (in.Name == nil && in.Description == nil && len(in.Data) == 0 || in.Data != nil && len(in.Data) == 0) || in.Action == "delete" && (in.Name != nil || in.Description != nil || in.Data != nil) {
		return projectmutations.ErrInvalid
	}
	return nil
}
