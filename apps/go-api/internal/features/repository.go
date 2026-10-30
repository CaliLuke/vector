package features

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db mutationreceipts.Database }

func NewRepository(db mutationreceipts.Database) *Repository { return &Repository{db: db} }

type Query struct {
	ActorID, WorkspaceID, ProjectID uuid.UUID
	WorkspaceSlug                   string
	IncludeRevision                 bool
}
type Command struct {
	Query
	Values           map[string]bool
	RequestID        *uuid.UUID
	ExpectedRevision *string
	Revalidate       func(context.Context) error
}

// Capabilities describe feature actions available in the selected workspace or
// any current membership when discovery has no workspace binding.
type Capabilities struct {
	Initiatives, ManageInitiatives, Customers, ManageCustomers bool
}

func (r *Repository) WorkspaceVisibility(ctx context.Context, q Query) (_ Capabilities, resultErr error) {
	var out Capabilities
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	err = tx.QueryRow(ctx, `SELECT COALESCE(bool_or(f.initiatives),false),COALESCE(bool_or(f.initiatives AND wm.role=20),false),
 COALESCE(bool_or(f.customers),false),COALESCE(bool_or(f.customers AND wm.role=20),false)
 FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id
 JOIN workspace_features f ON f.workspace_id=w.id
 WHERE u.id=$1 AND u.is_active AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) AND w.deleted_at IS NULL
 AND (($2::uuid='00000000-0000-0000-0000-000000000000'::uuid AND $3='') OR (w.id=$2 AND w.slug=$3))`, q.ActorID, q.WorkspaceID, q.WorkspaceSlug).Scan(&out.Initiatives, &out.ManageInitiatives, &out.Customers, &out.ManageCustomers)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

var workspaceFields = map[string]string{"project_grouping": "project_grouping", "initiatives": "initiatives", "teams": "teams", "customers": "customers", "wiki": "wiki", "pi": "pi"}
var projectFields = map[string]string{"modules": "module_view", "cycles": "cycle_view", "views": "issue_views_view", "pages": "page_view", "intakes": "intake_view", "work_item_types": "is_issue_type_enabled", "epics": "epics_enabled", "parallel_cycles": "parallel_cycles_enabled", "project_updates": "project_updates_enabled", "workflows": "workflows_enabled"}

func scope(ctx context.Context, tx pgx.Tx, q Query, write bool) error {
	role, err := projectmutations.WorkspaceRole(ctx, tx, q.ActorID, q.WorkspaceID, q.WorkspaceSlug, write)
	if err != nil {
		return err
	}
	if q.ProjectID != uuid.Nil {
		suffix := ""
		if write {
			// Archived projects retain readable settings, but cannot be changed.
			suffix = " AND p.archived_at IS NULL FOR SHARE OF p,pm"
		}
		err = tx.QueryRow(ctx, `SELECT pm.role FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20)`+suffix, q.ProjectID, q.WorkspaceID, q.ActorID).Scan(&role)
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
	return nil
}
func token(q Query, rev int64) string {
	return fmt.Sprintf("feature:%s:%s:%d", q.WorkspaceID, q.ProjectID, rev)
}
func read(ctx context.Context, tx pgx.Tx, q Query) (map[string]any, error) {
	fields := workspaceFields
	query := `SELECT to_jsonb(t) FROM workspace_features t WHERE workspace_id=$1`
	args := []any{q.WorkspaceID}
	if q.ProjectID != uuid.Nil {
		fields = projectFields
		query = `SELECT to_jsonb(t) FROM projects t WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL`
		args = append(args, q.ProjectID)
	}
	var raw []byte
	if err := tx.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return nil, err
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for key, column := range fields {
		out[key] = row[column]
	}
	if q.ProjectID == uuid.Nil {
		out["work_item_types"], out["releases"], out["states_owned_by_workspace"] = true, true, false
	}
	return out, nil
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
	row, err := read(ctx, tx, q)
	if err != nil {
		return nil, err
	}
	if q.IncludeRevision {
		var rev int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM workspace_features WHERE workspace_id=$1`, q.WorkspaceID).Scan(&rev); err != nil {
			return nil, err
		}
		row["revision"] = token(q, rev)
	}
	raw, err := json.Marshal(row)
	if err != nil {
		return nil, err
	}
	return raw, tx.Commit(ctx)
}
func (r *Repository) Mutate(ctx context.Context, in Command) (projectmutations.Result, error) {
	fields, table, tool := workspaceFields, "workspace_features", "workspace"
	id := in.WorkspaceID
	var project *uuid.UUID
	if in.ProjectID != uuid.Nil {
		fields, table, tool = projectFields, "projects", "project"
		id = in.ProjectID
		project = &in.ProjectID
	}
	if in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	in.Values = maps.Clone(in.Values)
	for key := range in.Values {
		if fields[key] == "" {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	}
	normalized, _ := json.Marshal(struct {
		Values   map[string]bool
		Revision *string
	}{in.Values, in.ExpectedRevision})
	receipt, err := mutationreceipts.Execute(ctx, r.db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, ProjectID: project, Tool: tool, Action: "update_features", RequestID: in.RequestID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); err != nil {
				return err
			}
			if err := scope(ctx, tx, in.Query, true); err != nil {
				return err
			}
			if in.Revalidate != nil {
				return in.Revalidate(ctx)
			}
			return nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var revision int64
			if err := tx.QueryRow(ctx, `SELECT revision FROM workspace_features WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&revision); err != nil {
				return nil, err
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != token(in.Query, revision) {
				return nil, projectmutations.ErrRevision
			}
			before, err := read(ctx, tx, in.Query)
			if err != nil {
				return nil, err
			}
			keys := slices.Sorted(maps.Keys(in.Values))
			set := []string{}
			args := []any{in.WorkspaceID}
			for _, key := range keys {
				args = append(args, in.Values[key])
				set = append(set, pgx.Identifier{fields[key]}.Sanitize()+fmt.Sprintf("=$%d", len(args)))
			}
			if len(keys) > 0 {
				where := "workspace_id=$1"
				if project != nil {
					args = append(args, *project)
					where += fmt.Sprintf(" AND id=$%d", len(args))
					args = append(args, in.ActorID)
					set = append(set, "updated_at=now()", fmt.Sprintf("updated_by_id=$%d", len(args)))
				}
				if _, err = tx.Exec(ctx, `UPDATE `+table+` SET `+strings.Join(set, ",")+` WHERE `+where, args...); err != nil {
					return nil, err
				}
				if project != nil && in.Values["intakes"] {
					if err = workspaceprojectreads.EnsureProjectIntake(ctx, tx, project.String(), in.ActorID.String()); err != nil {
						return nil, err
					}
				}
			}
			after, err := read(ctx, tx, in.Query)
			if err != nil {
				return nil, err
			}
			old, _ := json.Marshal(before)
			current, _ := json.Marshal(after)
			if _, err = tx.Exec(ctx, `INSERT INTO feature_events(workspace_id,project_id,actor_id,before_value,after_value) VALUES($1,$2,$3,$4::jsonb,$5::jsonb)`, in.WorkspaceID, project, in.ActorID, old, current); err != nil {
				return nil, err
			}
			if err = tx.QueryRow(ctx, `SELECT revision FROM workspace_features WHERE workspace_id=$1`, in.WorkspaceID).Scan(&revision); err != nil {
				return nil, err
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: token(in.Query, revision), Record: current})
		},
	})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if in.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = in.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}
