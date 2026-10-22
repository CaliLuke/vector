package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
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
	issueVersionTaskWorkflowName = "plane.issues.record_version"
	issueVersionTaskActivityName = "plane.issues.write_version"
	issueVersionTaskQueue        = "plane-background"
	issueVersionTaskTimeout      = 30 * time.Second
)

var ErrPermanentIssueVersionTask = errors.New("permanent issue version task failure")

type IssueVersionTaskInput struct {
	UpdatedIssue string `json:"updated_issue"`
	IssueID      string `json:"issue_id"`
	UserID       string `json:"user_id"`
}

type issueVersionTaskStore interface {
	TrackIssueVersion(context.Context, IssueVersionTaskInput) error
}

type IssueVersionTaskDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[IssueVersionTaskInput, struct{}]
}

func RegisterIssueVersionTaskWorkflow(app *selvedge.App, store issueVersionTaskStore, policy *retry.Policy) (*IssueVersionTaskDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and issue version task store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentIssueVersionTask)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[IssueVersionTaskInput, struct{}]{Name: issueVersionTaskActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input IssueVersionTaskInput) (struct{}, error) {
		if err := validateIssueVersionTaskInput(input); err != nil {
			return struct{}{}, err
		}
		if err := store.TrackIssueVersion(ctx, input); err != nil {
			return struct{}{}, classifyIssueVersionTaskError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueVersionTaskQueue, StartToCloseTimeout: issueVersionTaskTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue version activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(issueVersionTaskWorkflowName, func(ctx *selvedge.WorkflowContext, input IssueVersionTaskInput) (struct{}, error) {
		if err := validateIssueVersionTaskInput(input); err != nil {
			return struct{}{}, err
		}
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &IssueVersionTaskDispatcher{app: app, workflow: workflow}, nil
}

func (dispatcher *IssueVersionTaskDispatcher) Admit(ctx context.Context, eventID string, input IssueVersionTaskInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable issue version dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("issue version event ID must be a UUID: %w", err)
	}
	if err := validateIssueVersionTaskInput(input); err != nil {
		return "", err
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "issue-version:"+eventID, input)
}

func validateIssueVersionTaskInput(input IssueVersionTaskInput) error {
	if _, err := uuid.Parse(input.IssueID); err != nil {
		return fmt.Errorf("%w: invalid issue ID", ErrPermanentIssueVersionTask)
	}
	if _, err := uuid.Parse(input.UserID); err != nil {
		return fmt.Errorf("%w: invalid owner ID", ErrPermanentIssueVersionTask)
	}
	if strings.TrimSpace(input.UpdatedIssue) != "" {
		var snapshot map[string]json.RawMessage
		if err := json.Unmarshal([]byte(input.UpdatedIssue), &snapshot); err != nil || snapshot == nil {
			return fmt.Errorf("%w: updated issue must be a JSON object", ErrPermanentIssueVersionTask)
		}
	}
	return nil
}

type issueVersionField struct {
	issueColumn   string
	versionColumn string
	cast          string
}

var issueVersionFields = map[string]issueVersionField{
	"parent":            {"parent_id", "parent", "uuid"},
	"parent_id":         {"parent_id", "parent", "uuid"},
	"state":             {"state_id", "state", "uuid"},
	"state_id":          {"state_id", "state", "uuid"},
	"estimate_point":    {"estimate_point_id", "estimate_point", "uuid"},
	"estimate_point_id": {"estimate_point_id", "estimate_point", "uuid"},
	"name":              {"name", "name", "text"},
	"priority":          {"priority", "priority", "varchar"},
	"start_date":        {"start_date", "start_date", "date"},
	"target_date":       {"target_date", "target_date", "date"},
	"sequence_id":       {"sequence_id", "sequence_id", "integer"},
	"sort_order":        {"sort_order", "sort_order", "double precision"},
	"completed_at":      {"completed_at", "completed_at", "timestamptz"},
	"archived_at":       {"archived_at", "archived_at", "date"},
	"is_draft":          {"is_draft", "is_draft", "boolean"},
	"external_source":   {"external_source", "external_source", "varchar"},
	"external_id":       {"external_id", "external_id", "varchar"},
	"type":              {"type_id", "type", "uuid"},
	"type_id":           {"type_id", "type", "uuid"},
}

type PostgresIssueVersionTaskStore struct{ pool *pgxpool.Pool }

func NewPostgresIssueVersionTaskStore(pool *pgxpool.Pool) (*PostgresIssueVersionTaskStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresIssueVersionTaskStore{pool: pool}, nil
}

func (store *PostgresIssueVersionTaskStore) TrackIssueVersion(ctx context.Context, input IssueVersionTaskInput) error {
	if err := validateIssueVersionTaskInput(input); err != nil {
		return err
	}
	issueID, _ := uuid.Parse(input.IssueID)
	userID, _ := uuid.Parse(input.UserID)
	var requested map[string]json.RawMessage
	if input.UpdatedIssue != "" {
		if err := json.Unmarshal([]byte(input.UpdatedIssue), &requested); err != nil {
			return fmt.Errorf("%w: decode updated issue snapshot: %v", ErrPermanentIssueVersionTask, err)
		}
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin issue version transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var currentJSON []byte
	err = tx.QueryRow(ctx, `SELECT to_jsonb(issue) FROM issues issue WHERE issue.id=$1 AND issue.deleted_at IS NULL FOR UPDATE`, issueID).Scan(&currentJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("load issue for version task: %w", err)
	}
	var current map[string]json.RawMessage
	if err := json.Unmarshal(currentJSON, &current); err != nil {
		return fmt.Errorf("decode current issue for version task: %w", err)
	}
	changed := make([]issueVersionChangedField, 0, len(requested))
	for key, value := range requested {
		field, ok := issueVersionFields[key]
		if !ok {
			return fmt.Errorf("%w: unsupported issue version field %q", ErrPermanentIssueVersionTask, key)
		}
		prior := current[field.issueColumn]
		if sameJSONValue(prior, value) {
			continue
		}
		var parsed any
		if string(value) != "null" {
			if err := json.Unmarshal(value, &parsed); err != nil {
				return fmt.Errorf("%w: invalid issue version field %q", ErrPermanentIssueVersionTask, key)
			}
		}
		changed = append(changed, issueVersionChangedField{field: field, value: parsed})
	}
	if len(changed) == 0 {
		return tx.Commit(ctx)
	}
	now := time.Now().UTC()
	var latestID, latestOwner uuid.UUID
	var lastSaved time.Time
	err = tx.QueryRow(ctx, `SELECT id,owned_by_id,last_saved_at FROM issue_versions WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY last_saved_at DESC LIMIT 1 FOR UPDATE`, issueID).Scan(&latestID, &latestOwner, &lastSaved)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load latest issue version: %w", err)
	}
	if err == nil && latestOwner == userID && now.Sub(lastSaved) <= versionCoalesceWindow {
		sets := make([]string, 0, len(changed)+1)
		args := make([]any, 0, len(changed)+2)
		for index, field := range changed {
			args = append(args, field.value)
			sets = append(sets, fmt.Sprintf("%s=$%d::%s", field.field.versionColumn, index+1, field.field.cast))
		}
		args = append(args, now)
		sets = append(sets, fmt.Sprintf("last_saved_at=$%d", len(args)))
		args = append(args, latestID)
		if _, err := tx.Exec(ctx, `UPDATE issue_versions SET `+strings.Join(sets, ",")+fmt.Sprintf(" WHERE id=$%d", len(args)), args...); err != nil {
			return fmt.Errorf("coalesce issue version: %w", err)
		}
		return tx.Commit(ctx)
	}
	if err := insertIssueVersionSnapshot(ctx, tx, issueID, userID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type issueVersionChangedField struct {
	field issueVersionField
	value any
}

func sameJSONValue(left, right json.RawMessage) bool {
	if len(left) == 0 {
		left = []byte("null")
	}
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func insertIssueVersionSnapshot(ctx context.Context, tx pgx.Tx, issueID, userID uuid.UUID, now time.Time) error {
	_, err := tx.Exec(ctx, `
INSERT INTO issue_versions (
 id,workspace_id,project_id,issue_id,created_by_id,updated_by_id,owned_by_id,activity_id,
 parent,state,estimate_point,name,priority,start_date,target_date,assignees,sequence_id,labels,
 sort_order,completed_at,archived_at,is_draft,external_source,external_id,type,cycle,modules,
 properties,meta,last_saved_at,created_at,updated_at,deleted_at
)
SELECT $1,issue.workspace_id,issue.project_id,issue.id,issue.created_by_id,issue.updated_by_id,$2,
 NULL,
 issue.parent_id,issue.state_id,issue.estimate_point_id,issue.name,issue.priority,issue.start_date,issue.target_date,
 COALESCE((SELECT array_agg(assignee.assignee_id) FROM issue_assignees assignee WHERE assignee.issue_id=issue.id AND assignee.deleted_at IS NULL),ARRAY[]::uuid[]),
 issue.sequence_id,
 COALESCE((SELECT array_agg(label.label_id) FROM issue_labels label WHERE label.issue_id=issue.id AND label.deleted_at IS NULL),ARRAY[]::uuid[]),
 issue.sort_order,issue.completed_at,issue.archived_at,issue.is_draft,issue.external_source,issue.external_id,issue.type_id,
 (SELECT cycle_issue.cycle_id FROM cycle_issues cycle_issue WHERE cycle_issue.issue_id=issue.id AND cycle_issue.deleted_at IS NULL ORDER BY cycle_issue.created_at DESC,cycle_issue.id DESC LIMIT 1),
 COALESCE((SELECT array_agg(module_issue.module_id) FROM module_issues module_issue JOIN modules module ON module.id=module_issue.module_id AND module.deleted_at IS NULL WHERE module_issue.issue_id=issue.id AND module_issue.deleted_at IS NULL),ARRAY[]::uuid[]),
 '{}'::jsonb,'{}'::jsonb,$3,$3,$3,NULL
FROM issues issue WHERE issue.id=$4 AND issue.deleted_at IS NULL`, uuid.New(), userID, now, issueID)
	if err != nil {
		return fmt.Errorf("insert issue version snapshot: %w", err)
	}
	return nil
}

func classifyIssueVersionTaskError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return fmt.Errorf("%w: invalid issue version reference", ErrPermanentIssueVersionTask)
	}
	return err
}
