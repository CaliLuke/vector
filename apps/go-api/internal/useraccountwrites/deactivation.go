package useraccountwrites

import (
	"context"
	"uuid"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5"
)

// DeactivationBlocked explains which administrator responsibility must be reassigned.
type DeactivationBlocked string

func (reason DeactivationBlocked) Error() string { return string(reason) }

// Deactivate commits account disablement and its notification together.
func (repository *Repository) Deactivate(ctx context.Context, userID uuid.UUID, sessionID, baseURL, lastLogoutIP string) (resultErr error) {
	tx, err := repository.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	// Deactivation is infrequent and spans every membership of an account.
	// Serialize writes to these tables so the administrator check sees preceding
	// changes and remains valid until this transaction commits, including inserts.
	if _, err = tx.Exec(ctx, `LOCK TABLE instance_admins,workspace_members,project_members IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	if _, err = repository.lockAccountSession(ctx, tx, userID, sessionID); err != nil {
		return err
	}
	blocker, err := deactivationBlocker(ctx, tx, userID)
	if err != nil {
		return err
	}
	if blocker != "" {
		return DeactivationBlocked(blocker)
	}
	for _, query := range []string{
		`UPDATE project_members SET is_active=false,updated_at=clock_timestamp() WHERE member_id=$1 AND is_active AND deleted_at IS NULL`,
		`UPDATE workspace_members SET is_active=false,updated_at=clock_timestamp() WHERE member_id=$1 AND is_active AND deleted_at IS NULL`,
		`UPDATE workspace_member_invites SET deleted_at=clock_timestamp(),updated_at=clock_timestamp() WHERE email=(SELECT email FROM users WHERE id=$1) AND deleted_at IS NULL`,
		`DELETE FROM account_email_challenges WHERE user_id=$1`,
	} {
		if _, err = tx.Exec(ctx, query, userID); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, userID.String()); err != nil {
		return err
	}
	profile, err := tx.Exec(ctx, `UPDATE profiles SET last_workspace_id=NULL,is_tour_completed=false,is_onboarded=false,onboarding_step='{"workspace_join":false,"profile_complete":false,"workspace_create":false,"workspace_invite":false}'::jsonb,updated_at=clock_timestamp() WHERE user_id=$1`, userID)
	if err != nil {
		return err
	}
	if profile.RowsAffected() != 1 {
		return ErrNotFound
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET password='!',is_password_autoset=true,is_active=false,last_logout_ip=$2,last_logout_time=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1`, userID, lastLogoutIP); err != nil {
		return err
	}
	if err = queueAccountEmail(ctx, tx, "plane.bgtasks.user_deactivation_email_task.user_deactivation_email", baseURL, userID.String()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func deactivationBlocker(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (string, error) {
	var blocker string
	err := tx.QueryRow(ctx, `SELECT CASE
 WHEN EXISTS(SELECT 1 FROM instance_admins a JOIN instances i ON i.id=a.instance_id
  WHERE a.user_id=$1 AND a.deleted_at IS NULL AND i.deleted_at IS NULL)
 THEN 'You cannot deactivate your account since you are an instance admin'
 WHEN EXISTS(SELECT 1 FROM project_members mine
  JOIN project_members members ON members.project_id=mine.project_id AND members.is_active AND members.deleted_at IS NULL
  WHERE mine.member_id=$1 AND mine.role=20 AND mine.is_active AND mine.deleted_at IS NULL
  GROUP BY mine.id HAVING count(*)>1 AND count(*) FILTER(WHERE members.role=20 AND members.member_id<>mine.member_id)=0)
 THEN 'You cannot deactivate account as you are the only admin in some projects.'
 WHEN EXISTS(SELECT 1 FROM workspace_members mine
  JOIN workspace_members members ON members.workspace_id=mine.workspace_id AND members.is_active AND members.deleted_at IS NULL
  WHERE mine.member_id=$1 AND mine.role=20 AND mine.is_active AND mine.deleted_at IS NULL
  GROUP BY mine.id HAVING count(*)>1 AND count(*) FILTER(WHERE members.role=20 AND members.member_id<>mine.member_id)=0)
 THEN 'You cannot deactivate account as you are the only admin in some workspaces.'
 ELSE '' END`, userID).Scan(&blocker)
	return blocker, err
}
