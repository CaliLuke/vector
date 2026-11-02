//go:build integration

package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const mutationWorkspace = "00000000-0000-4000-8000-000000000001"
const mutationProject = "00000000-0000-4000-8000-000000000002"
const mutationActor = "00000000-0000-4000-8000-000000000003"
const mutationPeer = "00000000-0000-4000-8000-000000000004"
const mutationNew = "00000000-0000-4000-8000-000000000005"

type memberFixture struct {
	ctx        context.Context
	db         *pgxpool.Pool
	projects   *LifecycleRepository
	workspaces *InvitationsRepository
}

func newMemberFixture(t *testing.T) memberFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20;
 INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(u)||jsonb_build_object('id',ids.id,'username',ids.id::text,'email',ids.id::text||'@example.test'))).* FROM users u CROSS JOIN unnest(ARRAY['00000000-0000-4000-8000-000000000004'::uuid,'00000000-0000-4000-8000-000000000005'::uuid]) AS ids(id) WHERE u.id='00000000-0000-4000-8000-000000000003';
 INSERT INTO workspace_members SELECT (jsonb_populate_record(NULL::workspace_members,to_jsonb(wm)||jsonb_build_object('id',gen_random_uuid(),'member_id',u.id,'role',15))).* FROM workspace_members wm CROSS JOIN users u WHERE wm.member_id='00000000-0000-4000-8000-000000000003' AND u.id IN ('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000005');
 INSERT INTO project_members SELECT (jsonb_populate_record(NULL::project_members,to_jsonb(pm)||jsonb_build_object('id',gen_random_uuid(),'member_id','00000000-0000-4000-8000-000000000004','role',15))).* FROM project_members pm WHERE pm.member_id='00000000-0000-4000-8000-000000000003'`)
	require.NoError(t, err)
	projects, err := NewLifecycleRepository(pool, "https://vector.example.test")
	require.NoError(t, err)
	workspaces, err := NewInvitationsRepositoryWithReaderWriter(pool, pool, &lifecycleTaskSpy{}, "https://vector.example.test", "member-test-secret")
	require.NoError(t, err)
	return memberFixture{ctx, pool, projects, workspaces}
}

func (f memberFixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := f.db.Exec(f.ctx, query, args...)
	require.NoError(t, err)
}
func (f memberFixture) id(t *testing.T, table, user string) string {
	t.Helper()
	var id string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT id::text FROM `+table+` WHERE member_id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, user, mutationWorkspace).Scan(&id))
	return id
}
func (f memberFixture) project(actor, operation, method, id, body string) (any, error) {
	return f.projects.ExecuteLifecycle(f.ctx, LifecycleCommand{Operation: operation, Method: method, Slug: "milestones", ProjectID: mutationProject, UserID: actor, MemberID: id, Body: strings.NewReader(body), ContentType: "application/json"})
}
func (f memberFixture) workspace(actor, operation, id, body string) (json.RawMessage, error) {
	return f.workspaces.Invitation(f.ctx, InvitationCommand{Operation: operation, Slug: "milestones", UserID: actor, ID: id, Body: strings.NewReader(body), ContentType: "application/json"})
}
func (f memberFixture) snapshot(t *testing.T) string {
	t.Helper()
	var result string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT jsonb_build_array(
 (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM workspace_members m),
 (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM project_members m),
 (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM project_user_properties p),
 (SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e))::text`).Scan(&result))
	return result
}
func memberFailure(t *testing.T, err error, status int) {
	t.Helper()
	var failure *apicontract.HTTPError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, status, failure.Status)
}

