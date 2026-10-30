package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CleanupQueue         = "plane-background"
	CleanupBatchSize     = 500
	cleanupActivityName  = "plane.cleanup.delete_batch"
	cleanupActivityLimit = 5 * time.Minute
)

type CleanupTask string

const (
	CleanupAPIActivityLogs          CleanupTask = "delete_api_logs"
	CleanupEmailNotificationLogs    CleanupTask = "delete_email_notification_logs"
	CleanupPageVersions             CleanupTask = "delete_page_versions"
	CleanupIssueDescriptionVersions CleanupTask = "delete_issue_description_versions"
	CleanupWebhookLogs              CleanupTask = "delete_webhook_logs"
	CleanupUnuploadedFileAssets     CleanupTask = "delete_unuploaded_file_asset"
	CleanupMutationReceipts         CleanupTask = "delete_expired_mutation_receipts"
)

var cleanupTasks = []CleanupTask{
	CleanupAPIActivityLogs,
	CleanupEmailNotificationLogs,
	CleanupPageVersions,
	CleanupIssueDescriptionVersions,
	CleanupWebhookLogs,
	CleanupUnuploadedFileAssets,
	CleanupMutationReceipts,
}

type cleanupBeatSpec struct {
	task   CleanupTask
	name   string
	hour   int
	minute int
}

// Legacy schedules retain their Celery Beat times; receipt expiry is native.
// These schedules use durable
// Selvedge timers and deterministic child identities so restarting the API
// process cannot lose or duplicate a scheduled cleanup run.
var cleanupBeatSchedules = []cleanupBeatSpec{
	{task: CleanupUnuploadedFileAssets, name: "delete-unuploaded-file-assets", hour: 2, minute: 0},
	{task: CleanupAPIActivityLogs, name: "delete-api-logs", hour: 2, minute: 30},
	{task: CleanupEmailNotificationLogs, name: "delete-email-notification-logs", hour: 2, minute: 45},
	{task: CleanupPageVersions, name: "delete-page-versions", hour: 3, minute: 0},
	{task: CleanupIssueDescriptionVersions, name: "delete-issue-description-versions", hour: 3, minute: 15},
	{task: CleanupWebhookLogs, name: "delete-webhook-logs", hour: 3, minute: 30},
	{task: CleanupMutationReceipts, name: "delete-expired-mutation-receipts", hour: 3, minute: 45},
}

var ErrPermanentCleanup = errors.New("permanent cleanup task failure")

type CleanupWorkflowInput struct {
	Task        CleanupTask `json:"task"`
	ScheduledAt time.Time   `json:"scheduled_at"`
	Cutoff      time.Time   `json:"cutoff"`
}

type CleanupBatchInput struct {
	Task   CleanupTask `json:"task"`
	Cutoff time.Time   `json:"cutoff"`
}

type CleanupBatchOutput struct {
	Deleted int64 `json:"deleted"`
}

type CleanupRunOutput struct {
	Deleted int64 `json:"deleted"`
	Batches int64 `json:"batches"`
}

type cleanupBatchDeleter interface {
	DeleteCleanupBatch(context.Context, CleanupBatchInput) (int64, error)
}

type CleanupDispatcher struct {
	app       *selvedge.App
	workflows map[CleanupTask]*selvedge.WorkflowFunc[CleanupWorkflowInput, CleanupRunOutput]
	schedules map[string]*selvedge.WorkflowFunc[struct{}, struct{}]
}

// PostgresCleanupStore removes one bounded batch with set-based PostgreSQL
// statements. FileAssets use the source model's soft-delete behavior; the
// other cleanup models use all_objects and are hard-deleted.
type PostgresCleanupStore struct {
	pool *pgxpool.Pool
}

func NewPostgresCleanupStore(pool *pgxpool.Pool) (*PostgresCleanupStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresCleanupStore{pool: pool}, nil
}

