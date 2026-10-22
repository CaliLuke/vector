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
	return nil
}

func supportedIssueActivityType(activityType string) bool {
	switch activityType {
	case "issue.activity.created", "issue.activity.deleted", "issue.activity.updated", "cycle.activity.created", "cycle.activity.deleted", "module.activity.created", "module.activity.deleted", "issue_relation.activity.created", "issue_relation.activity.deleted", "comment.activity.created", "comment.activity.updated", "comment.activity.deleted",
		"link.activity.created", "link.activity.updated", "link.activity.deleted",
		"attachment.activity.created", "attachment.activity.deleted",
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
	if input.Origin != "" && input.IssueID != nil {
		if store.origin == nil {
			return nil, errors.New("REDIS_URL is required to preserve issue activity origin")
		}
		if err := store.origin.Set(ctx, *input.IssueID, input.Origin, 10*time.Minute).Err(); err != nil {
			return nil, fmt.Errorf("set issue request origin: %w", err)
		}
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
	requested := decodeActivityObject(input.RequestedData)
	current := decodeActivityObject(input.CurrentInstance)
	row := issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Epoch: input.Epoch}
	get := func(data map[string]json.RawMessage, key string) *string { return activityString(data[key]) }
	switch input.Type {
	case "issue.activity.created":
		var createdAt time.Time
		var creatorID *string
		if err := tx.QueryRow(ctx, `SELECT created_at,created_by_id FROM issues WHERE id=$1`, *input.IssueID).Scan(&createdAt, &creatorID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: issue does not exist", ErrPermanentIssueActivity)
			}
			return nil, fmt.Errorf("load issue creation metadata: %w", err)
		}
		rows := []issueActivityRow{{IssueID: input.IssueID, ActorID: creatorID, Verb: "created", Comment: "created the issue", Epoch: input.Epoch, CreatedAt: &createdAt}}
		if _, ok := requested["assignee_ids"]; ok {
			requestedAssignees, currentAssignees := activityIDSet(requested, "assignee_ids"), activityIDSet(current, "assignee_ids")
			for _, id := range sortedActivityIDs(requestedAssignees) {
				if _, exists := currentAssignees[id]; exists {
					continue
				}
				name, err := issueActivityRelatedName(ctx, tx, "users", "id", "display_name", id)
				if err != nil {
					return nil, err
				}
				rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "assignees", OldValue: stringPointerValue(""), NewValue: name, Comment: "added assignee ", NewIdentifier: stringPointerValue(id), Epoch: input.Epoch})
			}
		}
		return rows, nil
	case "issue.activity.updated":
		rows := make([]issueActivityRow, 0, 12)
		for _, field := range []struct{ key, column, comment string }{
			{"name", "name", "updated the name to"},
			{"priority", "priority", "updated the priority to"},
			{"target_date", "target_date", "updated the target date to"},
			{"start_date", "start_date", "updated the start date to "},
			{"description_html", "description", "updated the description to"},
		} {
			requestedValue, requestedOK := requested[field.key]
			if !requestedOK || sameActivityJSON(current[field.key], requestedValue) {
				continue
			}
			oldValue := activityText(current[field.key])
			newValue := activityText(requestedValue)
			if field.key == "description_html" {
				var lastID string
				err := tx.QueryRow(ctx, `SELECT id FROM issue_activities WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1`, *input.IssueID).Scan(&lastID)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					return nil, fmt.Errorf("load last issue activity: %w", err)
				}
				if err == nil {
					var sameActor bool
					if err := tx.QueryRow(ctx, `SELECT field='description' AND actor_id IS NOT DISTINCT FROM $2::uuid FROM issue_activities WHERE id=$1`, lastID, nullableUUIDString(input.ActorID)).Scan(&sameActor); err != nil {
						return nil, fmt.Errorf("inspect last description activity: %w", err)
					}
					if sameActor {
						if _, err := tx.Exec(ctx, `UPDATE issue_activities SET created_at=NOW(),updated_at=NOW() WHERE id=$1`, lastID); err != nil {
							return nil, fmt.Errorf("refresh description activity timestamp: %w", err)
						}
						continue
					}
				}
			}
			rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: field.column, OldValue: oldValue, NewValue: newValue, Comment: field.comment, Epoch: input.Epoch})
		}
		if requestedParent, ok := firstActivityValue(requested, "parent_id", "parent"); ok && !sameActivityJSON(firstActivityValueRaw(current, "parent_id", "parent"), requestedParent) {
			oldID, newID := activityText(firstActivityValueRaw(current, "parent_id", "parent")), activityText(requestedParent)
			oldValue, err := issueActivityIdentifier(ctx, tx, oldID)
			if err != nil {
				return nil, err
			}
			newValue, err := issueActivityIdentifier(ctx, tx, newID)
			if err != nil {
				return nil, err
			}
			rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "parent", OldValue: oldValue, NewValue: newValue, Comment: "updated the parent issue to", OldIdentifier: oldID, NewIdentifier: newID, Epoch: input.Epoch})
		}
		if requestedState, ok := firstActivityValue(requested, "state_id", "state"); ok && !sameActivityJSON(firstActivityValueRaw(current, "state_id", "state"), requestedState) {
			oldID, newID := activityText(firstActivityValueRaw(current, "state_id", "state")), activityText(requestedState)
			oldValue, err := issueActivityStateName(ctx, tx, input.ProjectID, oldID)
			if err != nil {
				return nil, err
			}
			newValue, err := issueActivityStateName(ctx, tx, input.ProjectID, newID)
			if err != nil {
				return nil, err
			}
			rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "state", OldValue: oldValue, NewValue: newValue, Comment: "updated the state to", OldIdentifier: oldID, NewIdentifier: newID, Epoch: input.Epoch})
		}
		if requestedPriority, ok := requested["closed_to"]; ok && string(requestedPriority) != "null" {
			newID := activityText(requestedPriority)
			newValue, err := issueActivityStateName(ctx, tx, input.ProjectID, newID)
			if err != nil {
				return nil, err
			}
			rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "state", NewValue: newValue, Comment: "Plane updated the state to ", NewIdentifier: newID, Epoch: input.Epoch})
		}
		if requestedArchivedAt, ok := requested["archived_at"]; ok && !sameActivityJSON(current["archived_at"], requestedArchivedAt) {
			if string(requestedArchivedAt) == "null" {
				rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "archived_at", OldValue: stringPointerValue("archive"), NewValue: stringPointerValue("restore"), Comment: "has restored the issue", Epoch: input.Epoch})
			} else if string(requested["automation"]) == "true" {
				rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "archived_at", NewValue: stringPointerValue("archive"), Comment: "Plane has archived the issue", Epoch: input.Epoch})
			} else {
				rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "archived_at", NewValue: stringPointerValue("manual_archive"), Comment: "Actor has archived the issue", Epoch: input.Epoch})
			}
		}
		for _, collection := range []struct {
			keys                                                          []string
			table, idColumn, nameColumn, field, addComment, removeComment string
		}{
			{[]string{"label_ids", "labels"}, "labels", "id", "name", "labels", "added label ", "removed label "},
			{[]string{"assignee_ids", "assignees"}, "users", "id", "display_name", "assignees", "added assignee ", "removed assignee "},
		} {
			requestedSet := activityIDSet(requested, collection.keys...)
			currentSet := activityIDSet(current, collection.keys...)
			for _, id := range sortedActivityIDs(requestedSet) {
				if _, exists := currentSet[id]; exists {
					continue
				}
				name, err := issueActivityRelatedName(ctx, tx, collection.table, collection.idColumn, collection.nameColumn, id)
				if err != nil {
					return nil, err
				}
				rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: collection.field, OldValue: stringPointerValue(""), NewValue: name, Comment: collection.addComment, NewIdentifier: stringPointerValue(id), Epoch: input.Epoch})
			}
			for _, id := range sortedActivityIDs(currentSet) {
				if _, exists := requestedSet[id]; exists {
					continue
				}
				name, err := issueActivityRelatedName(ctx, tx, collection.table, collection.idColumn, collection.nameColumn, id)
				if err != nil {
					return nil, err
				}
				rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: collection.field, OldValue: name, NewValue: stringPointerValue(""), Comment: collection.removeComment, OldIdentifier: stringPointerValue(id), Epoch: input.Epoch})
			}
		}
		return rows, nil
	case "issue.activity.deleted":
		row.Verb, row.Field, row.Comment = "deleted", "issue", "deleted the issue"
	case "cycle.activity.created":
		rows := make([]issueActivityRow, 0)
		for _, record := range rawObjectArray(current["updated_cycle_issues"]) {
			issueID := activityText(record["issue_id"])
			oldID := activityText(record["old_cycle_id"])
			newID := activityText(record["new_cycle_id"])
			oldName, err := lookupActivityName(ctx, tx, "cycles", oldID)
			if err != nil {
				return nil, err
			}
			newName, err := lookupActivityName(ctx, tx, "cycles", newID)
			if err != nil {
				return nil, err
			}
			oldText, newText := stringOrEmpty(oldName), stringOrEmpty(newName)
			rows = append(rows, issueActivityRow{IssueID: issueID, ActorID: input.ActorID, Verb: "updated", Field: "cycles", OldValue: oldText, NewValue: newText, Comment: fmt.Sprintf("updated cycle from %s\n                to %s", *oldText, *newText), OldIdentifier: oldID, NewIdentifier: newID, Epoch: input.Epoch})
		}
		for _, record := range rawObjectArray(current["created_cycle_issues"]) {
			fields := decodeActivityObject(record["fields"])
			issueID := activityText(fields["issue"])
			cycleID := activityText(fields["cycle"])
			name, err := lookupActivityName(ctx, tx, "cycles", cycleID)
			if err != nil {
				return nil, err
			}
			name = stringOrEmpty(name)
			rows = append(rows, issueActivityRow{IssueID: issueID, ActorID: input.ActorID, Verb: "created", Field: "cycles", OldValue: stringPointerValue(""), NewValue: name, Comment: "added cycle " + *name, NewIdentifier: cycleID, Epoch: input.Epoch})
		}
		return rows, nil
	case "cycle.activity.deleted":
		cycleID := activityText(requested["cycle_id"])
		cycleName, err := lookupActivityName(ctx, tx, "cycles", cycleID)
		if err != nil {
			return nil, err
		}
		if cycleName == nil {
			cycleName = stringOrEmpty(activityText(requested["cycle_name"]))
		}
		rows := make([]issueActivityRow, 0)
		for _, issue := range rawStringArray(requested["issues"]) {
			name := stringOrEmpty(cycleName)
			rows = append(rows, issueActivityRow{IssueID: &issue, ActorID: input.ActorID, Verb: "deleted", Field: "cycles", OldValue: name, NewValue: stringPointerValue(""), Comment: "removed this issue from " + *name, OldIdentifier: cycleID, Epoch: input.Epoch})
		}
		return rows, nil
	case "module.activity.created":
		moduleID := activityText(requested["module_id"])
		name, err := lookupActivityName(ctx, tx, "modules", moduleID)
		if err != nil {
			return nil, err
		}
		name = stringOrEmpty(name)
		row.Verb, row.Field, row.OldValue, row.NewValue = "created", "modules", stringPointerValue(""), name
		row.Comment = "added module " + *name
		row.NewIdentifier = moduleID
	case "module.activity.deleted":
		name := stringOrEmpty(activityText(current["module_name"]))
		row.Verb, row.Field, row.OldValue, row.NewValue = "deleted", "modules", name, stringPointerValue("")
		row.Comment = "removed this issue from " + *name
		row.OldIdentifier = activityText(requested["module_id"])
	case "issue_relation.activity.created":
		if len(input.CurrentInstance) == 0 || string(input.CurrentInstance) == "null" {
			relation := stringOrEmpty(activityText(requested["relation_type"]))
			rows := make([]issueActivityRow, 0)
			for _, relatedID := range rawStringArray(requested["issues"]) {
				identifier, err := issueActivityIdentifier(ctx, tx, &relatedID)
				if err != nil {
					return nil, err
				}
				rows = append(rows, issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: *relation, OldValue: stringPointerValue(""), NewValue: identifier, Comment: "added " + *relation + " relation", OldIdentifier: &relatedID, Epoch: input.Epoch})
				currentIdentifier, err := issueActivityIdentifier(ctx, tx, input.IssueID)
				if err != nil {
					return nil, err
				}
				inverse := inverseActivityRelation(*relation)
				rows = append(rows, issueActivityRow{IssueID: &relatedID, ActorID: input.ActorID, Verb: "updated", Field: inverse, OldValue: stringPointerValue(""), NewValue: currentIdentifier, Comment: "added " + inverse + " relation", OldIdentifier: input.IssueID, Epoch: input.Epoch})
			}
			return rows, nil
		}
		return nil, nil
	case "issue_relation.activity.deleted":
		relation := stringOrEmpty(activityText(requested["relation_type"]))
		relatedID := activityText(requested["related_issue"])
		identifier, err := issueActivityIdentifier(ctx, tx, relatedID)
		if err != nil {
			return nil, err
		}
		row.Verb, row.Field, row.OldValue, row.NewValue = "deleted", *relation, identifier, stringPointerValue("")
		row.Comment = "deleted " + *relation + " relation"
		row.OldIdentifier = relatedID
		currentIdentifier, err := issueActivityIdentifier(ctx, tx, input.IssueID)
		if err != nil {
			return nil, err
		}
		inverse := inverseActivityRelation(*relation)
		return []issueActivityRow{row, {IssueID: relatedID, ActorID: input.ActorID, Verb: "deleted", Field: inverse, OldValue: currentIdentifier, NewValue: stringPointerValue(""), Comment: "deleted " + *relation + " relation", OldIdentifier: relatedID, Epoch: input.Epoch}}, nil
	case "comment.activity.created":
		row.Verb, row.Field, row.Comment, row.NewValue = "created", "comment", "created a comment", stringOrEmpty(get(requested, "comment_html"))
		row.CommentID = get(requested, "id")
		row.NewIdentifier = row.CommentID
	case "comment.activity.updated":
		oldValue, newValue := stringOrEmpty(get(current, "comment_html")), stringOrEmpty(get(requested, "comment_html"))
		if oldValue == newValue {
			return nil, nil
		}
		row.Verb, row.Field, row.Comment, row.OldValue, row.NewValue = "updated", "comment", "updated a comment", oldValue, newValue
		row.CommentID = get(current, "id")
		row.OldIdentifier = row.CommentID
		row.NewIdentifier = row.CommentID
	case "comment.activity.deleted":
		row.Verb, row.Field, row.Comment = "deleted", "comment", "deleted the comment"
		row.CommentID = get(requested, "comment_id")
	case "link.activity.created":
		row.Verb, row.Field, row.Comment, row.NewValue = "created", "link", "created a link", stringOrEmpty(get(requested, "url"))
		row.NewIdentifier = get(requested, "id")
	case "link.activity.updated":
		oldValue, newValue := stringOrEmpty(get(current, "url")), stringOrEmpty(get(requested, "url"))
		if oldValue == newValue {
			return nil, nil
		}
		row.Verb, row.Field, row.Comment, row.OldValue, row.NewValue = "updated", "link", "updated a link", oldValue, newValue
		row.OldIdentifier = get(current, "id")
		row.NewIdentifier = get(current, "id")
	case "link.activity.deleted":
		oldValue := stringOrEmpty(get(current, "url"))
		row.Verb, row.Field, row.Comment, row.OldValue = "deleted", "link", "deleted the link", oldValue
		empty := ""
		row.NewValue = &empty
	case "attachment.activity.created":
		row.Verb, row.Field, row.Comment = "created", "attachment", "created an attachment"
		row.NewValue = get(current, "asset")
		row.NewIdentifier = get(current, "id")
	case "attachment.activity.deleted":
		row.Verb, row.Field, row.Comment = "deleted", "attachment", "deleted the attachment"
	case "issue_reaction.activity.created", "comment_reaction.activity.created":
		reaction := get(requested, "reaction")
		if reaction == nil {
			return nil, nil
		}
		row.Verb, row.Field, row.Comment, row.NewValue = "created", "reaction", "added the reaction", reaction
		if input.Type == "issue_reaction.activity.created" {
			var err error
			row.NewIdentifier, err = findActivityReactionID(ctx, tx, input.ProjectID, input.ActorID, *reaction)
			if err != nil {
				return nil, err
			}
		} else {
			var commentID *string
			var err error
			row.NewIdentifier, commentID, err = findCommentReaction(ctx, tx, input.ProjectID, input.ActorID, *reaction)
			if err != nil {
				return nil, err
			}
			if commentID != nil {
				var issueID string
				if err := tx.QueryRow(ctx, `SELECT issue_id FROM issue_comments WHERE id=$1 AND project_id=$2 AND deleted_at IS NULL`, *commentID, input.ProjectID).Scan(&issueID); err == nil {
					row.IssueID = &issueID
				} else if !errors.Is(err, pgx.ErrNoRows) {
					return nil, fmt.Errorf("load comment reaction issue: %w", err)
				}
			}
		}
		if row.NewIdentifier == nil {
			return nil, nil
		}
	case "issue_reaction.activity.deleted", "comment_reaction.activity.deleted":
		reaction := get(current, "reaction")
		if reaction == nil {
			return nil, nil
		}
		id := get(current, "identifier")
		row.Verb, row.Field, row.Comment, row.OldValue, row.OldIdentifier = "deleted", "reaction", "removed the reaction", reaction, id
		if input.Type == "comment_reaction.activity.deleted" {
			if commentID := get(current, "comment_id"); commentID != nil {
				var issueID string
				if err := tx.QueryRow(ctx, `SELECT issue_id FROM issue_comments WHERE id=$1 AND project_id=$2 AND deleted_at IS NULL`, *commentID, input.ProjectID).Scan(&issueID); err == nil {
					row.IssueID = &issueID
				} else if !errors.Is(err, pgx.ErrNoRows) {
					return nil, fmt.Errorf("load comment reaction issue: %w", err)
				}
			}
		}
	case "issue_vote.activity.created":
		if vote := get(requested, "vote"); vote != nil {
			row.Verb, row.Field, row.Comment, row.NewValue = "updated", "vote", "added the vote", vote
		} else {
			return nil, nil
		}
	case "issue_vote.activity.deleted":
		if vote := get(current, "vote"); vote != nil {
			row.Verb, row.Field, row.Comment, row.OldValue = "deleted", "vote", "removed the vote", vote
			row.OldIdentifier = get(current, "identifier")
		} else {
			return nil, nil
		}
	case "issue_draft.activity.created":
		row.Verb, row.Field, row.Comment = "created", "draft", "drafted the issue"
	case "issue_draft.activity.updated":
		row.Verb = "updated"
		if raw := requested["is_draft"]; len(raw) > 0 && string(raw) == "false" {
			row.Comment = "created the issue"
		} else {
			row.Field, row.Comment = "draft", "updated the draft issue"
		}
	case "issue_draft.activity.deleted":
		row.Verb, row.Field, row.Comment = "deleted", "draft", "deleted the draft issue"
		row.IssueID = nil
	case "intake.activity.created":
		status := get(requested, "status")
		if status == nil {
			return nil, nil
		}
		row.Verb, row.Field, row.Comment = "updated", "intake", "updated the intake status"
		row.Verb = *status
		row.OldValue = intakeStatus(get(current, "status"))
		row.NewValue = intakeStatus(status)
	}
	return []issueActivityRow{row}, nil
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