func TestNativeBulkMembershipAndEmailAdmissionAreAtomic(t *testing.T) {
	f := newMemberFixture(t)
	body := `{"members":[{"member_id":"` + mutationPeer + `","role":5},{"member_id":"` + mutationNew + `","role":15}]}`

	f.exec(t, `ALTER TABLE project_user_properties ADD CONSTRAINT unavailable_member_property CHECK(user_id<>'00000000-0000-4000-8000-000000000005')`)
	before := f.snapshot(t)
	_, err := f.project(mutationActor, "legacy-create-members", "POST", "", body)
	require.Error(t, err)
	require.JSONEq(t, before, f.snapshot(t), "property failure must roll back the batch before any email admission")
	f.exec(t, `ALTER TABLE project_user_properties DROP CONSTRAINT unavailable_member_property`)
	f.exec(t, `CREATE FUNCTION reject_member_email() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'email admission unavailable'; END $$; CREATE TRIGGER reject_member_email BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_member_email()`)
	before = f.snapshot(t)
	_, err = f.project(mutationActor, "legacy-create-members", "POST", "", body)
	require.Error(t, err)
	require.JSONEq(t, before, f.snapshot(t), "late email admission must roll back updated members, new members and properties")
	f.exec(t, `DROP TRIGGER reject_member_email ON work_item_effects`)
	value, err := f.project(mutationActor, "legacy-create-members", "POST", "", body)
	require.NoError(t, err)
	require.Len(t, value, 2)
	var role, members, properties, emails int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT role FROM project_members WHERE member_id=$1),(SELECT count(*) FROM project_members),(SELECT count(*) FROM project_user_properties),(SELECT count(*) FROM work_item_effects WHERE kind='email_task')`, mutationPeer).Scan(&role, &members, &properties, &emails))
	require.Equal(t, 5, role)
	require.Equal(t, 3, members)
	require.Equal(t, 2, properties)
	require.Equal(t, 1, emails)
	var task, args string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT payload->>'name',(payload->'args')::text FROM work_item_effects WHERE kind='email_task'`).Scan(&task, &args))
	require.Equal(t, "plane.bgtasks.project_add_user_email_task.project_add_user_email", task)
	require.JSONEq(t, `["https://vector.example.test","`+f.id(t, "project_members", mutationNew)+`","`+mutationActor+`"]`, args)
	// Updating an existing active member must not resend an addition email.
	_, err = f.project(mutationActor, "legacy-create-members", "POST", "", body)
	require.NoError(t, err)
	var after int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&after))
	require.Equal(t, emails, after)
}

func TestMembershipIdentityAndTenantAreImmutable(t *testing.T) {
	f := newMemberFixture(t)
	f.exec(t, `INSERT INTO workspaces SELECT (jsonb_populate_record(NULL::workspaces,to_jsonb(w)||jsonb_build_object('id','00000000-0000-4000-8000-000000000010','slug','foreign'))).* FROM workspaces w WHERE id=$1`, mutationWorkspace)
	f.exec(t, `INSERT INTO projects SELECT (jsonb_populate_record(NULL::projects,to_jsonb(p)||jsonb_build_object('id','00000000-0000-4000-8000-000000000011','workspace_id','00000000-0000-4000-8000-000000000010'))).* FROM projects p WHERE id=$1`, mutationProject)
	f.exec(t, `INSERT INTO project_members SELECT (jsonb_populate_record(NULL::project_members,to_jsonb(pm)||jsonb_build_object('id','00000000-0000-4000-8000-000000000012','project_id','00000000-0000-4000-8000-000000000011','workspace_id','00000000-0000-4000-8000-000000000010'))).* FROM project_members pm WHERE member_id=$1`, mutationPeer)
	before := f.snapshot(t)
	_, err := f.workspace(mutationActor, "member-patch", f.id(t, "workspace_members", mutationPeer), `{"workspace":"00000000-0000-4000-8000-000000000010","created_by":"`+mutationPeer+`"}`)
	memberFailure(t, err, 400)
	_, err = f.project(mutationActor, "v1-patch-members", "PATCH", f.id(t, "project_members", mutationPeer), `{"member":"`+mutationNew+`"}`)
	memberFailure(t, err, 400)

	_, err = f.project(mutationActor, "v1-patch-members", "PATCH", "00000000-0000-4000-8000-000000000012", `{"role":20}`)
	memberFailure(t, err, 404)
	require.JSONEq(t, before, f.snapshot(t))
	// A workspace member cannot change roles or remove another member.
	_, err = f.workspace(mutationPeer, "member-delete", f.id(t, "workspace_members", mutationNew), "")
	memberFailure(t, err, 403)
	_, err = f.project(mutationPeer, "legacy-patch-members", "PATCH", f.id(t, "project_members", mutationActor), `{"role":5}`)
	memberFailure(t, err, 403)
	require.JSONEq(t, before, f.snapshot(t))
}

