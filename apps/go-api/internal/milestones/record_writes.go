package milestones

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) Create(ctx context.Context, opts WriteOptions, fields Fields) (MutationResult, error) {
	if opts.ExpectedRevision != nil {
		return MutationResult{}, ErrInvalidInput
	}
	values, err := normalizedFields(fields, true)
	if err != nil {
		return MutationResult{}, err
	}
	return r.write(ctx, opts, "create", map[string]any{"fields": values}, func(tx pgx.Tx) (MutationResult, error) {
		id := uuid.New()
		_, err := tx.Exec(ctx, `INSERT INTO milestones(id,project_id,workspace_id,title,target_date,external_source,external_id,created_by_id,updated_by_id,created_at,updated_at)
 VALUES($1,$2,$3,$4,$5::date,$6,$7,$8,$8,statement_timestamp(),statement_timestamp())`, id, opts.ProjectID, opts.WorkspaceID, values["title"], values["target_date"], values["external_source"], values["external_id"], opts.ActorID)
		if err != nil {
			return MutationResult{}, err
		}
		record, err := retrieve(ctx, tx, opts.Scope, id)
		if err != nil {
			return MutationResult{}, err
		}
		if err = admitRecordEffect(ctx, tx, opts, id, values, nil); err != nil {
			return MutationResult{}, err
		}
		return mutationResult(ctx, tx, id, &record)
	})
}

func (r *Repository) Update(ctx context.Context, opts WriteOptions, id uuid.UUID, fields Fields) (MutationResult, error) {
	if id == uuid.Nil {
		return MutationResult{}, ErrInvalidInput
	}
	values, err := normalizedFields(fields, false)
	if err != nil {
		return MutationResult{}, err
	}
	return r.write(ctx, opts, "update", map[string]any{"milestone_id": id.String(), "fields": values}, func(tx pgx.Tx) (MutationResult, error) {
		if err := lockMilestone(ctx, tx, opts, id); err != nil {
			return MutationResult{}, err
		}
		if err := opts.revalidate(ctx); err != nil {
			return MutationResult{}, err
		}
		before, err := retrieve(ctx, tx, opts.Scope, id)
		if err != nil {
			return MutationResult{}, err
		}
		previous := recordFields(before)
		changed := make(map[string]any)
		args := []any{id, opts.ActorID}
		set := []string{}
		// Field names are authored SQL. Only normalized values become parameters.
		for _, field := range []string{"title", "target_date", "external_source", "external_id"} {
			value, present := values[field]
			if !present || value == previous[field] {
				continue
			}
			changed[field] = value
			args = append(args, value)
			set = append(set, fmt.Sprintf("%s=$%d", field, len(args)))
		}
		if len(set) == 0 {
			return mutationResult(ctx, tx, id, &before)
		}
		set = append(set, "updated_by_id=$2", "updated_at=statement_timestamp()")
		if _, err = tx.Exec(ctx, `UPDATE milestones SET `+strings.Join(set, ",")+` WHERE id=$1`, args...); err != nil {
			return MutationResult{}, err
		}
		record, err := retrieve(ctx, tx, opts.Scope, id)
		if err != nil {
			return MutationResult{}, err
		}
		if err = admitRecordEffect(ctx, tx, opts, id, changed, previous); err != nil {
			return MutationResult{}, err
		}
		return mutationResult(ctx, tx, id, &record)
	})
}
