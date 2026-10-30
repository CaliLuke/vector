package backgroundtasks

import (
	"context"
	"encoding/json"
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
	IssueAutomationWorkflowName  = "plane.issues.archive_and_close_old"
	IssueAutomationActivityName  = "plane.issues.apply_archive_and_close_batch"
	issueAutomationScheduleName  = "plane.beat.archive-and-close-old-issues"
	issueAutomationScheduleID    = "plane-beat:archive-and-close-old-issues"
	issueAutomationBatchSize     = 250
	issueAutomationActivityLimit = 5 * time.Minute
)

type issueAutomationOperation string

const (
	archiveIssuesOperation issueAutomationOperation = "archive"
	closeIssuesOperation   issueAutomationOperation = "close"
)

type issueAutomationInput struct {
	ScheduledAt time.Time `json:"scheduled_at"`
}

type issueAutomationBatchInput struct {
	ScheduledAt time.Time                `json:"scheduled_at"`
	Operation   issueAutomationOperation `json:"operation"`
	AfterID     string                   `json:"after_id"`
}

type issueAutomationBatchOutput struct {
	LastID  string                        `json:"last_id"`
	HasMore bool                          `json:"has_more"`
	Events  []issueActivityExecutionInput `json:"events"`
}

type issueAutomationStore interface {
	ApplyIssueAutomationBatch(context.Context, issueAutomationBatchInput) (issueAutomationBatchOutput, error)
}

type IssueAutomationDispatcher struct {
	app              *selvedge.App
	workflow         *selvedge.WorkflowFunc[issueAutomationInput, struct{}]
	scheduleWorkflow *selvedge.WorkflowFunc[struct{}, struct{}]
}

type PostgresIssueAutomationStore struct{ pool *pgxpool.Pool }

func NewPostgresIssueAutomationStore(pool *pgxpool.Pool) (*PostgresIssueAutomationStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresIssueAutomationStore{pool: pool}, nil
}

