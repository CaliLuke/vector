package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/go-plane/internal/deletion"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const deletionBatchSize = 250

type DeletionInput struct {
	EventID string `json:"event_id"`
	App     string `json:"app"`
	Model   string `json:"model"`
	ID      string `json:"id"`
}
type deletionPlanInput struct {
	Root     DeletionInput `json:"root"`
	Relation int           `json:"relation"`
	AfterID  string        `json:"after_id"`
}
type deletionPlan struct {
	Input deletionPlanInput `json:"input"`
	IDs   []string          `json:"ids"`
	More  bool              `json:"more"`
}
type hardDeletionInput struct {
	Cutoff time.Time `json:"cutoff"`
	Model  int       `json:"model"`
}
type deletionStore interface {
	MarkRoot(context.Context, DeletionInput) (bool, error)
	Plan(context.Context, deletionPlanInput) (deletionPlan, error)
	Apply(context.Context, deletionPlan) ([]DeletionInput, error)
	HardDelete(context.Context, hardDeletionInput) (int, error)
	Policy() deletion.Policy
}
type PostgresDeletionStore struct {
	pool   *pgxpool.Pool
	policy deletion.Policy
}

func NewPostgresDeletionStore(pool *pgxpool.Pool) (*PostgresDeletionStore, error) {
	if pool == nil {
		return nil, errors.New("deletion PostgreSQL pool is required")
	}
	policy, err := deletion.Load()
	if err != nil {
		return nil, err
	}
	return &PostgresDeletionStore{pool: pool, policy: policy}, nil
}
func (s *PostgresDeletionStore) Policy() deletion.Policy { return s.policy }
func (s *PostgresDeletionStore) validate(in DeletionInput) (deletion.Model, error) {
	if _, err := uuid.Parse(in.EventID); err != nil {
		return deletion.Model{}, fmt.Errorf("deletion event ID: %w", err)
	}
	if _, err := uuid.Parse(in.ID); err != nil {
		return deletion.Model{}, fmt.Errorf("deletion object ID: %w", err)
	}
	m, err := s.policy.Lookup(in.App, in.Model)
	if err != nil {
		return m, err
	}
	if !m.SoftDelete {
		return m, errors.New("model does not support soft deletion")
	}
	return m, nil
}
func deletionTable(m deletion.Model) string { return pgx.Identifier{"public", m.Table}.Sanitize() }
func (s *PostgresDeletionStore) MarkRoot(ctx context.Context, in DeletionInput) (bool, error) {
	m, err := s.validate(in)
	if err != nil {
		return false, err
	}
	updated := ""
	if m.UpdatedAt {
		updated = ",updated_at=CASE WHEN deleted_at IS NULL THEN clock_timestamp() ELSE updated_at END"
	}
	tag, err := s.pool.Exec(ctx, "UPDATE "+deletionTable(m)+" SET deleted_at=COALESCE(deleted_at,clock_timestamp())"+updated+" WHERE "+pgx.Identifier{m.PK}.Sanitize()+"=$1::uuid", in.ID)
	return tag.RowsAffected() > 0, err
}

type reverseDeletionRelation struct {
	model    deletion.Model
	relation deletion.Relation
}

func deletionRelations(p deletion.Policy, parent deletion.Model) []reverseDeletionRelation {
	var result []reverseDeletionRelation
	for _, child := range p.Models {
		for _, r := range child.Relations {
			if r.TargetTable == parent.Table && r.SoftReverse && r.OnDelete != "DO_NOTHING" && (r.OnDelete == "SET_NULL" || child.SoftDelete) {
				result = append(result, reverseDeletionRelation{child, r})
			}
		}
	}
	return result
}
func (s *PostgresDeletionStore) relation(in deletionPlanInput) (reverseDeletionRelation, error) {
	m, err := s.validate(in.Root)
	if err != nil {
		return reverseDeletionRelation{}, err
	}
	rs := deletionRelations(s.policy, m)
	if in.Relation < 0 || in.Relation >= len(rs) {
		return reverseDeletionRelation{}, errors.New("invalid deletion relation")
	}
	return rs[in.Relation], nil
}

