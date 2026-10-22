package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
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
	pageVersionWorkflowName             = "plane.pages.track_version"
	pageVersionActivityName             = "plane.pages.write_version"
	issueDescriptionVersionWorkflowName = "plane.issues.track_description_version"
	issueDescriptionVersionActivityName = "plane.issues.write_description_version"
	versionHistoryQueue                 = "plane-background"
	versionHistoryActivityTimeout       = 30 * time.Second
	versionCoalesceWindow               = 10 * time.Minute
)

var ErrPermanentVersionHistory = errors.New("permanent version history task failure")

type PageVersionInput struct {
	PageID           string `json:"page_id"`
	ExistingInstance string `json:"existing_instance"`
	UserID           string `json:"user_id"`
}

type IssueDescriptionVersionInput struct {
	IssueID      string `json:"issue_id"`
	UpdatedIssue string `json:"updated_issue"`
	UserID       string `json:"user_id"`
	IsCreating   bool   `json:"is_creating"`
}

type versionStore interface {
	TrackPageVersion(context.Context, PageVersionInput) error
	TrackIssueDescriptionVersion(context.Context, IssueDescriptionVersionInput) error
}

type VersionHistoryDispatcher struct {
	app   *selvedge.App
	page  *selvedge.WorkflowFunc[PageVersionInput, struct{}]
	issue *selvedge.WorkflowFunc[IssueDescriptionVersionInput, struct{}]
}

func NewVersionHistoryDispatcher(app *selvedge.App, page *selvedge.WorkflowFunc[PageVersionInput, struct{}], issue *selvedge.WorkflowFunc[IssueDescriptionVersionInput, struct{}]) (*VersionHistoryDispatcher, error) {
	if app == nil || page == nil || issue == nil {
		return nil, errors.New("Selvedge app and both version history workflows are required")
	}
	return &VersionHistoryDispatcher{app: app, page: page, issue: issue}, nil
}

