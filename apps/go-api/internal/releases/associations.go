package releases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func authorize(ctx context.Context, tx pgx.Tx, in Command) error {
	if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); err != nil {
		return err
	}
	if err := workspaceScope(ctx, tx, in.Query, true); err != nil {
		return err
	}
	// These checks run before receipts so stale selection permissions cannot be
	// used to replay a response containing a newly inaccessible reference.
	if lead, ok := in.Fields["lead_id"].(string); ok {
		var id string
		err := tx.QueryRow(ctx, `SELECT u.id::text FROM users u JOIN workspace_members wm ON wm.member_id=u.id WHERE u.id=$1 AND u.is_active AND wm.workspace_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,wm`, lead, in.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if err != nil {
			return err
		}
	}
	if tag, ok := in.Fields["tag_id"].(string); ok {
		var id string
		err := tx.QueryRow(ctx, `SELECT id::text FROM release_tags WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR SHARE`, tag, in.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if err != nil {
			return err
		}
	}
	if in.Action == "manage_workitems" {
		ids := append(slices.Clone(in.Add), in.Remove...)
		slices.SortFunc(ids, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		ids = slices.Compact(ids)
		projects := []uuid.UUID{}
		for _, id := range ids {
			project, err := visibleItem(ctx, tx, in.Query, id, false)
			if err != nil {
				return err
			}
			projects = append(projects, project)
		}
		slices.SortFunc(projects, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		projects = slices.Compact(projects)
		for _, project := range projects {
			if err := projectmutations.Authorize(ctx, tx, projectmutations.Resource{}, projectmutations.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, WorkspaceSlug: in.WorkspaceSlug, ProjectID: project}); err != nil {
				return err
			}
		}
		for _, id := range ids {
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
func manageLabels(ctx context.Context, tx pgx.Tx, in Command) (bool, error) {
	for _, id := range in.Add {
		var found uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM release_labels WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR SHARE`, id, in.WorkspaceID).Scan(&found)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, projectmutations.ErrNotFound
		}
		if err != nil {
			return false, err
		}
	}
	changed := false
	for _, id := range in.Add {
		sql := `INSERT INTO release_item_labels(release_id,label_id,workspace_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`
		if in.Action == "detach" {
			sql = `DELETE FROM release_item_labels WHERE release_id=$1 AND label_id=$2 AND workspace_id=$3`
		}
		command, err := tx.Exec(ctx, sql, in.ReleaseID, id, in.WorkspaceID)
		if err != nil {
			return false, err
		}
		changed = changed || command.RowsAffected() > 0
	}
	return changed, nil
}
func manageItems(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID, name string) (bool, error) {
	changed := false
	for _, item := range in.Add {
		// Removal wins without producing a spurious add/remove activity pair.
		if slices.Contains(in.Remove, item) {
			continue
		}
		project, err := visibleItem(ctx, tx, in.Query, item, true)
		if err != nil {
			return false, err
		}
		command, err := tx.Exec(ctx, `INSERT INTO release_work_items(release_id,issue_id,project_id,workspace_id,created_by_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, id, item, project, in.WorkspaceID, in.ActorID)
		if err != nil {
			return false, err
		}
		if command.RowsAffected() > 0 {
			changed = true
			if err = associationActivity(ctx, tx, in, id, item, project, name, "created"); err != nil {
				return false, err
			}
		}
	}
	for _, item := range in.Remove {
		var project uuid.UUID
		err := tx.QueryRow(ctx, `DELETE FROM release_work_items WHERE release_id=$1 AND issue_id=$2 AND workspace_id=$3 RETURNING project_id`, id, item, in.WorkspaceID).Scan(&project)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return false, err
		}
		changed = true
		if err = associationActivity(ctx, tx, in, id, item, project, name, "deleted"); err != nil {
			return false, err
		}
	}
	return changed, nil
}
func removeAllItems(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID, name string) (bool, error) {
	rows, err := tx.Query(ctx, `DELETE FROM release_work_items WHERE release_id=$1 AND workspace_id=$2 RETURNING issue_id,project_id`, id, in.WorkspaceID)
	if err != nil {
		return false, err
	}
	var pairs [][2]uuid.UUID
	for rows.Next() {
		var pair [2]uuid.UUID
		if err = rows.Scan(&pair[0], &pair[1]); err != nil {
			rows.Close()
			return false, err
		}
		pairs = append(pairs, pair)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return false, err
	}
	for _, pair := range pairs {
		if err = associationActivity(ctx, tx, in, id, pair[0], pair[1], name, "deleted"); err != nil {
			return false, err
		}
	}
	return len(pairs) > 0, nil
}
func associationActivity(ctx context.Context, tx pgx.Tx, in Command, release, item, project uuid.UUID, name, verb string) error {
	// The native relation activity worker accepts an immutable related-resource
	// identity and label. No post-commit catalogue lookup can change this history.
	data := map[string]any{"type": "workitem_relation.activity." + verb, "workspace_id": in.WorkspaceID.String(), "project_id": project.String(), "issue_id": item.String(), "actor_id": in.ActorID.String(), "epoch": time.Now().Unix(), "notification": false, "requested_data": map[string]any{"related_id": release.String(), "label": "release", "identifier": name}}
	return (projectmutations.Effects{Tx: tx}).Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", nil, data)
}
