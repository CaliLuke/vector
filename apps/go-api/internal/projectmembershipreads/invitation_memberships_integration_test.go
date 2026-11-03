//go:build integration

package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func (f memberFixture) invite(t *testing.T, project, accepted bool, role int) string {
	t.Helper()
	table, columns, values := "workspace_member_invites", "", ""
	if project {
		table, columns, values = "project_member_invites", ",project_id", ",'"+mutationProject+"'"
	}
	var id string
	require.NoError(t, f.db.QueryRow(f.ctx, `INSERT INTO `+table+`(id,workspace_id,email,role,token,accepted,created_at,updated_at,created_by_id`+columns+`) VALUES(gen_random_uuid(),$1,$2,$3,'join-token',$4,clock_timestamp(),clock_timestamp(),$5`+values+`) RETURNING id::text`, mutationWorkspace, mutationNew+"@example.test", role, accepted, mutationActor).Scan(&id))
	return id
}
func (f memberFixture) joinInvite(id string, project bool, body string) (json.RawMessage, error) {
	operation := "workspace-join"
	if project {
		operation = "project-join"
	}
	return f.workspaces.Invitation(f.ctx, InvitationCommand{Operation: operation, Slug: "milestones", ProjectID: mutationProject, ID: id, UserID: mutationNew, ContentType: "application/json", Body: strings.NewReader(body)})
}
func (f memberFixture) pendingInvitation(t *testing.T, id string) bool {
	t.Helper()
	var pending bool
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT NOT accepted AND responded_at IS NULL AND deleted_at IS NULL FROM workspace_member_invites WHERE id=$1`, id).Scan(&pending))
	return pending
}

func TestInvitationAcceptanceRollsBackWhenTrackingAdmissionFails(t *testing.T) {
	f := newMemberFixture(t)
	f.exec(t, `DELETE FROM workspace_members WHERE member_id=$1`, mutationNew)
	id := f.invite(t, false, false, 15)
	f.exec(t, `ALTER TABLE work_item_effects ADD CONSTRAINT reject_join_event CHECK(kind<>'event_tracking')`)
	before := f.snapshot(t)
	_, err := f.joinInvite(id, false, `{"token":"join-token","accepted":true}`)
	require.Error(t, err)
	require.JSONEq(t, before, f.snapshot(t))
	require.True(t, f.pendingInvitation(t, id))
	f.exec(t, `ALTER TABLE work_item_effects DROP CONSTRAINT reject_join_event`)
	_, err = f.joinInvite(id, false, `{"token":"join-token","accepted":true}`)
	require.NoError(t, err)
	var members, events int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM workspace_members WHERE member_id=$1 AND is_active),(SELECT count(*) FROM work_item_effects WHERE kind='event_tracking')`, mutationNew).Scan(&members, &events))
	require.Equal(t, 1, members)
	require.Equal(t, 1, events)
	_, err = f.joinInvite(id, false, `{"token":"join-token","accepted":true}`)
	memberFailure(t, err, 404)
}