func RegisterIssueAutomationWorkflow(app *selvedge.App, store issueAutomationStore, issueActivities *IssueActivityDispatcher, policy *retry.Policy) (*IssueAutomationDispatcher, error) {
	if app == nil || store == nil || issueActivities == nil {
		return nil, errors.New("workflow app, issue automation store and issue activity dispatcher are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[issueAutomationBatchInput, issueAutomationBatchOutput]{Name: IssueAutomationActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input issueAutomationBatchInput) (issueAutomationBatchOutput, error) {
		if input.Operation != archiveIssuesOperation && input.Operation != closeIssuesOperation {
			return issueAutomationBatchOutput{}, fmt.Errorf("unsupported issue automation %q", input.Operation)
		}
		result, err := store.ApplyIssueAutomationBatch(ctx, input)
		if err != nil {
			return issueAutomationBatchOutput{}, err
		}
		return result, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: issueAutomationActivityLimit, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue automation activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(IssueAutomationWorkflowName, func(ctx *selvedge.WorkflowContext, input issueAutomationInput) (struct{}, error) {
		if input.ScheduledAt.IsZero() {
			return struct{}{}, errors.New("issue automation schedule time is required")
		}
		scheduledAt := input.ScheduledAt.UTC()
		for _, operation := range []issueAutomationOperation{archiveIssuesOperation, closeIssuesOperation} {
			afterID := ""
			for {
				batchInput := issueAutomationBatchInput{ScheduledAt: scheduledAt, Operation: operation, AfterID: afterID}
				batch, err := activity.Execute(ctx.Context(), activities.Scope(ctx), batchInput, execution.ActivityOptions{})
				if err != nil {
					return struct{}{}, err
				}
				for _, event := range batch.Events {
					childID := "plane-issue-activity:" + event.EventID
					child, err := selvedge.StartChild[struct{}](ctx, issueActivityWorkflowName, event, selvedge.WithChildID(childID))
					if err != nil {
						return struct{}{}, err
					}
					if _, err := child.Get(ctx); err != nil {
						return struct{}{}, err
					}
				}
				if !batch.HasMore {
					break
				}
				afterID = batch.LastID
			}
		}
		return struct{}{}, nil
	})
	selvedge.RegisterWorkflow(app, workflow)
	schedule := selvedge.DefineWorkflow(issueAutomationScheduleName, func(ctx *selvedge.WorkflowContext, _ struct{}) (struct{}, error) {
		for {
			now, err := selvedge.Now(ctx)
			if err != nil {
				return struct{}{}, err
			}
			next := nextDailyUTC(now, 1, 0)
			if err := selvedge.SleepUntil(ctx, next, selvedge.WithSleepID("issue-automation-beat:"+next.Format(time.RFC3339))); err != nil {
				return struct{}{}, err
			}
			for attempt := 0; ; attempt++ {
				childID := fmt.Sprintf("plane-issue-automation:%s:%d", next.Format(time.RFC3339), attempt)
				child, err := selvedge.StartChild[struct{}](ctx, IssueAutomationWorkflowName, issueAutomationInput{ScheduledAt: next}, selvedge.WithChildID(childID))
				if err != nil {
					return struct{}{}, err
				}
				if _, err := child.Get(ctx); err == nil {
					break
				}
				if err := selvedge.Sleep(ctx, time.Minute, selvedge.WithSleepID(fmt.Sprintf("issue-automation-retry:%s:%d", next.Format(time.RFC3339), attempt))); err != nil {
					return struct{}{}, err
				}
			}
		}
	})
	selvedge.RegisterWorkflow(app, schedule)
	return &IssueAutomationDispatcher{app: app, workflow: workflow, scheduleWorkflow: schedule}, nil
}

func (dispatcher *IssueAutomationDispatcher) Admit(ctx context.Context, scheduledAt time.Time) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("issue automation dispatcher is not configured")
	}
	if scheduledAt.IsZero() {
		return "", errors.New("issue automation schedule time is required")
	}
	scheduledAt = scheduledAt.UTC()
	instanceID := "plane-issue-automation:" + scheduledAt.Format(time.RFC3339Nano)
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, instanceID, issueAutomationInput{ScheduledAt: scheduledAt})
}

func (dispatcher *IssueAutomationDispatcher) StartBeatSchedule(ctx context.Context) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.scheduleWorkflow == nil {
		return "", errors.New("issue automation Beat schedule is not configured")
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.scheduleWorkflow, issueAutomationScheduleID, struct{}{})
}

type issueAutomationCandidate struct {
	ProjectID, ActorID, IssueID string
	TargetState                 *string
	ArchivedAt                  *time.Time
}

