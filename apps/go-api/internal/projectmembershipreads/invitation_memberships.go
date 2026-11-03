package projectmembershipreads

import (
	"context"
	"errors"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/datetime"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/jackc/pgx/v5"
)

type invitationMemberships struct {
	tx           pgx.Tx
	actor, email string
}

func beginInvitationMemberships(ctx context.Context, db membershipDatabase, actor string) (*invitationMemberships, error) {
	if actor == "" {
		return nil, inviteFailure(401, "Authentication required to accept invitation")
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	m := &invitationMemberships{tx: tx, actor: actor}
	// Use the same order as member changes and account deactivation, including
	// inserts. Read the account only after preceding membership writes commit.
	if _, err = tx.Exec(ctx, `LOCK TABLE workspace_members,project_members IN SHARE ROW EXCLUSIVE MODE`); err == nil {
		err = tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1 AND is_active FOR SHARE`, actor).Scan(&m.email)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = inviteFailure(403, "Active account is required")
		}
		return nil, dbutil.RollbackTransaction(ctx, tx.Rollback, err)
	}
	return m, nil
}

type membershipInvitation struct {
	id, workspace, slug, project, email, token string
	role                                       int
	responded                                  *time.Time
}

func (m *invitationMemberships) join(ctx context.Context, i membershipInvitation) error {
	if i.role != 5 && i.role != 15 && i.role != 20 {
		return inviteFailure(400, "Invalid invitation role")
	}
	var workspace string
	if err := m.tx.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE id=$1 AND deleted_at IS NULL FOR SHARE`, i.workspace).Scan(&workspace); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inviteMissing()
		}
		return err
	}
	if i.project != "" {
		var project string
		if err := m.tx.QueryRow(ctx, `SELECT id::text FROM projects WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR SHARE`, i.project, i.workspace).Scan(&project); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return inviteMissing()
			}
			return err
		}
	}
	workspaceRole := i.role
	if i.project != "" && workspaceRole == 20 {
		workspaceRole = 15
	}
	var existingRole int
	var active bool
	err := m.tx.QueryRow(ctx, `SELECT role,is_active FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL`, i.workspace, m.actor).Scan(&existingRole, &active)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = insertWorkspaceMembership(ctx, m.tx, i.workspace, m.actor, workspaceRole, m.actor)
	case err == nil:
		if existingRole != 5 && existingRole != 15 && existingRole != 20 {
			return inviteFailure(403, "Invalid workspace role")
		}
		workspaceRole = existingRole
		if !active {
			_, err = m.tx.Exec(ctx, `UPDATE workspace_members SET is_active=true,updated_at=clock_timestamp(),updated_by_id=$3 WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL`, i.workspace, m.actor, m.actor)
		}
	}
	if err != nil {
		return err
	}
	if i.project != "" {
		role := i.role
		if workspaceRole == 5 {
			role = 5
		}
		if err = insertProjectMembership(ctx, m.tx, i.project, i.workspace, m.actor, role, m.actor, true); err != nil {
			return err
		}
		// Preserve current roles; reactivation cannot promote a workspace guest.
		_, err = m.tx.Exec(ctx, `UPDATE project_members SET is_active=true,role=CASE WHEN $4=5 THEN 5 ELSE role END,updated_at=clock_timestamp(),updated_by_id=$3 WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND deleted_at IS NULL AND (NOT is_active OR ($4=5 AND role<>5))`, i.project, i.workspace, m.actor, workspaceRole)
		return err
	}
	if active {
		return nil
	}
	return (projectmutations.Effects{Tx: m.tx}).Task(ctx, "plane.bgtasks.event_tracking_task.track_event", nil, map[string]any{
		"user_id": m.actor, "event_name": "user_joined_workspace", "slug": i.slug,
		"event_properties": map[string]any{"user_id": m.actor, "workspace_id": i.workspace, "workspace_slug": i.slug, "role": workspaceRole, "joined_at": datetime.Format(time.Now(), time.UTC)},
	})
}

func (m *invitationMemberships) consume(ctx context.Context, i membershipInvitation) error {
	table := "workspace_member_invites"
	if i.project != "" {
		table = "project_member_invites"
	}
	_, err := m.tx.Exec(ctx, `UPDATE `+table+` SET accepted=true,responded_at=COALESCE(responded_at,clock_timestamp()),deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, i.id, m.actor)
	return err
}

func (m *invitationMemberships) invitations(ctx context.Context, project, acceptedOnly bool, ids []string) ([]membershipInvitation, error) {
	table, projectColumn := "workspace_member_invites", "''"
	if project {
		table, projectColumn = "project_member_invites", "i.project_id::text"
	}
	rows, err := m.tx.Query(ctx, `SELECT i.id::text,i.workspace_id::text,w.slug,`+projectColumn+`,i.email,i.token,i.role,i.responded_at FROM `+table+` i JOIN workspaces w ON w.id=i.workspace_id WHERE lower(i.email)=lower($1) AND i.deleted_at IS NULL AND w.deleted_at IS NULL AND (($2 AND i.accepted) OR (NOT $2 AND i.responded_at IS NULL AND i.id::text=ANY($3))) ORDER BY i.id FOR UPDATE OF i`, m.email, acceptedOnly, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []membershipInvitation{}
	for rows.Next() {
		var i membershipInvitation
		if err = rows.Scan(&i.id, &i.workspace, &i.slug, &i.project, &i.email, &i.token, &i.role, &i.responded); err != nil {
			return nil, err
		}
		result = append(result, i)
	}
	return result, rows.Err()
}

// JoinAcceptedInvitations consumes invitations previously accepted by this
// account. The email argument at authentication is never an authority source.
func (r *InvitationsRepository) JoinAcceptedInvitations(ctx context.Context, actor string) (resultErr error) {
	m, err := beginInvitationMemberships(ctx, r.writer, actor)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, m.tx.Rollback, resultErr) }()
	for _, project := range []bool{false, true} {
		invitations, err := m.invitations(ctx, project, true, nil)
		if err != nil {
			return err
		}
		for _, i := range invitations {
			if err = m.join(ctx, i); err != nil {
				return err
			}
			if err = m.consume(ctx, i); err != nil {
				return err
			}
		}
	}
	return m.tx.Commit(ctx)
}