func TestMemberRemovalRollsBackWhenInvitationRevocationFails(t *testing.T) {
	for _, test := range []struct {
		name, table string
		project     bool
	}{
		{"project invitation", "project_member_invites", true},
		{"workspace invitation", "workspace_member_invites", false},
		{"workspace cascade", "project_member_invites", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMemberFixture(t)
			_, err := f.project(mutationActor, "legacy-create-members", "POST", "", `{"members":[{"member_id":"`+mutationNew+`","role":15}]}`)
			require.NoError(t, err)
			f.invite(t, false, false, 15)
			f.invite(t, true, false, 15)
			f.exec(t, `ALTER TABLE `+test.table+` ADD CONSTRAINT keep_pending_invitation CHECK(deleted_at IS NULL)`)
			remove := func() error {
				if test.project {
					_, err := f.project(mutationActor, "legacy-delete-members", "DELETE", f.id(t, "project_members", mutationNew), "")
					return err
				}
				_, err := f.workspace(mutationActor, "member-delete", f.id(t, "workspace_members", mutationNew), "")
				return err
			}
			before := f.snapshot(t)
			require.Error(t, remove())
			require.JSONEq(t, before, f.snapshot(t), "failed revocation must leave membership and effects unchanged")
			var pending int
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM workspace_member_invites WHERE deleted_at IS NULL)+(SELECT count(*) FROM project_member_invites WHERE deleted_at IS NULL)`).Scan(&pending))
			require.Equal(t, 2, pending, "a failed cascade must also roll back earlier invitation revocations")
			f.exec(t, `ALTER TABLE `+test.table+` DROP CONSTRAINT keep_pending_invitation`)
			require.NoError(t, remove())
			var active int
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM project_members WHERE member_id=$1 AND is_active`, mutationNew).Scan(&active))
			require.Zero(t, active)
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM project_member_invites WHERE deleted_at IS NULL`).Scan(&pending))
			require.Zero(t, pending)
		})
	}
}

func TestProjectInvitationUsesTargetTenantAndCapsGuestRole(t *testing.T) {
	f := newMemberFixture(t)
	f.exec(t, `UPDATE workspace_members SET role=5 WHERE member_id=$1`, mutationNew)
	id := f.invite(t, true, false, 20)
	_, err := f.joinInvite(id, true, `{"token":"join-token","accepted":true}`)
	require.NoError(t, err)
	var role int
	var workspace string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT role,workspace_id::text FROM project_members WHERE project_id=$1 AND member_id=$2`, mutationProject, mutationNew).Scan(&role, &workspace))
	require.Equal(t, 5, role)
	require.Equal(t, mutationWorkspace, workspace)
	var properties int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM project_user_properties WHERE project_id=$1 AND user_id=$2`, mutationProject, mutationNew).Scan(&properties))
	require.Equal(t, 1, properties)
}

func TestInvitationJoinRejectsTokenRecipientAndNonBooleanAcceptance(t *testing.T) {
	f := newMemberFixture(t)
	id := f.invite(t, false, false, 15)
	before := f.snapshot(t)
	_, err := f.joinInvite(id, false, `{"token":"wrong","accepted":true}`)
	memberFailure(t, err, 403)
	_, err = f.joinInvite(id, false, `{"token":"join-token","accepted":"false"}`)
	memberFailure(t, err, 400)
	f.exec(t, `UPDATE workspace_member_invites SET email='someone-else@example.test' WHERE id=$1`, id)
	_, err = f.joinInvite(id, false, `{"token":"join-token","accepted":true}`)
	memberFailure(t, err, 403)
	require.True(t, f.pendingInvitation(t, id))
	require.JSONEq(t, before, f.snapshot(t))
}

func TestAcceptedInvitationsCommitAsOneBatch(t *testing.T) {
	f := newMemberFixture(t)
	f.exec(t, `DELETE FROM workspace_members WHERE member_id=$1`, mutationNew)
	workspace := f.invite(t, false, true, 15)
	project := f.invite(t, true, true, 20)
	f.exec(t, `ALTER TABLE project_user_properties ADD CONSTRAINT reject_join_property CHECK(user_id<>'00000000-0000-4000-8000-000000000005')`)
	before := f.snapshot(t)
	require.Error(t, f.workspaces.JoinAcceptedInvitations(f.ctx, mutationNew))
	require.JSONEq(t, before, f.snapshot(t))
	var pending int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM workspace_member_invites WHERE id=$1 AND deleted_at IS NULL)+(SELECT count(*) FROM project_member_invites WHERE id=$2 AND deleted_at IS NULL)`, workspace, project).Scan(&pending))
	require.Equal(t, 2, pending)
	f.exec(t, `ALTER TABLE project_user_properties DROP CONSTRAINT reject_join_property`)
	require.NoError(t, f.workspaces.JoinAcceptedInvitations(f.ctx, mutationNew))
	after := f.snapshot(t)
	require.NoError(t, f.workspaces.JoinAcceptedInvitations(f.ctx, mutationNew))
	require.JSONEq(t, after, f.snapshot(t))
}

func TestInvitationJoinRechecksAccountAndMembershipAfterWaiting(t *testing.T) {
	for _, change := range []string{"deactivate", "remove", "role"} {
		t.Run(change, func(t *testing.T) {
			f := newMemberFixture(t)
			id := f.invite(t, false, false, 20)
			tx, err := f.db.Begin(f.ctx)
			require.NoError(t, err)
			t.Cleanup(func() {
				rollbackErr := tx.Rollback(context.Background())
				if !errors.Is(rollbackErr, pgx.ErrTxClosed) {
					require.NoError(t, rollbackErr)
				}
			})
			_, err = tx.Exec(f.ctx, `LOCK TABLE workspace_members,project_members IN SHARE ROW EXCLUSIVE MODE`)
			require.NoError(t, err)
			result := make(chan error, 1)
			go func() {
				_, joinErr := f.joinInvite(id, false, `{"token":"join-token","accepted":true}`)
				result <- joinErr
			}()
			require.Eventually(t, func() bool {
				var waiting bool
				queryErr := f.db.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'LOCK TABLE workspace_members,%')`).Scan(&waiting)
				return queryErr == nil && waiting
			}, 5*time.Second, 10*time.Millisecond)
			switch change {
			case "deactivate":
				_, err = tx.Exec(f.ctx, `UPDATE users SET is_active=false WHERE id=$1`, mutationNew)
			case "remove":
				bound := *f.workspaces
				bound.writer = tx
				_, err = bound.Invitation(f.ctx, InvitationCommand{Operation: "member-delete", Slug: "milestones", UserID: mutationActor, ID: f.id(t, "workspace_members", mutationNew)})
			case "role":
				_, err = tx.Exec(f.ctx, `UPDATE workspace_members SET role=5 WHERE member_id=$1`, mutationNew)
			}
			require.NoError(t, err)
			require.NoError(t, tx.Commit(f.ctx))
			err = <-result
			switch change {
			case "deactivate":
				memberFailure(t, err, 403)
			case "remove":
				memberFailure(t, err, 404)
			case "role":
				require.NoError(t, err)
			}
			var role int
			var active bool
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT role,is_active FROM workspace_members WHERE member_id=$1`, mutationNew).Scan(&role, &active))
			if change == "role" {
				require.Equal(t, 5, role)
			}
			if change == "remove" {
				require.False(t, active)
			}
		})
	}
}
