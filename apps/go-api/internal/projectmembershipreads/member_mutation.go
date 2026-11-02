package projectmembershipreads

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"

	dbutil "github.com/CaliLuke/go-plane/internal/database"

	"github.com/CaliLuke/go-plane/internal/requestinput"
	"github.com/jackc/pgx/v5"
)

type membershipDatabase interface {
	database
	Begin(context.Context) (pgx.Tx, error)
}

type memberMutation struct {
	tx             pgx.Tx
	workspace      string
	workspaceAdmin bool
	projects       []string
}

func beginMemberMutation(ctx context.Context, db membershipDatabase, slug string) (*memberMutation, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	m := &memberMutation{tx: tx}
	// Match account deactivation's table lock order. These infrequent writes
	// serialize admission and cascades, including membership inserts.
	if _, err = tx.Exec(ctx, `LOCK TABLE workspace_members,project_members IN SHARE ROW EXCLUSIVE MODE`); err == nil {
		err = tx.QueryRow(ctx, `SELECT w.id::text,
   EXISTS(SELECT 1 FROM workspace_members wm JOIN users u ON u.id=wm.member_id WHERE wm.workspace_id=w.id AND wm.role=20 AND wm.is_active AND wm.deleted_at IS NULL AND u.is_active),
   ARRAY(SELECT DISTINCT pm.project_id::text FROM project_members pm JOIN projects p ON p.id=pm.project_id AND p.workspace_id=pm.workspace_id JOIN users u ON u.id=pm.member_id WHERE pm.workspace_id=w.id AND pm.role=20 AND pm.is_active AND pm.deleted_at IS NULL AND p.deleted_at IS NULL AND u.is_active)
   FROM workspaces w WHERE w.slug=$1 AND w.deleted_at IS NULL`, slug).Scan(&m.workspace, &m.workspaceAdmin, &m.projects)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = inviteMissing()
		}
		return nil, dbutil.RollbackTransaction(ctx, tx.Rollback, err)
	}
	return m, nil
}

func (m *memberMutation) actorRole(ctx context.Context, actor string) (int, error) {
	var role int
	err := m.tx.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN users u ON u.id=wm.member_id
  WHERE wm.workspace_id=$1 AND wm.member_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND u.is_active AND wm.role IN (5,15,20)`, m.workspace, actor).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, inviteFailure(http.StatusForbidden, "Active workspace membership is required")
	}
	return role, err
}

func (m *memberMutation) commit(ctx context.Context) error {
	var workspaceLost, projectLost bool
	err := m.tx.QueryRow(ctx, `SELECT
  $2 AND NOT EXISTS(SELECT 1 FROM workspace_members wm JOIN users u ON u.id=wm.member_id WHERE wm.workspace_id=$1 AND wm.role=20 AND wm.is_active AND wm.deleted_at IS NULL AND u.is_active),
  EXISTS(SELECT 1 FROM unnest($3::uuid[]) AS required(id) WHERE NOT EXISTS(
   SELECT 1 FROM project_members pm JOIN users u ON u.id=pm.member_id WHERE pm.project_id=required.id AND pm.workspace_id=$1 AND pm.role=20 AND pm.is_active AND pm.deleted_at IS NULL AND u.is_active))`, m.workspace, m.workspaceAdmin, m.projects).Scan(&workspaceLost, &projectLost)
	if err != nil {
		return err
	}
	if workspaceLost {
		return inviteFailure(http.StatusBadRequest, "Promote another workspace administrator before removing the last administrator")
	}
	if projectLost {
		return inviteFailure(http.StatusBadRequest, "Promote another project administrator before removing the last administrator")
	}
	return m.tx.Commit(ctx)
}

func memberFields(body io.Reader, contentType string, allowed ...string) (map[string]any, error) {
	input, err := requestinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if input.Null || input.Kind != "" {
		return nil, inviteFailure(http.StatusBadRequest, "Membership data must be an object")
	}
	for field := range input.Fields {
		if !slices.Contains(allowed, field) {
			return nil, inviteFailure(http.StatusBadRequest, "Unsupported membership field: "+field)
		}
	}
	return input.Fields, nil
}

func memberRole(value any) (int, error) {
	number, err := requestinput.IntegerField(value)
	if err != nil || !number.IsInt64() {
		return 0, inviteFailure(http.StatusBadRequest, "Role must be 5, 15, or 20")
	}
	role := int(number.Int64())
	if role != 5 && role != 15 && role != 20 {
		return 0, inviteFailure(http.StatusBadRequest, "Role must be 5, 15, or 20")
	}
	return role, nil
}