// Plan is a read-only activity: its bounded result is journaled before Apply
// changes visibility. An activity retry after a committed update cannot lose
// the IDs whose child workflows still need admission.
func (s *PostgresDeletionStore) Plan(ctx context.Context, in deletionPlanInput) (deletionPlan, error) {
	r, err := s.relation(in)
	if err != nil {
		return deletionPlan{}, err
	}
	result := deletionPlan{Input: in, IDs: []string{}}
	predicate := ""
	if r.model.SoftDelete && (r.relation.OnDelete == "CASCADE" || !r.relation.OneToOne) {
		predicate = " AND deleted_at IS NULL"
	}
	query := "SELECT " + pgx.Identifier{r.model.PK}.Sanitize() + "::text FROM " + deletionTable(r.model) + " WHERE " + pgx.Identifier{r.relation.Column}.Sanitize() + "=$1::uuid AND " + pgx.Identifier{r.model.PK}.Sanitize() + "::text>$2" + predicate + " ORDER BY " + pgx.Identifier{r.model.PK}.Sanitize() + "::text LIMIT $3"
	rows, err := s.pool.Query(ctx, query, in.Root.ID, in.AfterID, deletionBatchSize)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return result, err
		}
		result.IDs = append(result.IDs, id)
	}
	result.More = len(result.IDs) == deletionBatchSize
	return result, rows.Err()
}
func (s *PostgresDeletionStore) Apply(ctx context.Context, plan deletionPlan) ([]DeletionInput, error) {
	r, err := s.relation(plan.Input)
	if err != nil {
		return nil, err
	}
	children := []DeletionInput{}
	if len(plan.IDs) == 0 {
		return children, nil
	}
	if len(plan.IDs) > deletionBatchSize {
		return nil, errors.New("deletion plan exceeds batch limit")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	for _, id := range plan.IDs {
		if _, err := uuid.Parse(id); err != nil {
			return nil, err
		}
		where := " WHERE " + pgx.Identifier{r.model.PK}.Sanitize() + "=$1::uuid AND " + pgx.Identifier{r.relation.Column}.Sanitize() + "=$2::uuid"
		if r.relation.OnDelete == "SET_NULL" {
			_, err = tx.Exec(ctx, "UPDATE "+deletionTable(r.model)+" SET "+pgx.Identifier{r.relation.Column}.Sanitize()+"=NULL"+where, id, plan.Input.Root.ID)
		} else {
			updated := ""
			if r.model.UpdatedAt {
				updated = ",updated_at=CASE WHEN deleted_at IS NULL THEN clock_timestamp() ELSE updated_at END"
			}
			_, err = tx.Exec(ctx, "UPDATE "+deletionTable(r.model)+" SET deleted_at=COALESCE(deleted_at,clock_timestamp())"+updated+where, id, plan.Input.Root.ID)
			children = append(children, DeletionInput{EventID: plan.Input.Root.EventID, App: r.model.App, Model: r.model.Name, ID: id})
		}
		if err != nil {
			return nil, err
		}
	}
	return children, tx.Commit(ctx)
}
func (s *PostgresDeletionStore) HardDelete(ctx context.Context, in hardDeletionInput) (int, error) {
	if in.Cutoff.IsZero() || in.Model < 0 || in.Model >= len(s.policy.Models) {
		return 0, errors.New("invalid hard deletion batch")
	}
	m := s.policy.Models[in.Model]
	if !m.SoftDelete {
		return 0, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	// Never physically delete against a source schema whose FK cascades still
	// exist only in Python. Bootstrap validates and installs this migration.
	var ready bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.plane_schema_migrations WHERE version>=2 AND NOT dirty)`).Scan(&ready); err != nil {
		return 0, err
	}
	if !ready {
		return 0, errors.New("native deletion schema migration is required")
	}
	pk := pgx.Identifier{m.PK}.Sanitize()
	table := deletionTable(m)
	tag, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE "+pk+" IN (SELECT "+pk+" FROM "+table+" WHERE deleted_at<$1 ORDER BY "+pk+" LIMIT $2 FOR UPDATE SKIP LOCKED)", in.Cutoff, deletionBatchSize)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), tx.Commit(ctx)
}