func TestWorkspaceGuestCascadeAndLastProjectAdministratorRollBackTogether(t *testing.T) {
	f := newMemberFixture(t)
	f.exec(t, `UPDATE workspace_members SET role=20 WHERE member_id=$1`, mutationPeer)
	before := f.snapshot(t)
	_, err := f.workspace(mutationPeer, "member-patch", f.id(t, "workspace_members", mutationActor), `{"role":5}`)
	memberFailure(t, err, 400)
	require.JSONEq(t, before, f.snapshot(t))
	// Once another project administrator exists, the same cascade may commit.
	f.exec(t, `UPDATE project_members SET role=20 WHERE member_id=$1`, mutationPeer)
	f.exec(t, `CREATE FUNCTION reject_workspace_save() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'member save unavailable'; END $$; CREATE TRIGGER reject_workspace_save BEFORE UPDATE ON workspace_members FOR EACH ROW EXECUTE FUNCTION reject_workspace_save()`)
	before = f.snapshot(t)
	_, err = f.workspace(mutationPeer, "member-patch", f.id(t, "workspace_members", mutationActor), `{"role":5}`)
	require.Error(t, err)
	require.JSONEq(t, before, f.snapshot(t), "a failed member save must roll back its project-role cascade")
	f.exec(t, `DROP TRIGGER reject_workspace_save ON workspace_members`)
	_, err = f.workspace(mutationPeer, "member-patch", f.id(t, "workspace_members", mutationActor), `{"role":5}`)
	require.NoError(t, err)
	var workspaceRole, projectRole int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT role FROM workspace_members WHERE member_id=$1),(SELECT role FROM project_members WHERE member_id=$1)`, mutationActor).Scan(&workspaceRole, &projectRole))
	require.Equal(t, 5, workspaceRole)
	require.Equal(t, 5, projectRole)
}

func TestConcurrentWorkspaceAndProjectDeparturesKeepAnAdministrator(t *testing.T) {
	f := newMemberFixture(t)
	f.exec(t, `UPDATE workspace_members SET role=20 WHERE member_id=$1`, mutationPeer)
	f.exec(t, `UPDATE project_members SET role=20 WHERE member_id=$1`, mutationPeer)
	start, results := make(chan struct{}), make(chan error, 2)
	go func() { <-start; _, err := f.workspace(mutationActor, "member-leave", "", ""); results <- err }()
	go func() {
		<-start
		_, err := f.project(mutationPeer, "legacy-leave-members", "POST", "", "")
		results <- err
	}()
	close(start)
	first, second := <-results, <-results
	if first == nil {
		first, second = second, first
	}
	memberFailure(t, first, 400)
	require.NoError(t, second)
	var admins int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM project_members WHERE role=20 AND is_active AND deleted_at IS NULL`).Scan(&admins))
	require.Equal(t, 1, admins)
}

func TestMembershipWriteRechecksPermissionAfterWaiting(t *testing.T) {
	f := newMemberFixture(t)
	id := f.id(t, "workspace_members", mutationPeer)
	tx, err := f.db.Begin(f.ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		rollbackErr := tx.Rollback(context.Background())
		if !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			require.NoError(t, rollbackErr)
		}
	})
	_, err = tx.Exec(f.ctx, `UPDATE workspace_members SET role=15 WHERE member_id=$1`, mutationActor)
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() {
		_, writeErr := f.workspace(mutationActor, "member-patch", id, `{"role":20}`)
		result <- writeErr
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		readErr := f.db.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'LOCK TABLE workspace_members,%')`).Scan(&waiting)
		return readErr == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, tx.Commit(f.ctx))
	memberFailure(t, <-result, 403)
	var role int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT role FROM workspace_members WHERE id=$1`, id).Scan(&role))
	require.Equal(t, 15, role)
}

