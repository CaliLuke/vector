package customers

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
	if in.CustomerID != uuid.Nil {
		if err := customerExists(ctx, tx, in.Query); err != nil {
			return err
		}
	}
	if in.Tool == "customer" && in.RequestID != uuid.Nil {
		if err := requestExists(ctx, tx, in.Query); err != nil {
			return err
		}
	}
	targets := append(slices.Clone(in.Add), in.Remove...)
	projects := []uuid.UUID{}
	for _, id := range targets {
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
	if in.Action == "set_values" {
		for _, id := range slices.Sorted(maps.Keys(in.Values)) {
			q := in.Query
			q.Tool = "customer_property"
			property, err := record(ctx, tx, q, uuid.MustParse(id))
			if err != nil {
				return err
			}
			if err = validateValues(ctx, tx, q, property, in.Values[id], false); err != nil {
				return err
			}
		}
	}
	if in.Revalidate != nil {
		return in.Revalidate(ctx)
	}
	return nil
}
func activity(ctx context.Context, tx pgx.Tx, in Command, item, project uuid.UUID, name, verb string) error {
	return (projectmutations.Effects{Tx: tx}).RelationActivity(ctx, projectmutations.RelationActivity{
		WorkspaceID: in.WorkspaceID, ProjectID: project, ItemID: item, ActorID: in.ActorID,
		RelatedID: in.CustomerID, Label: "customer", Identifier: name, Verb: verb,
	})
}
func manageItems(ctx context.Context, tx pgx.Tx, in Command, name string) error {
	for _, id := range in.Add {
		project, err := visibleItem(ctx, tx, in.Query, id, true)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO customer_work_items(workspace_id,customer_id,request_id,issue_id,project_id,created_by_id) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, in.WorkspaceID, in.CustomerID, optionalID(in.RequestID), id, project, in.ActorID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			if err = activity(ctx, tx, in, id, project, name, "created"); err != nil {
				return err
			}
		}
	}
	for _, id := range in.Remove {
		pairs, err := unlinkItems(ctx, tx, in, optionalID(id))
		if err != nil {
			return err
		}
		if len(pairs) == 0 {
			return projectmutations.ErrNotFound
		}
		for _, pair := range pairs {
			if err = activity(ctx, tx, in, pair[0], pair[1], name, "deleted"); err != nil {
				return err
			}
		}
	}
	return nil
}
func unlinkItems(ctx context.Context, tx pgx.Tx, in Command, item any) ([][2]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `DELETE FROM customer_work_items WHERE customer_id=$1 AND workspace_id=$2 AND ($3::uuid IS NULL OR request_id=$3) AND ($4::uuid IS NULL OR issue_id=$4) RETURNING issue_id,project_id`, in.CustomerID, in.WorkspaceID, optionalID(in.RequestID), item)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return pairs, rows.Err()
}