func (store *PostgresCleanupStore) DeleteCleanupBatch(ctx context.Context, input CleanupBatchInput) (int64, error) {
	var query string
	switch input.Task {
	case CleanupAPIActivityLogs:
		query = `WITH batch AS (
 SELECT id FROM api_activity_logs WHERE created_at <= $1 ORDER BY created_at ASC LIMIT $2
)
DELETE FROM api_activity_logs AS target USING batch WHERE target.id = batch.id`
	case CleanupEmailNotificationLogs:
		query = `WITH batch AS (
 SELECT id FROM email_notification_logs WHERE sent_at <= $1 ORDER BY sent_at ASC LIMIT $2
)
DELETE FROM email_notification_logs AS target USING batch WHERE target.id = batch.id`
	case CleanupPageVersions:
		query = `WITH excess AS (
 SELECT id FROM (
  SELECT id, created_at, row_number() OVER (PARTITION BY page_id ORDER BY created_at DESC) AS row_num
  FROM page_versions
 ) AS ranked
 WHERE row_num > 20
 ORDER BY created_at ASC
 LIMIT $1
)
DELETE FROM page_versions AS target USING excess WHERE target.id = excess.id`
	case CleanupIssueDescriptionVersions:
		query = `WITH excess AS (
 SELECT id FROM (
  SELECT id, created_at, row_number() OVER (PARTITION BY issue_id ORDER BY created_at DESC) AS row_num
  FROM issue_description_versions
 ) AS ranked
 WHERE row_num > 20
 ORDER BY created_at ASC
 LIMIT $1
)
DELETE FROM issue_description_versions AS target USING excess WHERE target.id = excess.id`
	case CleanupWebhookLogs:
		query = `WITH batch AS (
 SELECT id FROM webhook_logs WHERE created_at <= $1 ORDER BY created_at ASC LIMIT $2
)
DELETE FROM webhook_logs AS target USING batch WHERE target.id = batch.id`
	case CleanupUnuploadedFileAssets:
		query = `WITH batch AS (
 SELECT id FROM file_assets WHERE created_at < $1 AND is_uploaded=false AND deleted_at IS NULL ORDER BY created_at ASC LIMIT $2
)
UPDATE file_assets AS target SET deleted_at=clock_timestamp() FROM batch WHERE target.id = batch.id`
	case CleanupMutationReceipts:
		// Use database time as an upper bound even if a caller supplies a future
		// cutoff. Skip receipt rows held by replay or renewal transactions. ctids
		// are used only within this statement while the selected rows are locked.
		query = `WITH batch AS (
 SELECT ctid FROM mutation_receipts
 WHERE replay_expires_at <= LEAST($1, statement_timestamp())
 ORDER BY replay_expires_at ASC LIMIT $2 FOR UPDATE SKIP LOCKED
)
DELETE FROM mutation_receipts AS target USING batch WHERE target.ctid = batch.ctid
 AND target.replay_expires_at <= LEAST($1, statement_timestamp())`
	default:
		return 0, fmt.Errorf("%w: unknown cleanup task %q", ErrPermanentCleanup, input.Task)
	}

	var result pgconn.CommandTag
	var err error
	if input.Task == CleanupPageVersions || input.Task == CleanupIssueDescriptionVersions {
		result, err = store.pool.Exec(ctx, query, CleanupBatchSize)
	} else {
		result, err = store.pool.Exec(ctx, query, input.Cutoff, CleanupBatchSize)
	}
	if err != nil {
		return 0, fmt.Errorf("delete %s batch: %w", input.Task, err)
	}
	return result.RowsAffected(), nil
}

