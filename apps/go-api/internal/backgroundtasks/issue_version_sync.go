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
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	IssueVersionSyncWorkflowName = "plane.issues.sync_versions"
	issueVersionSyncActivity     = "plane.issues.sync_version_batch"
	issueVersionSyncQueue        = "plane-background"
)

type IssueVersionSyncInput struct {
	RunID     string        `json:"run_id"`
	Offset    int           `json:"offset"`
	BatchSize int           `json:"batch_size"`
	Countdown time.Duration `json:"countdown"`
}

type IssueVersionSyncBatchInput struct {
	RunID     string `json:"run_id"`
	Offset    int    `json:"offset"`
	BatchSize int    `json:"batch_size"`
}

type IssueVersionSyncBatchOutput struct {
	Total    int `json:"total"`
	End      int `json:"end"`
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

type IssueVersionSyncOutput struct {
	Batches  int `json:"batches"`
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

type issueVersionSyncStore interface {
	SyncIssueVersionBatch(context.Context, IssueVersionSyncBatchInput) (IssueVersionSyncBatchOutput, error)
}

type IssueVersionSyncDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[IssueVersionSyncInput, IssueVersionSyncOutput]
}

func RegisterIssueVersionSyncWorkflow(app *selvedge.App, store issueVersionSyncStore, policy *retry.Policy) (*IssueVersionSyncDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and issue version sync store are required")
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
	activity := execution.Activity[IssueVersionSyncBatchInput, IssueVersionSyncBatchOutput]{Name: issueVersionSyncActivity}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input IssueVersionSyncBatchInput) (IssueVersionSyncBatchOutput, error) {
		if _, err := uuid.Parse(input.RunID); err != nil || input.Offset < 0 || input.BatchSize < 1 || input.BatchSize > issueSyncBatchLimit {
			return IssueVersionSyncBatchOutput{}, fmt.Errorf("%w: invalid issue version sync batch", ErrPermanentVersionHistory)
		}
		result, err := store.SyncIssueVersionBatch(ctx, input)
		if err != nil {
			return IssueVersionSyncBatchOutput{}, classifyVersionHistoryError(err)
		}
		return result, nil
	}, selvedge.ActivityOptions{Queue: issueVersionSyncQueue, StartToCloseTimeout: 2 * time.Minute, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue version sync activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(IssueVersionSyncWorkflowName, func(ctx *selvedge.WorkflowContext, input IssueVersionSyncInput) (IssueVersionSyncOutput, error) {
		if _, err := uuid.Parse(input.RunID); err != nil || input.Offset < 0 || input.BatchSize < 1 || input.BatchSize > issueSyncBatchLimit || input.Countdown < 0 {
			return IssueVersionSyncOutput{}, fmt.Errorf("%w: invalid issue version sync workflow input", ErrPermanentVersionHistory)
		}
		var result IssueVersionSyncOutput
		offset := input.Offset
		for {
			batch, err := activity.Execute(ctx.Context(), activities.Scope(ctx), IssueVersionSyncBatchInput{RunID: input.RunID, Offset: offset, BatchSize: input.BatchSize}, execution.ActivityOptions{})
			if err != nil {
				return IssueVersionSyncOutput{}, err
			}
			result.Batches++
			result.Inserted += batch.Inserted
			result.Skipped += batch.Skipped
			if batch.Total == 0 || batch.End <= offset || batch.End >= batch.Total {
				return result, nil
			}
			if err := selvedge.Sleep(ctx, input.Countdown, selvedge.WithSleepID(fmt.Sprintf("issue-version-sync-offset-%d", batch.End))); err != nil {
				return IssueVersionSyncOutput{}, err
			}
			offset = batch.End
		}
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &IssueVersionSyncDispatcher{app: app, workflow: workflow}, nil
}

func (dispatcher *IssueVersionSyncDispatcher) Admit(ctx context.Context, eventID string, batchSize, offset int, countdown time.Duration) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable issue version sync dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("issue version sync event ID must be a UUID: %w", err)
	}
	batchSize, offset, countdown, err := normalizeIssueSyncAdmission("issue version sync", batchSize, offset, countdown)
	if err != nil {
		return "", err
	}
	input := IssueVersionSyncInput{RunID: eventID, Offset: offset, BatchSize: batchSize, Countdown: countdown}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "issue-version-sync:"+eventID, input)
}

type PostgresIssueVersionSyncStore struct{ pool *pgxpool.Pool }

func NewPostgresIssueVersionSyncStore(pool *pgxpool.Pool) (*PostgresIssueVersionSyncStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresIssueVersionSyncStore{pool: pool}, nil
}

type syncIssueVersionRow struct {
	ID, Workspace, Project                         uuid.UUID
	CreatedBy, UpdatedBy, Owner                    *uuid.UUID
	Activity, Parent, State, Estimate, Type, Cycle *uuid.UUID
	Name, Priority                                 string
	StartDate, TargetDate, ArchivedAt              *time.Time
	Assignees, Labels, Modules                     []uuid.UUID
	SequenceID                                     int
	SortOrder                                      float64
	CompletedAt                                    *time.Time
	IsDraft                                        bool
	ExternalSource, ExternalID                     *string
}

