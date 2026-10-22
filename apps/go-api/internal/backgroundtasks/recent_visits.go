package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	recentVisitWorkflowName    = "plane.activity.record_recent_visit"
	recentVisitActivityName    = "plane.activity.write_recent_visit"
	recentVisitQueue           = "plane-background"
	recentVisitActivityTimeout = 15 * time.Second
	recentVisitLimit           = 20
)

var ErrPermanentRecentVisit = errors.New("permanent recent visit task failure")

type RecentVisitInput struct {
	Slug             string  `json:"slug"`
	EntityName       string  `json:"entity_name"`
	EntityIdentifier *string `json:"entity_identifier"`
	UserID           string  `json:"user_id"`
	ProjectID        string  `json:"project_id"`
}

type recentVisitStore interface {
	RecordRecentVisit(context.Context, RecentVisitInput) error
}

type RecentVisitDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[RecentVisitInput, struct{}]
}

// RecentVisitAdmission adapts read-side visit effects to durable execution.
// Each distinct request gets a fresh operation ID; dispatcher reconciliation
// reuses that ID only when one admission result is uncertain.
type RecentVisitAdmission struct{ dispatcher *RecentVisitDispatcher }

func NewRecentVisitAdmission(dispatcher *RecentVisitDispatcher) (*RecentVisitAdmission, error) {
	if dispatcher == nil {
		return nil, errors.New("recent visit dispatcher is required")
	}
	return &RecentVisitAdmission{dispatcher: dispatcher}, nil
}

func (admission *RecentVisitAdmission) AdmitRecentVisit(ctx context.Context, eventID, slug, entityName, entityIdentifier, userID, projectID string) error {
	if admission == nil || admission.dispatcher == nil {
		return errors.New("native recent visit admission is not configured")
	}
	var identifier *string
	if entityIdentifier != "" {
		identifier = &entityIdentifier
	}
	input := RecentVisitInput{Slug: slug, EntityName: entityName, EntityIdentifier: identifier, UserID: userID, ProjectID: projectID}
	if _, err := admission.dispatcher.Admit(ctx, eventID, input); err != nil {
		return fmt.Errorf("admit recent visit workflow: %w", err)
	}
	return nil
}

func RegisterRecentVisitWorkflow(app *selvedge.App, store recentVisitStore, policy *retry.Policy) (*RecentVisitDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and recent visit store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentRecentVisit)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[RecentVisitInput, struct{}]{Name: recentVisitActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input RecentVisitInput) (struct{}, error) {
		if err := validateRecentVisitInput(input); err != nil {
			return struct{}{}, err
		}
		if err := store.RecordRecentVisit(ctx, input); err != nil {
			return struct{}{}, classifyRecentVisitError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: recentVisitQueue, StartToCloseTimeout: recentVisitActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register recent visit activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(recentVisitWorkflowName, func(ctx *selvedge.WorkflowContext, input RecentVisitInput) (struct{}, error) {
		if err := validateRecentVisitInput(input); err != nil {
			return struct{}{}, err
		}
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &RecentVisitDispatcher{app: app, workflow: workflow}, nil
}

func (dispatcher *RecentVisitDispatcher) Admit(ctx context.Context, eventID string, input RecentVisitInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable recent visit dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("recent visit event ID must be a UUID: %w", err)
	}
	if err := validateRecentVisitInput(input); err != nil {
		return "", err
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "recent-visit:"+eventID, input)
}

func validateRecentVisitInput(input RecentVisitInput) error {
	if input.Slug == "" || input.EntityName == "" {
		return fmt.Errorf("%w: workspace slug and entity name are required", ErrPermanentRecentVisit)
	}
	for name, value := range map[string]string{"user ID": input.UserID} {
		if _, err := uuid.Parse(value); err != nil {
			return fmt.Errorf("%w: invalid %s", ErrPermanentRecentVisit, name)
		}
	}
	if input.EntityIdentifier != nil {
		if _, err := uuid.Parse(*input.EntityIdentifier); err != nil {
			return fmt.Errorf("%w: invalid entity identifier", ErrPermanentRecentVisit)
		}
	}
	if input.ProjectID != "" {
		if _, err := uuid.Parse(input.ProjectID); err != nil {
			return fmt.Errorf("%w: invalid project ID", ErrPermanentRecentVisit)
		}
	}
	return nil
}

type PostgresRecentVisitStore struct{ pool *pgxpool.Pool }

func NewPostgresRecentVisitStore(pool *pgxpool.Pool) (*PostgresRecentVisitStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresRecentVisitStore{pool: pool}, nil
}

func (store *PostgresRecentVisitStore) RecordRecentVisit(ctx context.Context, input RecentVisitInput) error {
	if err := validateRecentVisitInput(input); err != nil {
		return err
	}
	userID, _ := uuid.Parse(input.UserID)
	var entityID *uuid.UUID
	if input.EntityIdentifier != nil {
		parsed, _ := uuid.Parse(*input.EntityIdentifier)
		entityID = &parsed
	}
	var projectID *uuid.UUID
	if input.ProjectID != "" {
		parsed, _ := uuid.Parse(input.ProjectID)
		projectID = &parsed
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin recent visit transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var workspaceID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, input.Slug).Scan(&workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("load workspace for recent visit: %w", err)
	}
	var lockedUserID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&lockedUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("lock recent visit owner: %w", err)
	}
	var visitID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM user_recent_visits WHERE entity_name=$1 AND entity_identifier IS NOT DISTINCT FROM $2 AND user_id=$3 AND project_id IS NOT DISTINCT FROM $4 AND workspace_id=$5 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, input.EntityName, entityID, userID, projectID, workspaceID).Scan(&visitID)
	if err == nil {
		now := time.Now().UTC()
		if _, err := tx.Exec(ctx, `UPDATE user_recent_visits SET visited_at=$2 WHERE id=$1`, visitID, now); err != nil {
			return fmt.Errorf("update recent visit timestamp: %w", err)
		}
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("find existing recent visit: %w", err)
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM user_recent_visits WHERE user_id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, userID, workspaceID).Scan(&count); err != nil {
		return fmt.Errorf("count workspace recent visits: %w", err)
	}
	now := time.Now().UTC()
	if count == recentVisitLimit {
		var oldestID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM user_recent_visits WHERE user_id=$1 AND workspace_id=$2 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT 1 FOR UPDATE`, userID, workspaceID).Scan(&oldestID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("find oldest recent visit: %w", err)
		}
		if err == nil {
			if _, err := tx.Exec(ctx, `UPDATE user_recent_visits SET deleted_at=$2 WHERE id=$1`, oldestID, now); err != nil {
				return fmt.Errorf("evict oldest recent visit: %w", err)
			}
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO user_recent_visits (id,workspace_id,project_id,entity_name,entity_identifier,user_id,created_by_id,updated_by_id,created_at,updated_at,visited_at,deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$6,$6,$7,$7,$7,NULL)`, uuid.New(), workspaceID, projectID, input.EntityName, entityID, userID, now)
	if err != nil {
		return fmt.Errorf("insert recent visit: %w", err)
	}
	return tx.Commit(ctx)
}

func classifyRecentVisitError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return fmt.Errorf("%w: invalid recent visit reference", ErrPermanentRecentVisit)
	}
	return err
}
