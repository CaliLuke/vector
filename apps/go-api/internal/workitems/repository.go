// Package workitems owns work-item persistence, including the model hooks that
// ordinary edits, intake and draft publication share.
package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type Publisher interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}

// IssueDescriptionVersionAdmitter is the native durable execution path for
// issue description history. It is optional during the gradual producer
// cutover; without it the repository retains the legacy Celery publisher.
type IssueDescriptionVersionAdmitter interface {
	AdmitIssueDescriptionVersion(context.Context, string, string, string, string, bool) error
}

// RecentVisitAdmitter is the native durable path for the activity emitted by
// app-side issue reads. The operation ID is stable only across reconciliation
// of that one uncertain admission attempt.
type RecentVisitAdmitter interface {
	AdmitRecentVisit(context.Context, string, string, string, string, string, string) error
}

type IssueActivityAdmitter interface {
	AdmitIssueActivityTask(context.Context, string, map[string]any) error
}

type Repository struct {
	pql                      *issuequery.PreparedPQL
	propertyFilter           *issuequery.PreparedPropertyFilter
	propertyPredicates       map[int]*workitemtypes.ScopedPropertyPredicate
	propertyDiscovery        *workitemtypes.PropertyDiscoverySnapshot
	propertyGroups           map[string]*workitemtypes.DiscoveryProperty
	parentGrouping           *parentGrouping
	typeDiscovery            *workitemtypes.TypeIdentityCatalog
	db                       Database
	publisher                Publisher
	issueDescriptionVersions IssueDescriptionVersionAdmitter
	recentVisits             RecentVisitAdmitter
	issueActivities          IssueActivityAdmitter
	transactionalEffects     bool
	itemEdit                 *itemEditState
	origin                   string
	now                      func() time.Time
}

func NewRepository(db Database, publisher Publisher, origin string, issueDescriptionVersions ...IssueDescriptionVersionAdmitter) (*Repository, error) {
	if db == nil || publisher == nil {
		return nil, errors.New("work-item database and durable publisher are required")
	}
	if len(issueDescriptionVersions) > 1 {
		return nil, errors.New("at most one native issue description version admitter may be configured")
	}
	var versions IssueDescriptionVersionAdmitter
	if len(issueDescriptionVersions) == 1 {
		versions = issueDescriptionVersions[0]
	}
	return NewRepositoryWithNativeTasks(db, publisher, origin, versions, nil)
}

func NewRepositoryWithNativeTasks(db Database, publisher Publisher, origin string, issueDescriptionVersions IssueDescriptionVersionAdmitter, recentVisits RecentVisitAdmitter, issueActivityAdmitters ...IssueActivityAdmitter) (*Repository, error) {
	if db == nil || publisher == nil {
		return nil, errors.New("work-item database and durable publisher are required")
	}
	if len(issueActivityAdmitters) > 1 {
		return nil, errors.New("at most one native issue activity admitter may be configured")
	}
	var issueActivities IssueActivityAdmitter
	if len(issueActivityAdmitters) == 1 {
		issueActivities = issueActivityAdmitters[0]
	}
	return &Repository{db: db, publisher: publisher, origin: origin, now: time.Now, issueDescriptionVersions: issueDescriptionVersions, recentVisits: recentVisits, issueActivities: issueActivities}, nil
}

func (r *Repository) publishTask(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if r.transactionalEffects && name == "plane.bgtasks.webhook_task.model_activity" {
		return r.enqueueEffect(ctx, "webhook", map[string]any{"name": name, "args": args, "kwargs": kwargs})
	}
	if r.transactionalEffects && name == "plane.bgtasks.issue_activities_task.issue_activity" {
		return r.enqueueEffect(ctx, "activity", kwargs)
	}
	if name == "plane.bgtasks.issue_activities_task.issue_activity" && r.issueActivities != nil {
		return r.issueActivities.AdmitIssueActivityTask(ctx, uuid.NewString(), kwargs)
	}
	return r.publisher.Task(ctx, name, args, kwargs)
}