func (store *PostgresIssueAutomationStore) ApplyIssueAutomationBatch(ctx context.Context, input issueAutomationBatchInput) (_ issueAutomationBatchOutput, resultErr error) {
	if input.ScheduledAt.IsZero() {
		return issueAutomationBatchOutput{}, errors.New("issue automation schedule time is required")
	}
	if input.Operation != archiveIssuesOperation && input.Operation != closeIssuesOperation {
		return issueAutomationBatchOutput{}, fmt.Errorf("unsupported issue automation %q", input.Operation)
	}
	input.ScheduledAt = input.ScheduledAt.UTC()
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueAutomationBatchOutput{}, fmt.Errorf("begin issue automation batch: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	candidates, err := loadIssueAutomationCandidates(ctx, tx, input, nil)
	if err != nil {
		return issueAutomationBatchOutput{}, err
	}
	result := issueAutomationBatchOutput{Events: make([]issueActivityExecutionInput, 0, len(candidates)), HasMore: len(candidates) == issueAutomationBatchSize}
	if len(candidates) > 0 {
		result.LastID = candidates[len(candidates)-1].IssueID
		if err = lockIssueAutomationSchemas(ctx, tx, candidates); err != nil {
			return issueAutomationBatchOutput{}, err
		}
		ids := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			ids = append(ids, candidate.IssueID)
		}
		// Recheck eligibility after acquiring schema locks and before row locks.
		candidates, err = loadIssueAutomationCandidates(ctx, tx, input, ids)
		if err != nil {
			return issueAutomationBatchOutput{}, err
		}
	}
	for _, candidate := range candidates {
		event, changed, err := applyIssueAutomationCandidate(ctx, tx, input, candidate)
		if err != nil {
			return issueAutomationBatchOutput{}, err
		}
		if changed {
			result.Events = append(result.Events, event)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return issueAutomationBatchOutput{}, fmt.Errorf("commit issue automation batch: %w", err)
	}
	return result, nil
}

func loadIssueAutomationCandidates(ctx context.Context, tx pgx.Tx, input issueAutomationBatchInput, lockedIDs []string) ([]issueAutomationCandidate, error) {
	query := `
SELECT project.id::text, COALESCE(project.created_by_id::text,''), issue.id::text,
       CASE WHEN project.default_state_id IS NULL THEN (
         SELECT state.id::text FROM states state WHERE state."group"='cancelled' AND state.deleted_at IS NULL ORDER BY state.sequence LIMIT 1
       ) ELSE project.default_state_id::text END,
       issue.archived_at
FROM projects project
JOIN issues issue ON issue.project_id=project.id
JOIN states state ON state.id=issue.state_id
WHERE project.deleted_at IS NULL AND project.archived_at IS NULL
  AND issue.deleted_at IS NULL AND issue.is_draft=false AND issue.id::text > $2
  AND state.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM cycle_issues ci JOIN cycles cycle ON cycle.id=ci.cycle_id
    WHERE ci.issue_id=issue.id AND ci.deleted_at IS NULL AND (cycle.end_date IS NULL OR cycle.end_date >= $1)
  )
  AND NOT EXISTS (
    SELECT 1 FROM module_issues mi JOIN modules module ON module.id=mi.module_id
    WHERE mi.issue_id=issue.id AND mi.deleted_at IS NULL AND (module.target_date IS NULL OR module.target_date >= $1)
  )
  AND (NOT EXISTS(SELECT 1 FROM intake_issues intake WHERE intake.issue_id=issue.id AND intake.deleted_at IS NULL)
       OR EXISTS(SELECT 1 FROM intake_issues intake WHERE intake.issue_id=issue.id AND intake.deleted_at IS NULL AND intake.status IN (1,-1,2)))`
	switch input.Operation {
	case archiveIssuesOperation:
		query += ` AND project.archive_in>0
  AND ((issue.archived_at IS NULL AND issue.updated_at <= $1 - project.archive_in * interval '30 days'
        AND state."group" IN ('completed','cancelled'))
       OR (issue.archived_at=$1::date AND EXISTS (
         SELECT 1 FROM issue_activities activity WHERE activity.issue_id=issue.id AND activity.deleted_at IS NULL
           AND activity.field='archived_at' AND activity.comment='Plane has archived the issue' AND activity.epoch=EXTRACT(EPOCH FROM $1)::double precision)))`
	case closeIssuesOperation:
		query += ` AND project.close_in>0 AND issue.archived_at IS NULL
  AND ((issue.updated_at <= $1 - project.close_in * interval '30 days' AND state."group" IN ('backlog','unstarted','started'))
       OR EXISTS (SELECT 1 FROM issue_activities activity WHERE activity.issue_id=issue.id AND activity.deleted_at IS NULL
         AND activity.field='state' AND activity.comment='Plane updated the state to ' AND activity.epoch=EXTRACT(EPOCH FROM $1)::double precision))`
	}
	args := []any{input.ScheduledAt, input.AfterID, issueAutomationBatchSize}
	if lockedIDs != nil {
		query += ` AND issue.id=ANY($4::uuid[])`
		args = append(args, lockedIDs)
	}
	query += ` ORDER BY issue.id LIMIT $3`
	if lockedIDs != nil {
		query += ` FOR UPDATE OF issue SKIP LOCKED`
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select issue automation batch: %w", err)
	}
	defer rows.Close()
	candidates := make([]issueAutomationCandidate, 0, issueAutomationBatchSize)
	for rows.Next() {
		var candidate issueAutomationCandidate
		if err := rows.Scan(&candidate.ProjectID, &candidate.ActorID, &candidate.IssueID, &candidate.TargetState, &candidate.ArchivedAt); err != nil {
			return nil, fmt.Errorf("read issue automation candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate issue automation candidates: %w", err)
	}
	return candidates, nil
}

func applyIssueAutomationCandidate(ctx context.Context, tx pgx.Tx, input issueAutomationBatchInput, candidate issueAutomationCandidate) (issueActivityExecutionInput, bool, error) {
	eventID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("plane.issue-automation:%s:%s:%s", input.Operation, input.ScheduledAt.Format(time.RFC3339Nano), candidate.IssueID))).String()
	if _, err := uuid.Parse(eventID); err != nil {
		return issueActivityExecutionInput{}, false, err
	}
	eventUUID, _ := uuid.Parse(eventID)
	activityID := uuid.NewSHA1(eventUUID, []byte("plane.issue_activity:0"))
	var alreadyRecorded bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_activities WHERE id=$1)`, activityID).Scan(&alreadyRecorded); err != nil {
		return issueActivityExecutionInput{}, false, fmt.Errorf("check issue automation activity: %w", err)
	}
	activity, err := issueAutomationActivityInput(input, candidate)
	if err != nil {
		return issueActivityExecutionInput{}, false, err
	}
	event := issueActivityExecutionInput{EventID: eventID, Activity: activity}
	if alreadyRecorded {
		return event, true, nil
	}
	projectID, err := uuid.Parse(candidate.ProjectID)
	if err != nil {
		return issueActivityExecutionInput{}, false, fmt.Errorf("parse issue automation project ID: %w", err)
	}
	issueID, err := uuid.Parse(candidate.IssueID)
	if err != nil {
		return issueActivityExecutionInput{}, false, fmt.Errorf("parse issue automation issue ID: %w", err)
	}
	workspaceID, err := issueActivityWorkspaceID(ctx, tx, projectID)
	if err != nil {
		return issueActivityExecutionInput{}, false, err
	}
	rows, err := buildIssueActivityRows(ctx, tx, activity)
	if err != nil {
		return issueActivityExecutionInput{}, false, fmt.Errorf("build issue automation activity: %w", err)
	}
	if len(rows) == 0 {
		return issueActivityExecutionInput{}, false, errors.New("issue automation produced no activity rows")
	}
	if input.Operation == archiveIssuesOperation {
		if _, err := tx.Exec(ctx, `UPDATE issues SET archived_at=$2::date,updated_at=clock_timestamp() WHERE id=$1`, issueID, input.ScheduledAt); err != nil {
			return issueActivityExecutionInput{}, false, fmt.Errorf("archive old issue: %w", err)
		}
	} else {
		if candidate.TargetState == nil {
			return issueActivityExecutionInput{}, false, errors.New("no cancelled state is configured for closing old issues")
		}
		if _, err := tx.Exec(ctx, `UPDATE issues SET state_id=$2::uuid,updated_at=clock_timestamp() WHERE id=$1`, issueID, *candidate.TargetState); err != nil {
			return issueActivityExecutionInput{}, false, fmt.Errorf("close old issue: %w", err)
		}
	}
	if err := persistIssueAutomationActivityRows(ctx, tx, eventUUID, workspaceID, projectID, rows); err != nil {
		return issueActivityExecutionInput{}, false, err
	}
	return event, true, nil
}

func issueActivityWorkspaceID(ctx context.Context, tx pgx.Tx, projectID uuid.UUID) (uuid.UUID, error) {
	var workspaceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT workspace_id FROM projects WHERE id=$1`, projectID).Scan(&workspaceID); err != nil {
		return uuid.Nil, fmt.Errorf("load workspace for issue automation: %w", err)
	}
	return workspaceID, nil
}