// RegisterCleanupWorkflows registers each cleanup task as a named workflow.
// Every run has a fixed cutoff and uses native Selvedge retries/checkpoints.
func RegisterCleanupWorkflows(app *selvedge.App, store cleanupBatchDeleter, policy *retry.Policy) (*CleanupDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("workflow app and cleanup store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentCleanup)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[CleanupBatchInput, CleanupBatchOutput]{Name: cleanupActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input CleanupBatchInput) (CleanupBatchOutput, error) {
		deleted, err := store.DeleteCleanupBatch(ctx, input)
		if err != nil {
			return CleanupBatchOutput{}, classifyCleanupError(err)
		}
		return CleanupBatchOutput{Deleted: deleted}, nil
	}, selvedge.ActivityOptions{Queue: CleanupQueue, StartToCloseTimeout: cleanupActivityLimit, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register cleanup batch activity: %w", err)
	}

	dispatcher := &CleanupDispatcher{
		app:       app,
		workflows: make(map[CleanupTask]*selvedge.WorkflowFunc[CleanupWorkflowInput, CleanupRunOutput], len(cleanupTasks)),
		schedules: make(map[string]*selvedge.WorkflowFunc[struct{}, struct{}], len(cleanupBeatSchedules)),
	}
	for _, task := range cleanupTasks {
		task := task
		workflow := selvedge.DefineWorkflow(cleanupWorkflowName(task), func(ctx *selvedge.WorkflowContext, input CleanupWorkflowInput) (CleanupRunOutput, error) {
			if input.Task != task {
				return CleanupRunOutput{}, fmt.Errorf("%w: workflow %q received task %q", ErrPermanentCleanup, task, input.Task)
			}
			var result CleanupRunOutput
			for {
				batch, err := activity.Execute(ctx.Context(), activities.Scope(ctx), CleanupBatchInput{Task: task, Cutoff: input.Cutoff}, execution.ActivityOptions{})
				if err != nil {
					return CleanupRunOutput{}, err
				}
				result.Deleted += batch.Deleted
				if batch.Deleted == 0 {
					return result, nil
				}
				result.Batches++
				if batch.Deleted < CleanupBatchSize {
					return result, nil
				}
			}
		})
		selvedge.RegisterWorkflow(app, workflow)
		dispatcher.workflows[task] = workflow
	}
	for _, spec := range cleanupBeatSchedules {
		spec := spec
		schedule := selvedge.DefineWorkflow("plane.beat."+spec.name, func(ctx *selvedge.WorkflowContext, _ struct{}) (struct{}, error) {
			for {
				now, err := selvedge.Now(ctx)
				if err != nil {
					return struct{}{}, err
				}
				next := nextDailyUTC(now, spec.hour, spec.minute)
				if err := selvedge.SleepUntil(ctx, next, selvedge.WithSleepID("cleanup-beat:"+spec.name+":"+next.Format(time.RFC3339))); err != nil {
					return struct{}{}, err
				}
				for attempt := 0; ; attempt++ {
					input := CleanupWorkflowInput{Task: spec.task, ScheduledAt: next}
					if spec.task == CleanupUnuploadedFileAssets {
						retention, err := unuploadedAssetRetentionDays()
						if err != nil {
							return struct{}{}, err
						}
						input.Cutoff = next.Add(-time.Duration(retention) * 24 * time.Hour)
					} else if retention, ok := cleanupRetention(spec.task); ok {
						input.Cutoff = next.Add(-time.Duration(retention) * 24 * time.Hour)
					} else {
						input.Cutoff = next
					}
					childID := fmt.Sprintf("plane-cleanup-scheduled:%s:%s:%d", spec.name, next.Format(time.RFC3339), attempt)
					child, err := selvedge.StartChild[CleanupRunOutput](ctx, cleanupWorkflowName(spec.task), input, selvedge.WithChildID(childID))
					if err != nil {
						return struct{}{}, err
					}
					if _, err := child.Get(ctx); err == nil {
						break
					}
					if err := selvedge.Sleep(ctx, time.Minute, selvedge.WithSleepID(fmt.Sprintf("cleanup-beat-retry:%s:%d", next.Format(time.RFC3339), attempt))); err != nil {
						return struct{}{}, err
					}
				}
			}
		})
		selvedge.RegisterWorkflow(app, schedule)
		dispatcher.schedules[spec.name] = schedule
	}
	return dispatcher, nil
}

