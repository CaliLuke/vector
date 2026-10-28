package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	issueActivityWorkflowName     = "plane.issues.record_activity"
	issueActivityActivityName     = "plane.issues.write_activity"
	issueNotificationActivityName = "plane.notifications.write_issue_activity"
	issueActivityQueue            = "plane-background"
)

var ErrPermanentIssueActivity = errors.New("permanent issue activity task failure")

// IssueActivityInput is the typed envelope used by the Go producer adapter.
// EventID identifies one producer operation and is deliberately independent
// of the activity payload so a later identical action remains a new event.
type IssueActivityInput struct {
	Type            string          `json:"type"`
	RequestedData   json.RawMessage `json:"requested_data"`
	CurrentInstance json.RawMessage `json:"current_instance"`
	IssueID         *string         `json:"issue_id"`
	ActorID         *string         `json:"actor_id"`
	ProjectID       string          `json:"project_id"`
	WorkspaceID     string          `json:"workspace_id,omitempty"`
	Epoch           float64         `json:"epoch"`
	Subscriber      *bool           `json:"subscriber,omitempty"`
	Notification    bool            `json:"notification,omitempty"`
	Origin          string          `json:"origin,omitempty"`
}

type issueActivityStore interface {
	RecordIssueActivity(context.Context, string, IssueActivityInput) ([]string, error)
	ProcessIssueNotifications(context.Context, string, IssueActivityInput, []string) error
}

type IssueActivityDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[issueActivityExecutionInput, struct{}]
	store    issueActivityStore
}

