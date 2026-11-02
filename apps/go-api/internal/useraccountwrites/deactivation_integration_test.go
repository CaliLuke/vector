//go:build integration

package useraccountwrites

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func newDeactivationFixture(t *testing.T) emailFixture {
	t.Helper()
	f := newEmailFixture(t)
	f.exec(t, testsupport.MilestoneWriteSeed)
	f.exec(t, `INSERT INTO profiles(id,user_id,theme,is_app_rail_docked,is_tour_completed,onboarding_step,is_onboarded,last_workspace_id,billing_address_country,has_billing_address,company_name,notification_view_mode,is_smooth_cursor_enabled,is_mobile_onboarded,mobile_onboarding_step,mobile_timezone_auto_set,language,start_of_the_week,goals,background_color,is_navigation_tour_completed,has_marketing_email_consent,is_subscribed_to_changelog,product_tour,created_at,updated_at)
 SELECT gen_random_uuid(),id,'{}',true,true,'{"profile_complete":true}',true,'00000000-0000-4000-8000-000000000001','','false','','full',false,false,'{}',false,'en',0,'{}','',false,false,false,'{}',now(),now() FROM users WHERE id IN ($1,$2)`, f.user, f.foreign)
	f.exec(t, `INSERT INTO workspace_members SELECT (jsonb_populate_record(NULL::workspace_members,to_jsonb(source)||jsonb_build_object('id',gen_random_uuid(),'member_id',u.id,'role',CASE WHEN u.id=$2 THEN 20 ELSE 15 END))).* FROM workspace_members source CROSS JOIN users u WHERE source.member_id='00000000-0000-4000-8000-000000000003' AND u.id IN ($1,$2)`, f.user, f.foreign)
	f.exec(t, `INSERT INTO project_members SELECT (jsonb_populate_record(NULL::project_members,to_jsonb(source)||jsonb_build_object('id',gen_random_uuid(),'member_id',u.id,'role',CASE WHEN u.id=$2 THEN 20 ELSE 15 END))).* FROM project_members source CROSS JOIN users u WHERE source.member_id='00000000-0000-4000-8000-000000000003' AND u.id IN ($1,$2)`, f.user, f.foreign)
	f.exec(t, `INSERT INTO workspace_member_invites(id,email,accepted,token,role,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),'original@example.test',false,'deactivation-proof',15,'00000000-0000-4000-8000-000000000001',now(),now())`)
	require.NoError(t, f.repo.QueueEmailCode(f.ctx, f.user, f.session, "change@example.test", "123456"))
	return f
}