func issueAutomationActivityInput(input issueAutomationBatchInput, candidate issueAutomationCandidate) (IssueActivityInput, error) {
	actorID := (*string)(nil)
	if candidate.ActorID != "" {
		actorID = &candidate.ActorID
	}
	requested := map[string]any{}
	current := any(nil)
	if input.Operation == archiveIssuesOperation {
		requested["archived_at"] = input.ScheduledAt.Format("2006-01-02")
		requested["automation"] = true
		current = map[string]any{"archived_at": nil}
	} else {
		if candidate.TargetState == nil {
			return IssueActivityInput{}, errors.New("no cancelled state is configured for closing old issues")
		}
		requested["closed_to"] = *candidate.TargetState
	}
	requestedJSON, err := json.Marshal(requested)
	if err != nil {
		return IssueActivityInput{}, fmt.Errorf("encode issue automation activity request: %w", err)
	}
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return IssueActivityInput{}, fmt.Errorf("encode issue automation current state: %w", err)
	}
	subscriber := false
	return IssueActivityInput{
		Type: "issue.activity.updated", RequestedData: requestedJSON, CurrentInstance: currentJSON,
		IssueID: &candidate.IssueID, ActorID: actorID, ProjectID: candidate.ProjectID,
		Epoch: float64(input.ScheduledAt.Unix()), Subscriber: &subscriber, Notification: true,
	}, nil
}

