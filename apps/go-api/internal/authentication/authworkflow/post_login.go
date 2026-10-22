package authworkflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const joinedWorkspaceTask = "plane.bgtasks.event_tracking_task.track_event"

type Database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type InvitationCache interface {
	Invalidate(context.Context, string, string, bool) error
}

type TaskPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}

type workspaceInvitation struct {
	workspaceID string
	slug        string
	role        int
}

// PostUserAuthWorkflow implements plane.authentication.utils.user_auth_workflow
// for successful email, magic-code, and OAuth authentication.
type PostUserAuthWorkflow struct {
	database  Database
	cache     InvitationCache
	publisher TaskPublisher
	now       func() time.Time
}

func NewPostUserAuthWorkflow(database Database, cache InvitationCache, publisher TaskPublisher) (*PostUserAuthWorkflow, error) {
	if database == nil {
		return nil, errors.New("post-user-auth workflow database is required")
	}
	if cache == nil {
		return nil, errors.New("post-user-auth workflow invitation cache is required")
	}
	if publisher == nil {
		return nil, errors.New("post-user-auth workflow task publisher is required")
	}
	return &PostUserAuthWorkflow{database: database, cache: cache, publisher: publisher, now: time.Now}, nil
}

// Run joins accepted workspace and project invitations for the authenticated
// user's email. It preserves the source order: workspace membership creation,
// per-workspace cache invalidation and tracking task, project workspace and
// project memberships, then soft deletion of both invitation sets.
func (workflow *PostUserAuthWorkflow) Run(ctx context.Context, userID, email string) error {
	workspaceInvites, err := workflow.workspaceInvitations(ctx, email)
	if err != nil {
		return err
	}
	if _, err := workflow.database.Exec(ctx, `
INSERT INTO workspace_members (id, workspace_id, member_id, role, created_at, updated_at)
SELECT gen_random_uuid(), invite.workspace_id, @user_id, invite.role, clock_timestamp(), clock_timestamp()
FROM workspace_member_invites invite
WHERE invite.email = @email AND invite.accepted = true AND invite.deleted_at IS NULL
ORDER BY invite.created_at DESC
ON CONFLICT DO NOTHING
`, pgx.StrictNamedArgs{"user_id": userID, "email": email}); err != nil {
		return fmt.Errorf("create workspace memberships from accepted invitations: %w", err)
	}
	for _, invite := range workspaceInvites {
		if err := workflow.cache.Invalidate(ctx, "/api/workspaces/"+invite.slug+"/members/", "", true); err != nil {
			return fmt.Errorf("invalidate workspace member cache for %s: %w", invite.slug, err)
		}
		joinedAt := workflow.now().UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.000000Z07:00")
		properties := map[string]any{
			"user_id":        userID,
			"workspace_id":   invite.workspaceID,
			"workspace_slug": invite.slug,
			"role":           invite.role,
			"joined_at":      joinedAt,
		}
		if err := workflow.publisher.Task(ctx, joinedWorkspaceTask, nil, map[string]any{
			"user_id": userID, "event_name": "user_joined_workspace", "slug": invite.slug,
			"event_properties": properties,
		}); err != nil {
			return fmt.Errorf("publish user-joined-workspace event for %s: %w", invite.slug, err)
		}
	}
	if _, err := workflow.database.Exec(ctx, `
INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, created_at, updated_at)
SELECT gen_random_uuid(), invite.workspace_id, @user_id,
       CASE WHEN invite.role IN (5, 15) THEN invite.role ELSE 15 END,
       invite.created_by_id, clock_timestamp(), clock_timestamp()
FROM project_member_invites invite
WHERE invite.email = @email AND invite.accepted = true AND invite.deleted_at IS NULL
ORDER BY invite.created_at DESC
ON CONFLICT DO NOTHING
`, pgx.StrictNamedArgs{"user_id": userID, "email": email}); err != nil {
		return fmt.Errorf("create workspace memberships from accepted project invitations: %w", err)
	}
	if _, err := workflow.database.Exec(ctx, `
INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, created_at, updated_at)
SELECT gen_random_uuid(), invite.workspace_id, invite.project_id, @user_id,
       CASE WHEN invite.role IN (5, 15) THEN invite.role ELSE 15 END,
       invite.created_by_id, clock_timestamp(), clock_timestamp()
FROM project_member_invites invite
WHERE invite.email = @email AND invite.accepted = true AND invite.deleted_at IS NULL
ORDER BY invite.created_at DESC
ON CONFLICT DO NOTHING
`, pgx.StrictNamedArgs{"user_id": userID, "email": email}); err != nil {
		return fmt.Errorf("create project memberships from accepted invitations: %w", err)
	}
	if _, err := workflow.database.Exec(ctx, `
UPDATE workspace_member_invites
SET deleted_at = clock_timestamp()
WHERE email = @email AND accepted = true AND deleted_at IS NULL
`, pgx.StrictNamedArgs{"email": email}); err != nil {
		return fmt.Errorf("soft-delete accepted workspace invitations: %w", err)
	}
	if _, err := workflow.database.Exec(ctx, `
UPDATE project_member_invites
SET deleted_at = clock_timestamp()
WHERE email = @email AND accepted = true AND deleted_at IS NULL
`, pgx.StrictNamedArgs{"email": email}); err != nil {
		return fmt.Errorf("soft-delete accepted project invitations: %w", err)
	}
	return nil
}

func (workflow *PostUserAuthWorkflow) workspaceInvitations(ctx context.Context, email string) ([]workspaceInvitation, error) {
	rows, err := workflow.database.Query(ctx, `
SELECT invite.workspace_id::text, workspace.slug, invite.role
FROM workspace_member_invites invite
JOIN workspaces workspace ON workspace.id = invite.workspace_id
WHERE invite.email = @email AND invite.accepted = true AND invite.deleted_at IS NULL
ORDER BY invite.created_at DESC
`, pgx.StrictNamedArgs{"email": email})
	if err != nil {
		return nil, fmt.Errorf("read accepted workspace invitations: %w", err)
	}
	defer rows.Close()
	invites := make([]workspaceInvitation, 0)
	for rows.Next() {
		var invite workspaceInvitation
		if err := rows.Scan(&invite.workspaceID, &invite.slug, &invite.role); err != nil {
			return nil, fmt.Errorf("scan accepted workspace invitation: %w", err)
		}
		invites = append(invites, invite)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accepted workspace invitations: %w", err)
	}
	return invites, nil
}