func deactivationState(t *testing.T, f emailFixture) json.RawMessage {
	t.Helper()
	var state json.RawMessage
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT jsonb_build_array(
 (SELECT jsonb_agg(to_jsonb(u) ORDER BY id) FROM users u),
 (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM profiles p),
 (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM workspace_members m),
 (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM project_members m),
 (SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM workspace_member_invites i),
 (SELECT jsonb_agg(to_jsonb(s) ORDER BY session_key) FROM sessions s),
 (SELECT jsonb_agg(to_jsonb(c) ORDER BY user_id) FROM account_email_challenges c),
 (SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e))`).Scan(&state))
	return state
}

func TestDeactivationRollsBackAndCommitsAllEffects(t *testing.T) {
	f := newDeactivationFixture(t)
	f.exec(t, `CREATE FUNCTION reject_deactivation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.payload->>'name'='plane.bgtasks.user_deactivation_email_task.user_deactivation_email' THEN RAISE EXCEPTION 'notification unavailable'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_deactivation BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_deactivation()`)
	before := deactivationState(t, f)
	require.Error(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", "192.0.2.10"))
	require.JSONEq(t, string(before), string(deactivationState(t, f)), "notification admission failure must roll back every account change")
	f.exec(t, `DROP TRIGGER reject_deactivation ON work_item_effects`)
	f.exec(t, `DELETE FROM profiles WHERE user_id=$1`, f.user)
	before = deactivationState(t, f)
	require.ErrorIs(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", "192.0.2.10"), ErrNotFound)
	require.JSONEq(t, string(before), string(deactivationState(t, f)), "missing profile must not leave partial membership or session writes")
	f.exec(t, `INSERT INTO profiles SELECT (jsonb_populate_record(NULL::profiles,to_jsonb(p)||jsonb_build_object('id',gen_random_uuid(),'user_id',$1::uuid))).* FROM profiles p WHERE user_id=$2`, f.user, f.foreign)
	require.NoError(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", "192.0.2.10"))
	var disabled, profileReset, invitesRemoved, membershipsRemoved, foreignIntact bool
	var sessions, challenges int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT NOT is_active AND password='!' AND is_password_autoset AND last_logout_ip='192.0.2.10' AND last_logout_time IS NOT NULL,
 (SELECT last_workspace_id IS NULL AND NOT is_tour_completed AND NOT is_onboarded AND onboarding_step='{"workspace_join":false,"profile_complete":false,"workspace_create":false,"workspace_invite":false}' FROM profiles WHERE user_id=$1),
 NOT EXISTS(SELECT 1 FROM workspace_member_invites WHERE email=users.email AND deleted_at IS NULL),
 NOT EXISTS(SELECT 1 FROM workspace_members WHERE member_id=$1 AND is_active) AND NOT EXISTS(SELECT 1 FROM project_members WHERE member_id=$1 AND is_active),
 (SELECT is_active FROM users WHERE id=$2) AND EXISTS(SELECT 1 FROM sessions WHERE user_id=$2::text),
 (SELECT count(*) FROM sessions WHERE user_id=$1::text),(SELECT count(*) FROM account_email_challenges WHERE user_id=$1) FROM users WHERE id=$1`, f.user, f.foreign).Scan(&disabled, &profileReset, &invitesRemoved, &membershipsRemoved, &foreignIntact, &sessions, &challenges))
	require.True(t, disabled && profileReset && invitesRemoved && membershipsRemoved && foreignIntact)
	require.Zero(t, sessions)
	require.Zero(t, challenges)
	var args string
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT (payload->'args')::text FROM work_item_effects WHERE kind='email_task' AND payload->>'name'='plane.bgtasks.user_deactivation_email_task.user_deactivation_email'`).Scan(&args))
	require.JSONEq(t, `["https://vector.example.test","`+f.user.String()+`"]`, args)
	require.ErrorIs(t, f.repo.Deactivate(f.ctx, f.user, f.secondSession, "https://vector.example.test", ""), ErrUnauthenticated)
}

func TestDeactivationRechecksNativeSession(t *testing.T) {
	f := newDeactivationFixture(t)
	before := deactivationState(t, f)
	require.ErrorIs(t, f.repo.Deactivate(f.ctx, f.user, f.foreignSession, "https://vector.example.test", ""), ErrUnauthenticated)
	require.JSONEq(t, string(before), string(deactivationState(t, f)))
	// Password rotation invalidates native session signatures even if rows remain.
	f.exec(t, `UPDATE users SET password='!rotated' WHERE id=$1`, f.user)
	before = deactivationState(t, f)
	require.ErrorIs(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", ""), ErrUnauthenticated)
	require.JSONEq(t, string(before), string(deactivationState(t, f)))
}

func TestConcurrentDeactivationKeepsAnAdministrator(t *testing.T) {
	f := newDeactivationFixture(t)
	f.exec(t, `UPDATE workspace_members SET role=20 WHERE member_id=$1`, f.user)
	f.exec(t, `UPDATE project_members SET role=20 WHERE member_id=$1`, f.user)
	start, results := make(chan struct{}), make(chan error, 2)
	go func() {
		<-start
		results <- f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", "")
	}()
	go func() {
		<-start
		results <- f.repo.Deactivate(f.ctx, f.foreign, f.foreignSession, "https://vector.example.test", "")
	}()
	close(start)
	first, second := <-results, <-results
	if first == nil {
		first, second = second, first
	}
	var blocker DeactivationBlocked
	require.ErrorAs(t, first, &blocker)
	require.NoError(t, second)
	var workspaceAdmins, projectAdmins, notices int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM workspace_members WHERE role=20 AND is_active AND deleted_at IS NULL),(SELECT count(*) FROM project_members WHERE role=20 AND is_active AND deleted_at IS NULL),(SELECT count(*) FROM work_item_effects WHERE payload->>'name'='plane.bgtasks.user_deactivation_email_task.user_deactivation_email')`).Scan(&workspaceAdmins, &projectAdmins, &notices))
	require.Equal(t, 1, workspaceAdmins)
	require.Equal(t, 1, projectAdmins)
	require.Equal(t, 1, notices)
}