func RegisterVersionHistoryWorkflows(app *selvedge.App, store versionStore, policy *retry.Policy) (*VersionHistoryDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and version history store are required")
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
	pageActivity := execution.Activity[PageVersionInput, struct{}]{Name: pageVersionActivityName}
	if err := executionSelvedge.Register(activities, pageActivity, func(ctx context.Context, input PageVersionInput) (struct{}, error) {
		if _, _, err := parseVersionIdentity(input.PageID, input.UserID); err != nil {
			return struct{}{}, err
		}
		if err := store.TrackPageVersion(ctx, input); err != nil {
			return struct{}{}, classifyVersionHistoryError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: versionHistoryQueue, StartToCloseTimeout: versionHistoryActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register page version activity: %w", err)
	}
	issueActivity := execution.Activity[IssueDescriptionVersionInput, struct{}]{Name: issueDescriptionVersionActivityName}
	if err := executionSelvedge.Register(activities, issueActivity, func(ctx context.Context, input IssueDescriptionVersionInput) (struct{}, error) {
		if _, _, err := parseVersionIdentity(input.IssueID, input.UserID); err != nil {
			return struct{}{}, err
		}
		if err := store.TrackIssueDescriptionVersion(ctx, input); err != nil {
			return struct{}{}, classifyVersionHistoryError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: versionHistoryQueue, StartToCloseTimeout: versionHistoryActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue description version activity: %w", err)
	}
	pageWorkflow := selvedge.DefineWorkflow(pageVersionWorkflowName, func(ctx *selvedge.WorkflowContext, input PageVersionInput) (struct{}, error) {
		return pageActivity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	issueWorkflow := selvedge.DefineWorkflow(issueDescriptionVersionWorkflowName, func(ctx *selvedge.WorkflowContext, input IssueDescriptionVersionInput) (struct{}, error) {
		return issueActivity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, pageWorkflow)
	selvedge.RegisterWorkflow(app, issueWorkflow)
	return NewVersionHistoryDispatcher(app, pageWorkflow, issueWorkflow)
}

func (dispatcher *VersionHistoryDispatcher) AdmitPage(ctx context.Context, eventID string, input PageVersionInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.page == nil {
		return "", errors.New("durable page version dispatcher is not configured")
	}
	if _, _, err := parseVersionIdentity(input.PageID, input.UserID); err != nil {
		return "", err
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("page version event ID must be a UUID: %w", err)
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.page, "page-version:"+eventID, input)
}

func (dispatcher *VersionHistoryDispatcher) AdmitIssueDescription(ctx context.Context, eventID string, input IssueDescriptionVersionInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.issue == nil {
		return "", errors.New("durable issue description version dispatcher is not configured")
	}
	if _, _, err := parseVersionIdentity(input.IssueID, input.UserID); err != nil {
		return "", err
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("issue description version event ID must be a UUID: %w", err)
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.issue, "issue-description-version:"+eventID, input)
}

type PostgresVersionHistoryStore struct{ pool *pgxpool.Pool }

func NewPostgresVersionHistoryStore(pool *pgxpool.Pool) (*PostgresVersionHistoryStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresVersionHistoryStore{pool: pool}, nil
}

func (store *PostgresVersionHistoryStore) TrackPageVersion(ctx context.Context, input PageVersionInput) error {
	pageID, userID, err := parseVersionIdentity(input.PageID, input.UserID)
	if err != nil {
		return err
	}
	var original map[string]json.RawMessage
	if input.ExistingInstance != "" {
		if err := json.Unmarshal([]byte(input.ExistingInstance), &original); err != nil {
			return fmt.Errorf("%w: invalid existing page snapshot JSON", ErrPermanentVersionHistory)
		}
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin page version transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var workspaceID uuid.UUID
	var html string
	var binary []byte
	var descriptionJSON []byte
	err = tx.QueryRow(ctx, `SELECT workspace_id,description_html,description_binary,description_json FROM pages WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, pageID).Scan(&workspaceID, &html, &binary, &descriptionJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("load page for version history: %w", err)
	}
	var priorHTML string
	priorRaw, exists := original["description_html"]
	if exists {
		_ = json.Unmarshal(priorRaw, &priorHTML)
	}
	if exists && priorHTML == html {
		return tx.Commit(ctx)
	}
	now := time.Now().UTC()
	var latestID, latestOwner uuid.UUID
	var latestSaved time.Time
	err = tx.QueryRow(ctx, `SELECT id,owned_by_id,last_saved_at FROM page_versions WHERE page_id=$1 AND deleted_at IS NULL ORDER BY last_saved_at DESC LIMIT 1 FOR UPDATE`, pageID).Scan(&latestID, &latestOwner, &latestSaved)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load latest page version: %w", err)
	}
	if err == nil && latestOwner == userID && now.Sub(latestSaved) <= versionCoalesceWindow {
		_, err = tx.Exec(ctx, `UPDATE page_versions SET description_html=$2,description_binary=$3,description_json=$4,description_stripped=$5,sub_pages_data='{}'::jsonb,last_saved_at=$6,updated_at=$6 WHERE id=$1`, latestID, html, binary, descriptionJSON, content.StripTags(html), now)
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO page_versions (id,workspace_id,page_id,owned_by_id,last_saved_at,description_binary,description_html,description_stripped,description_json,sub_pages_data,created_at,updated_at,created_by_id,updated_by_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'{}'::jsonb,$5,$5,NULL,NULL)`, uuid.New(), workspaceID, pageID, userID, now, binary, html, content.StripTags(html), descriptionJSON)
	}
	if err != nil {
		return fmt.Errorf("write page version: %w", err)
	}
	if _, err := tx.Exec(ctx, `WITH oldest AS (SELECT id FROM page_versions WHERE page_id=$1 AND deleted_at IS NULL ORDER BY last_saved_at ASC LIMIT 1) DELETE FROM page_versions WHERE id IN (SELECT id FROM oldest) AND (SELECT COUNT(*) FROM page_versions WHERE page_id=$1 AND deleted_at IS NULL)>20`, pageID); err != nil {
		return fmt.Errorf("retain page versions: %w", err)
	}
	return tx.Commit(ctx)
}

func (store *PostgresVersionHistoryStore) TrackIssueDescriptionVersion(ctx context.Context, input IssueDescriptionVersionInput) error {
	issueID, userID, err := parseVersionIdentity(input.IssueID, input.UserID)
	if err != nil {
		return err
	}
	var original map[string]json.RawMessage
	if input.UpdatedIssue != "" {
		if err := json.Unmarshal([]byte(input.UpdatedIssue), &original); err != nil {
			return fmt.Errorf("%w: invalid updated issue JSON", ErrPermanentVersionHistory)
		}
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin issue description version transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var workspaceID, projectID uuid.UUID
	var createdBy, updatedBy *uuid.UUID
	var html string
	var binary []byte
	var descriptionJSON []byte
	err = tx.QueryRow(ctx, `SELECT workspace_id,project_id,created_by_id,updated_by_id,description_html,description_binary,description_json FROM issues WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, issueID).Scan(&workspaceID, &projectID, &createdBy, &updatedBy, &html, &binary, &descriptionJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("load issue for description version: %w", err)
	}
	if !input.IsCreating {
		var priorHTML string
		priorRaw, exists := original["description_html"]
		if exists && json.Unmarshal(priorRaw, &priorHTML) == nil && priorHTML == html {
			return tx.Commit(ctx)
		}
	}
	now := time.Now().UTC()
	var latestID, latestOwner uuid.UUID
	var latestSaved time.Time
	err = tx.QueryRow(ctx, `SELECT id,owned_by_id,last_saved_at FROM issue_description_versions WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY last_saved_at DESC LIMIT 1 FOR UPDATE`, issueID).Scan(&latestID, &latestOwner, &latestSaved)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load latest issue description version: %w", err)
	}
	if err == nil && latestOwner == userID && now.Sub(latestSaved) <= versionCoalesceWindow {
		_, err = tx.Exec(ctx, `UPDATE issue_description_versions SET description_json=$2,description_html=$3,description_binary=$4,description_stripped=$5,last_saved_at=$6,updated_at=$6 WHERE id=$1`, latestID, descriptionJSON, html, binary, content.StripTags(html), now)
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO issue_description_versions (id,workspace_id,project_id,issue_id,owned_by_id,created_by_id,updated_by_id,last_saved_at,description_json,description_html,description_binary,description_stripped,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$8,$8,NULL)`, uuid.New(), workspaceID, projectID, issueID, userID, createdBy, updatedBy, now, descriptionJSON, html, binary, content.StripTags(html))
	}
	if err != nil {
		return fmt.Errorf("write issue description version: %w", err)
	}
	return tx.Commit(ctx)
}

func parseVersionIdentity(objectID, userID string) (uuid.UUID, uuid.UUID, error) {
	object, err := uuid.Parse(objectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("%w: invalid object ID", ErrPermanentVersionHistory)
	}
	user, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("%w: invalid owner ID", ErrPermanentVersionHistory)
	}
	return object, user, nil
}

func classifyVersionHistoryError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return fmt.Errorf("%w: invalid version history reference", ErrPermanentVersionHistory)
	}
	return err
}
