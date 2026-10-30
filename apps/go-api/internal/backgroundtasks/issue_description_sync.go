package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	IssueDescriptionSyncWorkflowName = "plane.issues.sync_description_versions"
	issueDescriptionSyncActivity     = "plane.issues.sync_description_version_batch"
	issueDescriptionSyncQueue        = "plane-background"
)

type IssueDescriptionSyncInput struct {
	RunID     string        `json:"run_id"`
	Offset    int           `json:"offset"`
	BatchSize int           `json:"batch_size"`
	Countdown time.Duration `json:"countdown"`
}

type IssueDescriptionSyncBatchInput struct {
	RunID     string `json:"run_id"`
	Offset    int    `json:"offset"`
	BatchSize int    `json:"batch_size"`
}

type IssueDescriptionSyncBatchOutput struct {
	Total    int `json:"total"`
	End      int `json:"end"`
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

type IssueDescriptionSyncOutput struct {
	Batches  int `json:"batches"`
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

type issueDescriptionSyncStore interface {
	SyncIssueDescriptionVersionBatch(context.Context, IssueDescriptionSyncBatchInput) (IssueDescriptionSyncBatchOutput, error)
}

type IssueDescriptionSyncDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[IssueDescriptionSyncInput, IssueDescriptionSyncOutput]
}

func RegisterIssueDescriptionSyncWorkflow(app *selvedge.App, store issueDescriptionSyncStore, policy *retry.Policy) (*IssueDescriptionSyncDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("workflow app and issue description sync store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentVersionHistory)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[IssueDescriptionSyncBatchInput, IssueDescriptionSyncBatchOutput]{Name: issueDescriptionSyncActivity}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input IssueDescriptionSyncBatchInput) (IssueDescriptionSyncBatchOutput, error) {
		if _, err := uuid.Parse(input.RunID); err != nil || input.Offset < 0 || input.BatchSize < 1 || input.BatchSize > issueSyncBatchLimit {
			return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("%w: invalid issue description sync batch", ErrPermanentVersionHistory)
		}
		result, err := store.SyncIssueDescriptionVersionBatch(ctx, input)
		if err != nil {
			return IssueDescriptionSyncBatchOutput{}, classifyVersionHistoryError(err)
		}
		return result, nil
	}, selvedge.ActivityOptions{Queue: issueDescriptionSyncQueue, StartToCloseTimeout: 2 * time.Minute, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue description sync activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(IssueDescriptionSyncWorkflowName, func(ctx *selvedge.WorkflowContext, input IssueDescriptionSyncInput) (IssueDescriptionSyncOutput, error) {
		if _, err := uuid.Parse(input.RunID); err != nil || input.Offset < 0 || input.BatchSize < 1 || input.BatchSize > issueSyncBatchLimit || input.Countdown < 0 {
			return IssueDescriptionSyncOutput{}, fmt.Errorf("%w: invalid issue description sync workflow input", ErrPermanentVersionHistory)
		}
		var result IssueDescriptionSyncOutput
		offset := input.Offset
		for {
			batch, err := activity.Execute(ctx.Context(), activities.Scope(ctx), IssueDescriptionSyncBatchInput{RunID: input.RunID, Offset: offset, BatchSize: input.BatchSize}, execution.ActivityOptions{})
			if err != nil {
				return IssueDescriptionSyncOutput{}, err
			}
			result.Batches++
			result.Inserted += batch.Inserted
			result.Skipped += batch.Skipped
			if batch.Total == 0 || batch.End <= offset || batch.End >= batch.Total {
				return result, nil
			}
			if err := selvedge.Sleep(ctx, input.Countdown, selvedge.WithSleepID(fmt.Sprintf("issue-description-sync-offset-%d", batch.End))); err != nil {
				return IssueDescriptionSyncOutput{}, err
			}
			offset = batch.End
		}
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &IssueDescriptionSyncDispatcher{app: app, workflow: workflow}, nil
}

// Admit starts the legacy schedule_issue_description_version task and its
// delayed, durable batch continuations. EventID identifies one explicit run.
func (dispatcher *IssueDescriptionSyncDispatcher) Admit(ctx context.Context, eventID string, batchSize, offset int, countdown time.Duration) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable issue description sync dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("issue description sync event ID must be a UUID: %w", err)
	}
	batchSize, offset, countdown, err := normalizeIssueSyncAdmission("issue description sync", batchSize, offset, countdown)
	if err != nil {
		return "", err
	}
	input := IssueDescriptionSyncInput{RunID: eventID, Offset: offset, BatchSize: batchSize, Countdown: countdown}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "issue-description-sync:"+eventID, input)
}

type PostgresIssueDescriptionSyncStore struct{ pool *pgxpool.Pool }

func NewPostgresIssueDescriptionSyncStore(pool *pgxpool.Pool) (*PostgresIssueDescriptionSyncStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresIssueDescriptionSyncStore{pool: pool}, nil
}

type syncIssueRow struct {
	ID          uuid.UUID
	Workspace   uuid.UUID
	Project     uuid.UUID
	CreatedBy   *uuid.UUID
	UpdatedBy   *uuid.UUID
	Owner       *uuid.UUID
	Description []byte
	HTML        string
	Binary      []byte
	Stripped    *string
}

func (store *PostgresIssueDescriptionSyncStore) SyncIssueDescriptionVersionBatch(ctx context.Context, input IssueDescriptionSyncBatchInput) (_ IssueDescriptionSyncBatchOutput, resultErr error) {
	if _, err := uuid.Parse(input.RunID); err != nil || input.Offset < 0 || input.BatchSize < 1 || input.BatchSize > issueSyncBatchLimit {
		return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("%w: invalid issue description sync batch", ErrPermanentVersionHistory)
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("begin issue description sync batch: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var total int
	if scanErr := tx.QueryRow(ctx, `SELECT COUNT(*) FROM issues WHERE deleted_at IS NULL`).Scan(&total); scanErr != nil {
		return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("count issues for description sync: %w", scanErr)
	}
	end := min(input.Offset+input.BatchSize, total)
	result := IssueDescriptionSyncBatchOutput{Total: total, End: end}
	if input.Offset >= total {
		return result, tx.Commit(ctx)
	}
	rows, err := tx.Query(ctx, `SELECT issue.id,issue.workspace_id,issue.project_id,issue.created_by_id,issue.updated_by_id,issue.description_json,issue.description_html,issue.description_binary,issue.description_stripped,(SELECT member.member_id FROM project_members member WHERE member.project_id=issue.project_id AND member.role=20 AND member.deleted_at IS NULL ORDER BY member.created_at DESC,member.id DESC LIMIT 1) AS fallback_owner FROM issues issue WHERE issue.deleted_at IS NULL ORDER BY issue.created_at ASC,issue.id ASC OFFSET $1 LIMIT $2`, input.Offset, input.BatchSize)
	if err != nil {
		return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("load issue description sync batch: %w", err)
	}
	batch := make([]syncIssueRow, 0, input.BatchSize)
	for rows.Next() {
		var issue syncIssueRow
		if err := rows.Scan(&issue.ID, &issue.Workspace, &issue.Project, &issue.CreatedBy, &issue.UpdatedBy, &issue.Description, &issue.HTML, &issue.Binary, &issue.Stripped, &issue.Owner); err != nil {
			rows.Close()
			return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("scan issue description sync batch: %w", err)
		}
		if issue.UpdatedBy != nil {
			issue.Owner = issue.UpdatedBy
		} else if issue.CreatedBy != nil {
			issue.Owner = issue.CreatedBy
		}
		batch = append(batch, issue)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("read issue description sync batch: %w", err)
	}
	rows.Close()
	if len(batch) == 0 {
		result.End = total
		return result, tx.Commit(ctx)
	}
	commands := &pgx.Batch{}
	for _, issue := range batch {
		if issue.Owner == nil {
			result.Skipped++
			continue
		}
		versionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(input.RunID+":"+issue.ID.String()))
		commands.Queue(`INSERT INTO issue_description_versions (id,workspace_id,project_id,issue_id,created_by_id,updated_by_id,owned_by_id,last_saved_at,description_binary,description_html,description_stripped,description_json,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$8,$8,NULL) ON CONFLICT (id) DO NOTHING`, versionID, issue.Workspace, issue.Project, issue.ID, issue.CreatedBy, issue.UpdatedBy, issue.Owner, time.Now().UTC(), issue.Binary, issue.HTML, issue.Stripped, issue.Description)
	}
	if len(batch) > result.Skipped {
		batchResults := tx.SendBatch(ctx, commands)
		for index := 0; index < len(batch)-result.Skipped; index++ {
			tag, err := batchResults.Exec()
			if err != nil {
				_ = batchResults.Close()
				return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("insert issue description sync version: %w", err)
			}
			result.Inserted += int(tag.RowsAffected())
		}
		if err := batchResults.Close(); err != nil {
			return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("close issue description sync batch: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return IssueDescriptionSyncBatchOutput{}, fmt.Errorf("commit issue description sync batch: %w", err)
	}
	return result, nil
}