func persistIssueAutomationActivityRows(ctx context.Context, tx pgx.Tx, eventID, workspaceID, projectID uuid.UUID, rows []issueActivityRow) error {
	for index, row := range rows {
		id := uuid.NewSHA1(eventID, []byte(fmt.Sprintf("plane.issue_activity:%d", index)))
		var issueID, commentID, actorID any
		if row.IssueID != nil {
			issueID = *row.IssueID
		}
		if row.CommentID != nil {
			commentID = *row.CommentID
		}
		if row.ActorID != nil {
			actorID = *row.ActorID
		}
		if _, err := tx.Exec(ctx, `INSERT INTO issue_activities (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,issue_id,issue_comment_id,actor_id,verb,field,old_value,new_value,comment,attachments,old_identifier,new_identifier,epoch) VALUES ($1,COALESCE($15::timestamptz,NOW()),NOW(),NULL,NULL,NULL,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10,$11,ARRAY[]::text[],$12,$13,NULLIF($14,0)) ON CONFLICT (id) DO NOTHING`, id, workspaceID, projectID, issueID, commentID, actorID, row.Verb, row.Field, row.OldValue, row.NewValue, row.Comment, row.OldIdentifier, row.NewIdentifier, row.Epoch, row.CreatedAt); err != nil {
			return fmt.Errorf("persist issue automation activity: %w", err)
		}
	}
	return nil
}

// Take workspace locks in a stable order before any item row lock. Batch
// candidates are read again under these locks, since eligibility may change.
func lockIssueAutomationSchemas(ctx context.Context, tx pgx.Tx, candidates []issueAutomationCandidate) error {
	projects := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		projects = append(projects, candidate.ProjectID)
	}
	rows, err := tx.Query(ctx, `SELECT DISTINCT workspace_id::text FROM projects WHERE id=ANY($1::uuid[]) ORDER BY workspace_id::text`, projects)
	if err != nil {
		return err
	}
	workspaces := []string{}
	for rows.Next() {
		var workspace string
		if err = rows.Scan(&workspace); err != nil {
			rows.Close()
			return err
		}
		workspaces = append(workspaces, workspace)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	for _, workspace := range workspaces {
		if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspace); err != nil {
			return err
		}
	}
	return nil
}