func (store *PostgresIssueVersionSyncStore) SyncIssueVersionBatch(ctx context.Context, input IssueVersionSyncBatchInput) (IssueVersionSyncBatchOutput, error) {
	if _, err := uuid.Parse(input.RunID); err != nil || input.Offset < 0 || input.BatchSize < 1 || input.BatchSize > issueSyncBatchLimit {
		return IssueVersionSyncBatchOutput{}, fmt.Errorf("%w: invalid issue version sync batch", ErrPermanentVersionHistory)
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return IssueVersionSyncBatchOutput{}, fmt.Errorf("begin issue version sync batch: %w", err)
	}
	defer tx.Rollback(ctx)
	var total int
	if scanErr := tx.QueryRow(ctx, `SELECT COUNT(*) FROM issues WHERE deleted_at IS NULL`).Scan(&total); scanErr != nil {
		return IssueVersionSyncBatchOutput{}, fmt.Errorf("count issues for version sync: %w", scanErr)
	}
	end := min(input.Offset+input.BatchSize, total)
	result := IssueVersionSyncBatchOutput{Total: total, End: end}
	if input.Offset >= total {
		return result, tx.Commit(ctx)
	}
	rows, err := tx.Query(ctx, `SELECT issue.id,issue.workspace_id,issue.project_id,issue.created_by_id,issue.updated_by_id,COALESCE(issue.updated_by_id,issue.created_by_id,(SELECT member.member_id FROM project_members member WHERE member.project_id=issue.project_id AND member.role=20 AND member.deleted_at IS NULL ORDER BY member.created_at DESC,member.id DESC LIMIT 1)),(SELECT activity.id FROM issue_activities activity WHERE activity.issue_id=issue.id AND activity.deleted_at IS NULL ORDER BY activity.created_at DESC,activity.id DESC LIMIT 1),issue.parent_id,issue.state_id,issue.estimate_point_id,issue.name,issue.priority,issue.start_date,issue.target_date,COALESCE((SELECT array_agg(assignee.assignee_id) FROM issue_assignees assignee WHERE assignee.issue_id=issue.id AND assignee.deleted_at IS NULL),ARRAY[]::uuid[]),issue.sequence_id,COALESCE((SELECT array_agg(label.label_id) FROM issue_labels label WHERE label.issue_id=issue.id AND label.deleted_at IS NULL),ARRAY[]::uuid[]),issue.sort_order,issue.completed_at,issue.archived_at,issue.is_draft,issue.external_source,issue.external_id,issue.type_id,(SELECT cycle_issue.cycle_id FROM cycle_issues cycle_issue WHERE cycle_issue.issue_id=issue.id AND cycle_issue.deleted_at IS NULL ORDER BY cycle_issue.created_at DESC,cycle_issue.id DESC LIMIT 1),COALESCE((SELECT array_agg(module_issue.module_id) FROM module_issues module_issue JOIN modules module ON module.id=module_issue.module_id AND module.deleted_at IS NULL WHERE module_issue.issue_id=issue.id AND module_issue.deleted_at IS NULL),ARRAY[]::uuid[]) FROM issues issue WHERE issue.deleted_at IS NULL ORDER BY issue.created_at ASC,issue.id ASC OFFSET $1 LIMIT $2`, input.Offset, input.BatchSize)
	if err != nil {
		return IssueVersionSyncBatchOutput{}, fmt.Errorf("load issue version sync batch: %w", err)
	}
	batch := make([]syncIssueVersionRow, 0, input.BatchSize)
	for rows.Next() {
		var issue syncIssueVersionRow
		if err := rows.Scan(&issue.ID, &issue.Workspace, &issue.Project, &issue.CreatedBy, &issue.UpdatedBy, &issue.Owner, &issue.Activity, &issue.Parent, &issue.State, &issue.Estimate, &issue.Name, &issue.Priority, &issue.StartDate, &issue.TargetDate, &issue.Assignees, &issue.SequenceID, &issue.Labels, &issue.SortOrder, &issue.CompletedAt, &issue.ArchivedAt, &issue.IsDraft, &issue.ExternalSource, &issue.ExternalID, &issue.Type, &issue.Cycle, &issue.Modules); err != nil {
			rows.Close()
			return IssueVersionSyncBatchOutput{}, fmt.Errorf("scan issue version sync batch: %w", err)
		}
		batch = append(batch, issue)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return IssueVersionSyncBatchOutput{}, fmt.Errorf("read issue version sync batch: %w", err)
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
		commands.Queue(`INSERT INTO issue_versions (id,workspace_id,project_id,issue_id,created_by_id,updated_by_id,owned_by_id,activity_id,parent,state,estimate_point,name,priority,start_date,target_date,assignees,sequence_id,labels,sort_order,completed_at,archived_at,is_draft,external_source,external_id,type,cycle,modules,properties,meta,last_saved_at,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,'{}'::jsonb,'{}'::jsonb,$28,$28,$28,NULL) ON CONFLICT (id) DO NOTHING`, versionID, issue.Workspace, issue.Project, issue.ID, issue.CreatedBy, issue.UpdatedBy, issue.Owner, issue.Activity, issue.Parent, issue.State, issue.Estimate, issue.Name, issue.Priority, issue.StartDate, issue.TargetDate, issue.Assignees, issue.SequenceID, issue.Labels, issue.SortOrder, issue.CompletedAt, issue.ArchivedAt, issue.IsDraft, issue.ExternalSource, issue.ExternalID, issue.Type, issue.Cycle, issue.Modules, time.Now().UTC())
	}
	if len(batch) > result.Skipped {
		batchResults := tx.SendBatch(ctx, commands)
		for index := 0; index < len(batch)-result.Skipped; index++ {
			tag, err := batchResults.Exec()
			if err != nil {
				_ = batchResults.Close()
				return IssueVersionSyncBatchOutput{}, fmt.Errorf("insert issue version sync snapshot: %w", err)
			}
			result.Inserted += int(tag.RowsAffected())
		}
		if err := batchResults.Close(); err != nil {
			return IssueVersionSyncBatchOutput{}, fmt.Errorf("close issue version sync batch: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return IssueVersionSyncBatchOutput{}, fmt.Errorf("commit issue version sync batch: %w", err)
	}
	return result, nil
}
