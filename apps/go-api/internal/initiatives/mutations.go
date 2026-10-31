package initiatives

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func normalize(in *Command) error {
	if in.ExpectedRevision != nil && *in.ExpectedRevision == "" || in.Action != "create" && in.ID == uuid.Nil {
		return projectmutations.ErrInvalid
	}
	switch in.Action {
	case "create", "update", "delete", "add_projects", "remove_projects", "manage_workitems":
	default:
		return projectmutations.ErrInvalid
	}
	in.Fields = maps.Clone(in.Fields)
	if in.Fields == nil {
		in.Fields = map[string]any{}
	}
	for key, value := range in.Fields {
		text, ok := value.(string)
		if !ok || strings.ContainsRune(text, 0) {
			return projectmutations.ErrInvalid
		}
		switch key {
		case "name":
			text = strings.TrimSpace(text)
			if text == "" || utf8.RuneCountInString(text) > 255 {
				return projectmutations.ErrInvalid
			}
			in.Fields[key] = text
		case "description_html":
			clean, _, err := content.ValidateHTML(text)
			if err != nil {
				return projectmutations.ErrInvalid
			}
			in.Fields[key] = clean
		case "start_date", "end_date":
			if _, err := time.Parse(time.DateOnly, text); err != nil {
				return projectmutations.ErrInvalid
			}
		case "state":
			if !slices.Contains([]string{"DRAFT", "PLANNED", "ACTIVE", "COMPLETED", "CLOSED"}, text) {
				return projectmutations.ErrInvalid
			}
		case "lead_id":
			if id, err := uuid.Parse(text); err != nil || id == uuid.Nil {
				return projectmutations.ErrInvalid
			}
		default:
			return projectmutations.ErrInvalid
		}
	}
	if in.Action == "create" && (in.ID != uuid.Nil || in.Fields["name"] == nil) {
		return projectmutations.ErrInvalid
	}
	if in.Action != "create" && in.Action != "update" && len(in.Fields) > 0 {
		return projectmutations.ErrInvalid
	}
	for _, ids := range []*[]uuid.UUID{&in.Add, &in.Remove} {
		if len(*ids) > 1000 {
			return projectmutations.ErrInvalid
		}
		*ids = slices.Clone(*ids)
		for _, id := range *ids {
			if id == uuid.Nil {
				return projectmutations.ErrInvalid
			}
		}
		slices.SortFunc(*ids, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		*ids = slices.Compact(*ids)
	}
	if (in.Action == "add_projects" || in.Action == "remove_projects" || in.Action == "manage_workitems") && len(in.Add)+len(in.Remove) == 0 {
		return projectmutations.ErrInvalid
	}
	return nil
}
func visibleItem(ctx context.Context, tx pgx.Tx, q Query, id uuid.UUID, lock bool) (uuid.UUID, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE OF i"
	}
	var project uuid.UUID
	err := tx.QueryRow(ctx, `SELECT i.project_id FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE i.id=$1 AND p.archived_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$2::uuid", "$3::uuid")+suffix, id, q.ActorID, q.WorkspaceID).Scan(&project)
	if errors.Is(err, pgx.ErrNoRows) {
		err = projectmutations.ErrNotFound
	}
	return project, err
}
func authorize(ctx context.Context, tx pgx.Tx, in Command) error {
	if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); err != nil {
		return err
	}
	if err := scope(ctx, tx, in.Query, true); err != nil {
		return err
	}
	if lead, ok := in.Fields["lead_id"].(string); ok {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `SELECT u.id FROM users u JOIN workspace_members wm ON wm.member_id=u.id WHERE u.id=$1 AND u.is_active AND wm.workspace_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,wm`, lead, in.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if err != nil {
			return err
		}
	}
	targets := append(slices.Clone(in.Add), in.Remove...)
	projects := slices.Clone(targets)
	if in.Action == "manage_workitems" {
		projects = nil
		for _, id := range targets {
			project, err := visibleItem(ctx, tx, in.Query, id, false)
			if err != nil {
				return err
			}
			projects = append(projects, project)
		}
	}
	slices.SortFunc(projects, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	projects = slices.Compact(projects)
	for _, project := range projects {
		if err := projectmutations.Authorize(ctx, tx, projectmutations.Resource{}, projectmutations.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, WorkspaceSlug: in.WorkspaceSlug, ProjectID: project}); err != nil {
			return err
		}
	}
	if in.Action == "manage_workitems" {
		slices.SortFunc(targets, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		for _, id := range slices.Compact(targets) {
			project, err := visibleItem(ctx, tx, in.Query, id, true)
			if err != nil {
				return err
			}
			if !slices.Contains(projects, project) {
				return projectmutations.ErrForbidden
			}
		}
	}
	if in.Revalidate != nil {
		return in.Revalidate(ctx)
	}
	return nil
}
func (r *Repository) Mutate(ctx context.Context, in Command) (projectmutations.Result, error) {
	if err := normalize(&in); err != nil {
		return projectmutations.Result{}, err
	}
	normalized, _ := json.Marshal(map[string]any{"id": in.ID, "fields": in.Fields, "add": in.Add, "remove": in.Remove, "expected_revision": in.ExpectedRevision})
	receipt, err := mutationreceipts.Execute(ctx, r.db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, Tool: "initiative", Action: in.Action, RequestID: in.RequestID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorize(ctx, tx, in) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var rev int64
			if err := tx.QueryRow(ctx, `SELECT revision FROM initiative_revisions WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&rev); err != nil {
				return nil, err
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != token(in.WorkspaceID, rev) {
				return nil, projectmutations.ErrRevision
			}
			id := in.ID
			var before, after map[string]any
			var err error
			if in.Action != "create" {
				before, err = record(ctx, tx, in.Query)
				if err != nil {
					return nil, err
				}
			} else {
				id = uuid.New()
			}
			switch in.Action {
			case "create", "update":
				err = writeRow(ctx, tx, in, id)
				if err == nil {
					q := in.Query
					q.ID = id
					after, err = record(ctx, tx, q)
				}
			case "add_projects", "remove_projects":
				err = manageProjects(ctx, tx, in)
			case "manage_workitems":
				err = manageItems(ctx, tx, in, before["name"].(string))
			case "delete":
				err = deleteItems(ctx, tx, in, before["name"].(string))
				if err == nil {
					_, err = tx.Exec(ctx, `DELETE FROM initiative_projects WHERE initiative_id=$1 AND workspace_id=$2`, id, in.WorkspaceID)
				}
				if err == nil {
					_, err = tx.Exec(ctx, `UPDATE initiatives SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID, in.ActorID)
				}
			}
			if err != nil {
				return nil, err
			}
			old, _ := json.Marshal(before)
			current, _ := json.Marshal(after)
			if _, err = tx.Exec(ctx, `INSERT INTO initiative_events(workspace_id,initiative_id,actor_id,action,input,before_value,after_value) VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7::jsonb)`, in.WorkspaceID, id, in.ActorID, in.Action, normalized, old, current); err != nil {
				return nil, err
			}
			if err = tx.QueryRow(ctx, `SELECT revision FROM initiative_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&rev); err != nil {
				return nil, err
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: token(in.WorkspaceID, rev), Record: current})
		},
	})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && (pg.Code == "23514" || pg.Code == "23505") {
			err = projectmutations.ErrInvalid
		}
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
func writeRow(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID) error {
	fields := maps.Clone(in.Fields)
	if html, ok := fields["description_html"].(string); ok {
		fields["description_stripped"] = content.StripTags(html)
	}
	fields["updated_by_id"] = in.ActorID.String()
	create := in.Action == "create"
	if create {
		fields["id"], fields["workspace_id"], fields["created_by_id"] = id.String(), in.WorkspaceID.String(), in.ActorID.String()
	}
	keys := slices.Sorted(maps.Keys(fields))
	cols, values := []string{}, []string{}
	for _, key := range keys {
		quoted := pgx.Identifier{key}.Sanitize()
		cols = append(cols, quoted)
		values = append(values, "x."+quoted)
	}
	raw, _ := json.Marshal(fields)
	query := `INSERT INTO initiatives (` + strings.Join(cols, ",") + `) SELECT ` + strings.Join(values, ",") + ` FROM jsonb_populate_record(NULL::initiatives,$1::jsonb) x`
	args := []any{raw}
	if !create {
		query = `UPDATE initiatives SET (` + strings.Join(cols, ",") + `)=ROW(` + strings.Join(values, ",") + `),updated_at=now() FROM jsonb_populate_record(NULL::initiatives,$1::jsonb) x WHERE initiatives.id=$2 AND initiatives.workspace_id=$3`
		args = append(args, id, in.WorkspaceID)
	}
	_, err := tx.Exec(ctx, query, args...)
	return err
}
func manageProjects(ctx context.Context, tx pgx.Tx, in Command) error {
	for _, id := range in.Add {
		query := `INSERT INTO initiative_projects(initiative_id,project_id,workspace_id,created_by_id) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`
		args := []any{in.ID, id, in.WorkspaceID, in.ActorID}
		if in.Action == "remove_projects" {
			query = `DELETE FROM initiative_projects WHERE initiative_id=$1 AND project_id=$2 AND workspace_id=$3`
			args = args[:3]
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}
func activity(ctx context.Context, tx pgx.Tx, in Command, item, project uuid.UUID, name, verb string) error {
	return (projectmutations.Effects{Tx: tx}).RelationActivity(ctx, projectmutations.RelationActivity{
		WorkspaceID: in.WorkspaceID, ProjectID: project, ItemID: item, ActorID: in.ActorID,
		RelatedID: in.ID, Label: "initiative", Identifier: name, Verb: verb,
	})
}
func manageItems(ctx context.Context, tx pgx.Tx, in Command, name string) error {
	for _, id := range in.Remove {
		var project uuid.UUID
		err := tx.QueryRow(ctx, `DELETE FROM initiative_work_items WHERE initiative_id=$1 AND issue_id=$2 AND workspace_id=$3 RETURNING project_id`, in.ID, id, in.WorkspaceID).Scan(&project)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if err = activity(ctx, tx, in, id, project, name, "deleted"); err != nil {
			return err
		}
	}
	for _, id := range in.Add {
		project, err := visibleItem(ctx, tx, in.Query, id, true)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO initiative_work_items(initiative_id,issue_id,project_id,workspace_id,created_by_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, in.ID, id, project, in.WorkspaceID, in.ActorID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			if err = activity(ctx, tx, in, id, project, name, "created"); err != nil {
				return err
			}
		}
	}
	return nil
}
func deleteItems(ctx context.Context, tx pgx.Tx, in Command, name string) error {
	rows, err := tx.Query(ctx, `DELETE FROM initiative_work_items WHERE initiative_id=$1 AND workspace_id=$2 RETURNING issue_id,project_id`, in.ID, in.WorkspaceID)
	if err != nil {
		return err
	}
	pairs := [][2]uuid.UUID{}
	for rows.Next() {
		var pair [2]uuid.UUID
		if err = rows.Scan(&pair[0], &pair[1]); err != nil {
			break
		}
		pairs = append(pairs, pair)
	}
	rows.Close()
	if err != nil {
		return err
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for _, pair := range pairs {
		if err = activity(ctx, tx, in, pair[0], pair[1], name, "deleted"); err != nil {
			return err
		}
	}
	return nil
}