func RegisterIssueActivityWorkflow(app *selvedge.App, store issueActivityStore, policy *retry.Policy) (*IssueActivityDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and issue activity store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentIssueActivity)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[issueActivityExecutionInput, []string]{Name: issueActivityActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input issueActivityExecutionInput) ([]string, error) {
		if err := validateIssueActivityInput(input.EventID, input.Activity); err != nil {
			return nil, err
		}
		activityIDs, err := store.RecordIssueActivity(ctx, input.EventID, input.Activity)
		if err != nil {
			return nil, classifyIssueActivityError(err)
		}
		return activityIDs, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 30 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue activity activity: %w", err)
	}
	notificationActivity := execution.Activity[issueNotificationExecutionInput, struct{}]{Name: issueNotificationActivityName}
	if err := executionSelvedge.Register(activities, notificationActivity, func(ctx context.Context, input issueNotificationExecutionInput) (struct{}, error) {
		if err := validateIssueActivityInput(input.EventID, input.Activity); err != nil {
			return struct{}{}, err
		}
		if err := store.ProcessIssueNotifications(ctx, input.EventID, input.Activity, input.ActivityIDs); err != nil {
			return struct{}{}, classifyIssueActivityError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 30 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue notification activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(issueActivityWorkflowName, func(ctx *selvedge.WorkflowContext, input issueActivityExecutionInput) (struct{}, error) {
		if err := validateIssueActivityInput(input.EventID, input.Activity); err != nil {
			return struct{}{}, err
		}
		activityIDs, err := activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err != nil {
			return struct{}{}, err
		}
		if input.Activity.Notification {
			return notificationActivity.Execute(ctx.Context(), activities.Scope(ctx), issueNotificationExecutionInput{EventID: input.EventID, Activity: input.Activity, ActivityIDs: activityIDs}, execution.ActivityOptions{})
		}
		return struct{}{}, nil
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &IssueActivityDispatcher{app: app, workflow: workflow, store: store}, nil
}

func (dispatcher *IssueActivityDispatcher) Close() error {
	if dispatcher == nil {
		return nil
	}
	if closer, ok := dispatcher.store.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

type issueActivityExecutionInput struct {
	EventID  string             `json:"event_id"`
	Activity IssueActivityInput `json:"activity"`
}

type issueNotificationExecutionInput struct {
	EventID     string             `json:"event_id"`
	Activity    IssueActivityInput `json:"activity"`
	ActivityIDs []string           `json:"activity_ids"`
}

func (dispatcher *IssueActivityDispatcher) Admit(ctx context.Context, eventID string, input IssueActivityInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable issue activity dispatcher is not configured")
	}
	if err := validateIssueActivityInput(eventID, input); err != nil {
		return "", err
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "issue-activity:"+eventID, issueActivityExecutionInput{EventID: eventID, Activity: input})
}

func validateIssueActivityInput(eventID string, input IssueActivityInput) error {
	if _, err := uuid.Parse(eventID); err != nil {
		return fmt.Errorf("%w: event ID must be a UUID", ErrPermanentIssueActivity)
	}
	if _, err := uuid.Parse(input.ProjectID); err != nil {
		return fmt.Errorf("%w: invalid project ID", ErrPermanentIssueActivity)
	}
	if input.IssueID != nil {
		if _, err := uuid.Parse(*input.IssueID); err != nil {
			return fmt.Errorf("%w: invalid issue ID", ErrPermanentIssueActivity)
		}
	}
	if (input.Type == "issue.activity.updated" || input.Type == "issue.activity.created" || input.Type == "issue.activity.deleted") && input.IssueID == nil {
		return fmt.Errorf("%w: issue ID is required for issue activity", ErrPermanentIssueActivity)
	}
	if input.ActorID != nil {
		if _, err := uuid.Parse(*input.ActorID); err != nil {
			return fmt.Errorf("%w: invalid actor ID", ErrPermanentIssueActivity)
		}
	}
	if input.Type == "" {
		return fmt.Errorf("%w: activity type is required", ErrPermanentIssueActivity)
	}
	for name, raw := range map[string]json.RawMessage{"requested_data": input.RequestedData, "current_instance": input.CurrentInstance} {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var object map[string]any
		if err := json.Unmarshal(raw, &object); err != nil || object == nil {
			return fmt.Errorf("%w: %s must be a JSON object or null", ErrPermanentIssueActivity, name)
		}
	}
	if !supportedIssueActivityType(input.Type) {
		return fmt.Errorf("%w: unsupported activity type %q", ErrPermanentIssueActivity, input.Type)
	}
	return validateMilestoneActivityInput(input)
}

func supportedIssueActivityType(activityType string) bool {
	switch activityType {
	case "workitem_relation.activity.created", "workitem_relation.activity.deleted", "issue.activity.created", "issue.activity.deleted", "issue.activity.updated", "cycle.activity.created", "cycle.activity.deleted", "module.activity.created", "module.activity.deleted", "issue_relation.activity.created", "issue_relation.activity.deleted", "comment.activity.created", "comment.activity.updated", "comment.activity.deleted",
		"link.activity.created", "link.activity.updated", "link.activity.deleted",
		"attachment.activity.created", "attachment.activity.deleted",
		"work_log.activity.created", "work_log.activity.updated", "work_log.activity.deleted",
		"issue_reaction.activity.created", "issue_reaction.activity.deleted",
		"comment_reaction.activity.created", "comment_reaction.activity.deleted",
		"issue_vote.activity.created", "issue_vote.activity.deleted",
		"issue_draft.activity.created", "issue_draft.activity.updated", "issue_draft.activity.deleted",
		"intake.activity.created":
		return true
	default:
		return false
	}
}

type issueActivityRow struct {
	IssueID                      *string
	CommentID                    *string
	Verb, Field                  string
	OldValue, NewValue           *string
	Comment                      string
	OldIdentifier, NewIdentifier *string
	ActorID                      *string
	Epoch                        float64
	CreatedAt                    *time.Time
}

type PostgresIssueActivityStore struct {
	pool   *pgxpool.Pool
	origin *redis.Client
}

func NewPostgresIssueActivityStore(pool *pgxpool.Pool) (*PostgresIssueActivityStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	store := &PostgresIssueActivityStore{pool: pool}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		options, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("configure issue activity origin Redis: %w", err)
		}
		store.origin = redis.NewClient(options)
	}
	return store, nil
}

func (store *PostgresIssueActivityStore) Close() error {
	if store == nil || store.origin == nil {
		return nil
	}
	return store.origin.Close()
}

func (store *PostgresIssueActivityStore) RecordIssueActivity(ctx context.Context, eventID string, input IssueActivityInput) ([]string, error) {
	if err := validateIssueActivityInput(eventID, input); err != nil {
		return nil, err
	}
	projectID, _ := uuid.Parse(input.ProjectID)
	eventUUID, _ := uuid.Parse(eventID)
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin issue activity transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var workspaceID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT workspace_id FROM projects WHERE id=$1 AND deleted_at IS NULL`, projectID).Scan(&workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: project does not exist", ErrPermanentIssueActivity)
	}
	if err != nil {
		return nil, fmt.Errorf("load project for issue activity: %w", err)
	}
	// Schema migrations lock the workspace before changing item rows.
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspaceID); err != nil {
		return nil, fmt.Errorf("lock issue activity schema: %w", err)
	}
	if scopeErr := lockBoundIssueActivity(ctx, tx, input, workspaceID, true); scopeErr != nil {
		return nil, scopeErr
	}
	if input.Origin != "" && input.IssueID != nil {
		if store.origin == nil {
			return nil, errors.New("REDIS_URL is required to preserve issue activity origin")
		}
		if originErr := store.origin.Set(ctx, *input.IssueID, input.Origin, 10*time.Minute).Err(); originErr != nil {
			return nil, fmt.Errorf("set issue request origin: %w", originErr)
		}
	}
	rows, err := buildIssueActivityRows(ctx, tx, input)
	if err != nil {
		return nil, err
	}
	if input.IssueID != nil {
		if _, err := tx.Exec(ctx, `UPDATE issues SET updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, *input.IssueID); err != nil {
			return nil, fmt.Errorf("update issue activity timestamp: %w", err)
		}
	}
	requested, current := decodeActivityObject(input.RequestedData), decodeActivityObject(input.CurrentInstance)
	if input.Type == "issue.activity.updated" || input.Type == "issue.activity.created" {
		requestedAssignees, currentAssignees := activityIDSet(requested, "assignee_ids", "assignees"), activityIDSet(current, "assignee_ids", "assignees")
		for _, id := range sortedActivityIDs(requestedAssignees) {
			if _, exists := currentAssignees[id]; exists {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO issue_subscribers (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,project_id,workspace_id,issue_id,subscriber_id) VALUES ($1,NOW(),NOW(),NULL,$2,$2,$3,$4,$5,$2) ON CONFLICT (issue_id,subscriber_id) WHERE deleted_at IS NULL DO NOTHING`, uuid.New(), id, input.ProjectID, workspaceID, *input.IssueID); err != nil {
				return nil, fmt.Errorf("subscribe newly assigned issue user: %w", err)
			}
		}
	}
	activityIDs := make([]string, 0, len(rows))
	for index, row := range rows {
		id := uuid.NewSHA1(eventUUID, []byte(fmt.Sprintf("plane.issue_activity:%d", index)))
		activityIDs = append(activityIDs, id.String())
		var issueID, commentID, actorID, oldID, newID any
		if row.IssueID != nil {
			issueID = *row.IssueID
		}
		if row.CommentID != nil {
			commentID = *row.CommentID
		}
		if row.ActorID != nil {
			actorID = *row.ActorID
		}
		if row.OldIdentifier != nil {
			oldID = *row.OldIdentifier
		}
		if row.NewIdentifier != nil {
			newID = *row.NewIdentifier
		}
		if _, err := tx.Exec(ctx, `INSERT INTO issue_activities (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,issue_id,issue_comment_id,actor_id,verb,field,old_value,new_value,comment,attachments,old_identifier,new_identifier,epoch) VALUES ($1,COALESCE($15::timestamptz,NOW()),NOW(),NULL,NULL,NULL,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10,$11,ARRAY[]::text[],$12,$13,NULLIF($14,0)) ON CONFLICT (id) DO NOTHING`, id, workspaceID, projectID, issueID, commentID, actorID, row.Verb, row.Field, row.OldValue, row.NewValue, row.Comment, oldID, newID, row.Epoch, row.CreatedAt); err != nil {
			return nil, fmt.Errorf("insert issue activity row: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit issue activity transaction: %w", err)
	}
	return activityIDs, nil
}

func buildIssueActivityRows(ctx context.Context, tx pgx.Tx, input IssueActivityInput) ([]issueActivityRow, error) {
	event := issueActivityEvent{IssueActivityInput: input, requested: decodeActivityObject(input.RequestedData), current: decodeActivityObject(input.CurrentInstance)}
	switch input.Type {
	case "workitem_relation.activity.created", "workitem_relation.activity.deleted":
		return relationActivityRows(input)
	case "work_log.activity.created", "work_log.activity.updated", "work_log.activity.deleted":
		return workLogActivityRows(input)
	case "issue.activity.created":
		return createdIssueActivityRows(ctx, tx, event)
	case "issue.activity.updated":
		return updatedIssueActivityRows(ctx, tx, event)
	case "cycle.activity.created", "cycle.activity.deleted":
		return cycleActivityRows(ctx, tx, event)
	case "module.activity.created", "module.activity.deleted":
		return moduleActivityRows(ctx, tx, event)
	case "issue_relation.activity.created", "issue_relation.activity.deleted":
		return legacyRelationActivityRows(ctx, tx, event)
	case "comment.activity.created", "comment.activity.updated", "comment.activity.deleted":
		return commentActivityRows(event), nil
	case "link.activity.created", "link.activity.updated", "link.activity.deleted":
		return linkActivityRows(event), nil
	case "attachment.activity.created", "attachment.activity.deleted":
		return attachmentActivityRows(event), nil
	case "issue_reaction.activity.created", "comment_reaction.activity.created", "issue_reaction.activity.deleted", "comment_reaction.activity.deleted":
		return reactionActivityRows(ctx, tx, event)
	case "issue_vote.activity.created", "issue_vote.activity.deleted":
		return voteActivityRows(event), nil
	case "issue_draft.activity.created", "issue_draft.activity.updated", "issue_draft.activity.deleted":
		return draftActivityRows(event), nil
	case "intake.activity.created":
		return intakeActivityRows(event), nil
	case "issue.activity.deleted":
		row := event.baseRow()
		row.Verb, row.Field, row.Comment = "deleted", "issue", "deleted the issue"
		return []issueActivityRow{row}, nil
	}
	return []issueActivityRow{event.baseRow()}, nil
}

func decodeActivityObject(raw json.RawMessage) map[string]json.RawMessage {
	var object map[string]json.RawMessage
	if len(raw) > 0 && string(raw) != "null" {
		_ = json.Unmarshal(raw, &object)
	}
	return object
}

func sameActivityJSON(left, right json.RawMessage) bool {
	if len(left) == 0 {
		left = []byte("null")
	}
	if len(right) == 0 {
		right = []byte("null")
	}
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return fmt.Sprint(leftValue) == fmt.Sprint(rightValue)
}

func activityText(raw json.RawMessage) *string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	text := fmt.Sprint(value)
	return &text
}

func firstActivityValue(values map[string]json.RawMessage, keys ...string) (json.RawMessage, bool) {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func firstActivityValueRaw(values map[string]json.RawMessage, keys ...string) json.RawMessage {
	value, _ := firstActivityValue(values, keys...)
	return value
}

func nullableUUIDString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func issueActivityIdentifier(ctx context.Context, tx pgx.Tx, id *string) (*string, error) {
	if id == nil || *id == "" {
		return stringPointerValue(""), nil
	}
	var value string
	err := tx.QueryRow(ctx, `SELECT project.identifier||'-'||issues.sequence_id::text FROM issues JOIN projects project ON project.id=issues.project_id WHERE issues.id=$1`, *id).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load related issue identifier: %w", err)
	}
	return &value, nil
}

func issueActivityStateName(ctx context.Context, tx pgx.Tx, projectID string, id *string) (*string, error) {
	if id == nil || *id == "" {
		return nil, nil
	}
	var name string
	err := tx.QueryRow(ctx, `SELECT name FROM states WHERE id=$1 AND project_id=$2`, *id, projectID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load issue state name: %w", err)
	}
	return &name, nil
}

func issueActivityRelatedName(ctx context.Context, tx pgx.Tx, table, idColumn, nameColumn, id string) (*string, error) {
	if table != "labels" && table != "users" {
		return nil, fmt.Errorf("%w: unsupported activity lookup", ErrPermanentIssueActivity)
	}
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s=$1`, nameColumn, table, idColumn)
	var value string
	err := tx.QueryRow(ctx, query, id).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: activity reference %s does not exist", ErrPermanentIssueActivity, id)
	}
	if err != nil {
		return nil, fmt.Errorf("load issue activity reference: %w", err)
	}
	return &value, nil
}

func activityIDSet(values map[string]json.RawMessage, keys ...string) map[string]struct{} {
	raw, ok := firstActivityValue(values, keys...)
	if !ok || string(raw) == "null" {
		return map[string]struct{}{}
	}
	var ids []string
	if json.Unmarshal(raw, &ids) != nil {
		return map[string]struct{}{}
	}
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, err := uuid.Parse(id); err == nil {
			set[id] = struct{}{}
		}
	}
	return set
}