func (r *Repository) admitIssueDescriptionVersion(ctx context.Context, updatedIssue any, issueID, userID string, isCreating bool) error {
	if r.transactionalEffects {
		snapshot, err := snapshotJSON(updatedIssue)
		if err != nil {
			return err
		}
		return r.enqueueEffect(ctx, "description_version", map[string]any{"snapshot": snapshot, "issue_id": issueID, "user_id": userID, "is_creating": isCreating})
	}
	if r.issueDescriptionVersions != nil {
		snapshot, err := snapshotJSON(updatedIssue)
		if err != nil {
			return err
		}
		// Allocate one identity for this effect before admission. The dispatcher
		// reuses it when reconciling an uncertain StartWorkflow result; distinct
		// identical operations deliberately receive different identities.
		return r.issueDescriptionVersions.AdmitIssueDescriptionVersion(ctx, uuid.NewString(), snapshot, issueID, userID, isCreating)
	}
	kwargs := map[string]any{"updated_issue": updatedIssue, "issue_id": issueID, "user_id": userID}
	if isCreating {
		kwargs["is_creating"] = true
	}
	return r.publisher.Task(ctx, "plane.bgtasks.issue_description_version_task.issue_description_version_task", []any{}, kwargs)
}

func snapshotJSON(value any) (string, error) {
	if text, ok := value.(string); ok {
		return text, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode issue version snapshot: %w", err)
	}
	return string(encoded), nil
}

type Request struct {
	// ExpectedWorkspaceID requires a bound, fresh primary-data read scope.
	ExpectedWorkspaceID                                            *uuid.UUID
	Operation, Method, Slug, ProjectID, IssueID, ActorID, Timezone string
	Params                                                         map[string]string
	Present                                                        map[string]bool
	API                                                            bool
}

type Result struct {
	Status int
	Body   any
}

func failure(status int, field, message string) error {
	return &djangoapi.HTTPError{Status: status, Body: map[string]any{field: message}}
}
func invalid(fields map[string]any) error { return &djangoapi.HTTPError{Status: 400, Body: fields} }
func requiredObject() error               { return failure(404, "error", "The required object does not exist.") }
func forbidden() error                    { return failure(403, "error", "You don't have the required permissions.") }

// allowProject is the app decorator's authorization, including its workspace
// administrator exception. API-v1 uses its distinct ProjectEntityPermission.
func (r *Repository) allowProject(ctx context.Context, q Request, roles []int, creator bool) error {
	var allowed bool
	if creator {
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL)`, q.Slug, q.ActorID).Scan(&allowed); err != nil {
			return err
		}
		if !allowed {
			return forbidden()
		}
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM issues WHERE id=$1 AND created_by_id=$2 AND deleted_at IS NULL)`, q.IssueID, q.ActorID).Scan(&allowed); err != nil {
			return err
		}
		if allowed {
			return nil
		}
	}
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM project_members m JOIN workspaces w ON w.id=m.workspace_id
 WHERE w.slug=$1 AND m.project_id=$2 AND m.member_id=$3 AND m.is_active AND m.deleted_at IS NULL
 AND (m.role=ANY($4::int[]) OR EXISTS (SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id
 AND wm.member_id=$3 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role=20)))`, q.Slug, q.ProjectID, q.ActorID, roles).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return forbidden()
	}
	return nil
}

func (r *Repository) allowWorkspace(ctx context.Context, q Request, roles []int) error {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL AND m.role=ANY($3::int[]))`, q.Slug, q.ActorID, roles).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return forbidden()
	}
	return nil
}

func (r *Repository) row(ctx context.Context, query string, args ...any) (map[string]any, error) {
	v, err := djangoapi.JSON(ctx, r.db, query, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, requiredObject()
	}
	if err != nil {
		return nil, err
	}
	row, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected work-item object, got %T", v)
	}
	return row, nil
}

func (r *Repository) project(ctx context.Context, id string) (map[string]any, error) {
	return r.row(ctx, `SELECT to_jsonb(p) FROM projects p WHERE id=$1 AND deleted_at IS NULL`, id)
}

const liveIssue = `i.deleted_at IS NULL AND i.archived_at IS NULL AND NOT i.is_draft
 AND NOT EXISTS (SELECT 1 FROM states s WHERE s.id=i.state_id AND s."group"='triage')
 AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id=i.project_id AND p.archived_at IS NOT NULL)`

func (r *Repository) issue(ctx context.Context, q Request, live bool) (map[string]any, error) {
	predicate := `i.deleted_at IS NULL`
	if live {
		predicate = liveIssue
	}
	row, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id
 WHERE i.id=$1 AND i.project_id=$2 AND w.slug=$3 AND `+predicate, q.IssueID, q.ProjectID, q.Slug)
	if q.API {
		var missing *djangoapi.HTTPError
		if errors.As(err, &missing) && missing.Status == 404 {
			return nil, failure(404, "error", "The requested resource does not exist.")
		}
	}
	return row, err
}