func (dispatcher *CleanupDispatcher) StartBeatSchedules(ctx context.Context) error {
	if dispatcher == nil || dispatcher.app == nil {
		return errors.New("durable cleanup schedules are not configured")
	}
	for _, spec := range cleanupBeatSchedules {
		workflow := dispatcher.schedules[spec.name]
		if workflow == nil {
			return fmt.Errorf("cleanup Beat schedule %q is not registered", spec.name)
		}
		if _, err := admitOrReconcile(ctx, dispatcher.app, workflow, "plane-beat:"+spec.name, struct{}{}); err != nil {
			return fmt.Errorf("start cleanup Beat schedule %s: %w", spec.name, err)
		}
	}
	return nil
}

func nextDailyUTC(now time.Time, hour, minute int) time.Time {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// Admit starts one scheduled cleanup. A repeated admission for the same task
// and schedule instant uses the same durable workflow identity.
func (dispatcher *CleanupDispatcher) Admit(ctx context.Context, task CleanupTask, scheduledAt time.Time) (string, error) {
	if dispatcher == nil || dispatcher.app == nil {
		return "", errors.New("durable cleanup dispatcher is not configured")
	}
	workflow, ok := dispatcher.workflows[task]
	if !ok {
		return "", fmt.Errorf("unknown cleanup task %q", task)
	}
	scheduledAt = scheduledAt.UTC()
	cutoff := scheduledAt
	if task == CleanupUnuploadedFileAssets {
		retention, err := unuploadedAssetRetentionDays()
		if err != nil {
			return "", err
		}
		cutoff = scheduledAt.Add(-time.Duration(retention) * 24 * time.Hour)
	}
	if retention, ok := cleanupRetention(task); ok {
		cutoff = scheduledAt.Add(-time.Duration(retention) * 24 * time.Hour)
	}
	input := CleanupWorkflowInput{Task: task, ScheduledAt: scheduledAt, Cutoff: cutoff}
	instanceID := fmt.Sprintf("plane-cleanup:%s:%s", task, scheduledAt.Format(time.RFC3339Nano))
	startedID, err := admitOrReconcile(ctx, dispatcher.app, workflow, instanceID, input)
	if err != nil {
		return "", fmt.Errorf("admit cleanup task %s: %w", task, err)
	}
	return startedID, nil
}

func unuploadedAssetRetentionDays() (int, error) {
	raw, ok := os.LookupEnv("UNUPLOADED_ASSET_DELETE_DAYS")
	if !ok {
		return 7, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 0 {
		return 0, fmt.Errorf("UNUPLOADED_ASSET_DELETE_DAYS must be a non-negative integer")
	}
	return days, nil
}

func cleanupWorkflowName(task CleanupTask) string {
	return "plane.cleanup." + string(task)
}

func cleanupRetention(task CleanupTask) (int, bool) {
	switch task {
	case CleanupAPIActivityLogs:
		return retentionDays("API_ACTIVITY_LOG_RETENTION_DAYS", 14), true
	case CleanupEmailNotificationLogs:
		return retentionDays("EMAIL_LOG_RETENTION_DAYS", 7), true
	case CleanupWebhookLogs:
		return retentionDays("WEBHOOK_LOG_RETENTION_DAYS", 14), true
	default:
		return 0, false
	}
}

func retentionDays(name string, fallback int) int {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 0 {
		return fallback
	}
	return days
}

func classifyCleanupError(err error) error {
	if errors.Is(err, ErrPermanentCleanup) {
		return err
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		if len(databaseError.Code) < 2 {
			return fmt.Errorf("%w: %v", ErrPermanentCleanup, err)
		}
		switch databaseError.Code[:2] {
		case "08", "40", "53", "57":
		default:
			return fmt.Errorf("%w: %w", ErrPermanentCleanup, err)
		}
	}
	return err
}