func TestDeactivationWaitsForMembershipChanges(t *testing.T) {
	f := newDeactivationFixture(t)
	f.exec(t, `UPDATE workspace_members SET role=20 WHERE member_id=$1`, f.user)
	f.exec(t, `UPDATE project_members SET role=20 WHERE member_id=$1`, f.user)
	tx, err := f.pool.Begin(f.ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		rollbackErr := tx.Rollback(context.Background())
		if !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			require.NoError(t, rollbackErr)
		}
	})
	_, err = tx.Exec(f.ctx, `UPDATE workspace_members SET role=15 WHERE member_id=$1`, f.foreign)
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() { result <- f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", "") }()
	require.Eventually(t, func() bool {
		var waiting bool
		err := f.pool.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'LOCK TABLE instance_admins,%')`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, tx.Commit(f.ctx))
	var blocker DeactivationBlocked
	require.ErrorAs(t, <-result, &blocker)
	require.Contains(t, blocker.Error(), "workspaces")
	require.Equal(t, 2, f.state(t).Sessions)
}

func TestDeactivationAdministratorResponsibilities(t *testing.T) {
	f := newDeactivationFixture(t)
	f.exec(t, `UPDATE workspace_members SET role=20 WHERE member_id=$1`, f.user)
	f.exec(t, `UPDATE project_members SET role=20 WHERE member_id=$1`, f.user)
	f.exec(t, `UPDATE project_members SET role=15 WHERE member_id=$1`, f.foreign)
	before := deactivationState(t, f)
	var blocker DeactivationBlocked
	require.ErrorAs(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", ""), &blocker)
	require.Contains(t, blocker.Error(), "projects")
	require.JSONEq(t, string(before), string(deactivationState(t, f)))
	f.exec(t, `UPDATE project_members SET role=20 WHERE member_id=$1`, f.foreign)
	f.exec(t, `UPDATE workspace_members SET role=15 WHERE member_id=$1`, f.foreign)
	require.ErrorAs(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", ""), &blocker)
	require.Contains(t, blocker.Error(), "workspaces")
	f.exec(t, `INSERT INTO instances(id,instance_name,instance_id,current_version,last_checked_at,is_telemetry_enabled,is_support_required,is_setup_done,is_signup_screen_visited,is_verified,domain,edition,is_test,is_current_version_deprecated,created_at,updated_at) VALUES(gen_random_uuid(),'Native account proof','deactivation','1',now(),false,false,true,true,true,'example.test','ce',true,false,now(),now())`)
	f.exec(t, `INSERT INTO instance_admins(id,instance_id,user_id,role,is_verified,created_at,updated_at) SELECT gen_random_uuid(),id,$1,20,true,now(),now() FROM instances`, f.user)
	require.ErrorAs(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", ""), &blocker)
	require.Contains(t, blocker.Error(), "instance admin")
	// Historical admin rows confer no responsibility. A solo member may leave
	// their account disabled without being forced to delete owned resources.
	f.exec(t, `UPDATE instance_admins SET deleted_at=now() WHERE user_id=$1`, f.user)
	f.exec(t, `UPDATE project_members SET is_active=false WHERE member_id<>$1`, f.user)
	f.exec(t, `UPDATE workspace_members SET is_active=false WHERE member_id<>$1`, f.user)
	require.NoError(t, f.repo.Deactivate(f.ctx, f.user, f.session, "https://vector.example.test", ""))
}