func TestNativeProjectMemberCRUDKeepsRoleAndIdentityBoundaries(t *testing.T) {
	f := newMemberFixture(t)
	value, err := f.project(mutationActor, "v1-create-members", "POST", "", `{"member":"`+mutationNew+`","role":15}`)
	require.NoError(t, err)
	created := value.(map[string]any)
	id := created["id"].(string)
	require.Equal(t, mutationNew, created["member"])
	before := f.snapshot(t)
	_, err = f.project(mutationActor, "v1-create-members", "POST", "", `{"member":"`+mutationNew+`","role":15}`)
	memberFailure(t, err, 400)
	require.JSONEq(t, before, f.snapshot(t), "duplicate creation cannot leave a property or another email")
	_, err = f.workspace(mutationActor, "member-patch", f.id(t, "workspace_members", mutationNew), `{"role":5}`)
	require.NoError(t, err)
	before = f.snapshot(t)
	_, err = f.project(mutationActor, "v1-patch-members", "PATCH", id, `{"role":20}`)
	memberFailure(t, err, 400)
	require.JSONEq(t, before, f.snapshot(t), "a workspace guest cannot gain project administrator access")
	_, err = f.workspace(mutationActor, "member-patch", f.id(t, "workspace_members", mutationNew), `{"role":15}`)
	require.NoError(t, err)
	f.exec(t, `UPDATE workspace_members SET role=15 WHERE member_id=$1`, mutationActor)
	_, err = f.project(mutationActor, "v1-patch-members", "PATCH", id, `{"role":20,"is_active":false}`)
	require.NoError(t, err)
	value, err = f.project(mutationActor, "legacy-patch-members", "PATCH", id, `{"is_active":true}`)
	require.NoError(t, err)
	require.Equal(t, mutationNew, value.(map[string]any)["member"])
	var active bool
	var role int
	var createdBy, updatedBy string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT is_active,role,created_by_id::text,updated_by_id::text FROM project_members WHERE id=$1`, id).Scan(&active, &role, &createdBy, &updatedBy))
	require.True(t, active)
	require.Equal(t, 20, role)
	require.Equal(t, mutationActor, createdBy)
	require.Equal(t, mutationActor, updatedBy)
	before = f.snapshot(t)
	_, err = f.project(mutationActor, "v1-patch-members", "PATCH", f.id(t, "project_members", mutationActor), `{"role":5}`)
	memberFailure(t, err, 400)
	require.JSONEq(t, before, f.snapshot(t))
	f.exec(t, `UPDATE workspace_members SET role=20 WHERE member_id=$1`, mutationActor)
	actorMember := f.id(t, "project_members", mutationActor)
	_, err = f.project(mutationActor, "legacy-patch-members", "PATCH", actorMember, `{"role":15}`)
	require.NoError(t, err)
	_, err = f.project(mutationActor, "legacy-patch-members", "PATCH", actorMember, `{"role":20}`)
	require.NoError(t, err)
	_, err = f.project(mutationActor, "v1-delete-members", "DELETE", id, "")
	require.NoError(t, err)

	f.exec(t, `UPDATE project_members SET role=20 WHERE member_id=$1`, mutationPeer)
	f.exec(t, `UPDATE users SET is_active=false WHERE id=$1`, mutationPeer)
	before = f.snapshot(t)
	_, err = f.project(mutationActor, "legacy-leave-members", "POST", "", "")
	memberFailure(t, err, 400)
	_, err = f.workspace(mutationActor, "member-leave", "", "")
	memberFailure(t, err, 400)
	require.JSONEq(t, before, f.snapshot(t), "the last project and workspace administrator must remain")
	_, err = f.project(mutationActor, "v1-patch-members", "PATCH", f.id(t, "project_members", mutationPeer), `{"is_active":true}`)
	memberFailure(t, err, 400)
	_, err = f.workspace(mutationActor, "member-delete", f.id(t, "workspace_members", mutationPeer), "")
	require.NoError(t, err)
	var workspaceActive, projectActive bool
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT is_active FROM workspace_members WHERE member_id=$1),(SELECT is_active FROM project_members WHERE member_id=$1)`, mutationPeer).Scan(&workspaceActive, &projectActive))
	require.False(t, workspaceActive)
	require.False(t, projectActive)
}