func sortedActivityIDs(ids map[string]struct{}) []string {
	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func rawStringArray(raw json.RawMessage) []string {
	var values []string
	_ = json.Unmarshal(raw, &values)
	return values
}

func rawObjectArray(raw json.RawMessage) []map[string]json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var encoded string
	if json.Unmarshal(raw, &encoded) == nil {
		raw = []byte(encoded)
	}
	var values []map[string]json.RawMessage
	_ = json.Unmarshal(raw, &values)
	return values
}

func lookupActivityName(ctx context.Context, tx pgx.Tx, table string, id *string) (*string, error) {
	if id == nil || *id == "" {
		return nil, nil
	}
	if table != "cycles" && table != "modules" {
		return nil, fmt.Errorf("%w: unsupported activity lookup", ErrPermanentIssueActivity)
	}
	var value string
	err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT name FROM %s WHERE id=$1`, table), *id).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load %s activity name: %w", table, err)
	}
	return &value, nil
}

func inverseActivityRelation(relation string) string {
	switch relation {
	case "start_after":
		return "start_before"
	case "finish_after":
		return "finish_before"
	case "blocked_by":
		return "blocking"
	case "blocking":
		return "blocked_by"
	case "start_before":
		return "start_after"
	case "finish_before":
		return "finish_after"
	case "implemented_by":
		return "implements"
	case "implements":
		return "implemented_by"
	default:
		return relation
	}
}

func stringPointerValue(value string) *string { return &value }

func activityString(raw json.RawMessage) *string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return &s
	}
	return nil
}
func stringOrEmpty(value *string) *string {
	if value == nil {
		empty := ""
		return &empty
	}
	return value
}
func intakeStatus(value *string) *string {
	if value == nil {
		return nil
	}
	var status int
	if _, err := fmt.Sscan(*value, &status); err != nil {
		return nil
	}
	mapped := map[int]string{-2: "Pending", -1: "Rejected", 0: "Snoozed", 1: "Accepted", 2: "Duplicate"}
	text, ok := mapped[status]
	if !ok {
		return nil
	}
	return &text
}
func findActivityReactionID(ctx context.Context, tx pgx.Tx, projectID string, actorID *string, reaction string) (*string, error) {
	if actorID == nil {
		return nil, nil
	}
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM issue_reactions WHERE reaction=$1 AND project_id=$2 AND actor_id=$3 AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1`, reaction, projectID, *actorID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load issue reaction: %w", err)
	}
	return &id, nil
}
func findCommentReaction(ctx context.Context, tx pgx.Tx, projectID string, actorID *string, reaction string) (*string, *string, error) {
	if actorID == nil {
		return nil, nil, nil
	}
	var id, commentID string
	err := tx.QueryRow(ctx, `SELECT id,comment_id FROM comment_reactions WHERE reaction=$1 AND project_id=$2 AND actor_id=$3 AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1`, reaction, projectID, *actorID).Scan(&id, &commentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load comment reaction: %w", err)
	}
	return &id, &commentID, nil
}
func classifyIssueActivityError(err error) error {
	if errors.Is(err, ErrPermanentIssueActivity) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return fmt.Errorf("%w: invalid activity reference", ErrPermanentIssueActivity)
	}
	return err
}
