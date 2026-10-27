//go:build integration

package bootstrap_test

import (
	"context"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type milestoneSchemaFixture struct {
	ctx                                                               context.Context
	db                                                                *pgx.Conn
	url                                                               string
	workspace, project, otherProject, milestone, otherMilestone, item string
}

func newMilestoneSchemaFixture(t *testing.T) milestoneSchemaFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgres.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close(context.WithoutCancel(ctx))) })
	f := milestoneSchemaFixture{ctx: ctx, db: db, url: pair.SeedURL(), workspace: uuid.New().String(), project: uuid.New().String(), otherProject: uuid.New().String(), milestone: uuid.New().String(), otherMilestone: uuid.New().String(), item: uuid.New().String()}
	owner := uuid.New().String()
	f.exec(t, `INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required) VALUES($1::uuid,$1::text,$1::text||'@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Milestones',true,false)`, owner)
	f.exec(t, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Milestones',$2,'milestones','UTC','',now(),now())`, f.workspace, owner)
	for _, project := range []string{f.project, f.otherProject} {
		f.exec(t, `INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES($1::uuid,$1::text,'',0,$2,left($1::text,12),false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, project, f.workspace)
	}
	f.exec(t, `INSERT INTO issues(id,workspace_id,project_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at) VALUES($1,$2,$3,'Work','{}','','none',0,1,false,now(),now())`, f.item, f.workspace, f.project)
	f.exec(t, `INSERT INTO milestones(id,workspace_id,project_id,title) VALUES($1,$2,$3,'Release'),($4,$2,$3,'Follow-up')`, f.milestone, f.workspace, f.project, f.otherMilestone)
	return f
}

func (f milestoneSchemaFixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := f.db.Exec(f.ctx, query, args...)
	require.NoError(t, err)
}

func requireMilestoneSQLState(t *testing.T, err error, state string) {
	t.Helper()
	var failure *pgconn.PgError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, state, failure.Code)
}

func (f milestoneSchemaFixture) revision(t *testing.T, id string) int64 {
	t.Helper()
	var revision int64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM milestone_edit_revisions WHERE milestone_id=$1`, id).Scan(&revision))
	return revision
}

const insertMilestoneLink = `INSERT INTO milestone_issues(milestone_id,issue_id,workspace_id,project_id) VALUES($1,$2,$3,$4)`

func TestMilestoneSchemaRejectsInvalidTitlesDatesAndOwnership(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	for _, title := range []string{"", " ", " Release", "Release\n", "\tRelease", "Release\r", "Release\v", "\fRelease"} {
		_, err := f.db.Exec(f.ctx, `INSERT INTO milestones(workspace_id,project_id,title) VALUES($1,$2,$3)`, f.workspace, f.project, title)
		requireMilestoneSQLState(t, err, "23514")
	}
	_, err := f.db.Exec(f.ctx, `INSERT INTO milestones(workspace_id,project_id,title) VALUES($1,$2,'Release')`, f.workspace, f.project)
	requireMilestoneSQLState(t, err, "23505")
	f.exec(t, `INSERT INTO milestones(workspace_id,project_id,title,target_date) VALUES($1,$2,'release','0001-01-01'),($1,$3,'Release','9999-12-31')`, f.workspace, f.project, f.otherProject)
	f.exec(t, `INSERT INTO milestones(workspace_id,project_id,title,external_source) VALUES($1,$2,$3,'  source  ')`, f.workspace, f.project, strings.Repeat("雪", 255))
	_, err = f.db.Exec(f.ctx, `UPDATE milestones SET title=$2 WHERE id=$1`, f.milestone, strings.Repeat("雪", 256))
	requireMilestoneSQLState(t, err, "22001")
	_, err = f.db.Exec(f.ctx, `UPDATE milestones SET external_id=$2 WHERE id=$1`, f.milestone, strings.Repeat("雪", 256))
	requireMilestoneSQLState(t, err, "22001")
	for _, date := range []string{"0001-01-01 BC", "10000-01-01", "infinity", "-infinity"} {
		_, err = f.db.Exec(f.ctx, `UPDATE milestones SET target_date=$2::date WHERE id=$1`, f.milestone, date)
		requireMilestoneSQLState(t, err, "23514")
	}
	_, err = f.db.Exec(f.ctx, `UPDATE milestones SET project_id=$2 WHERE id=$1`, f.milestone, f.otherProject)
	requireMilestoneSQLState(t, err, "23514")
	_, err = f.db.Exec(f.ctx, `INSERT INTO milestones(workspace_id,project_id,title) VALUES(gen_random_uuid(),$1,'Foreign')`, f.project)
	requireMilestoneSQLState(t, err, "23503")
	require.EqualValues(t, 1, f.revision(t, f.milestone))
	require.NoError(t, bootstrap.MigrateSchema(f.ctx, f.url))
	require.EqualValues(t, 1, f.revision(t, f.milestone))
}

func TestMilestoneSchemaLiveOwnershipAndHistoricalItemMoves(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	f.exec(t, insertMilestoneLink, f.milestone, f.item, f.workspace, f.project)
	_, err := f.db.Exec(f.ctx, `UPDATE milestone_issues SET id=gen_random_uuid() WHERE issue_id=$1`, f.item)
	requireMilestoneSQLState(t, err, "23514")
	_, err = f.db.Exec(f.ctx, insertMilestoneLink, f.otherMilestone, f.item, f.workspace, f.project)
	requireMilestoneSQLState(t, err, "23505")
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET project_id=$2 WHERE id=$1`, f.item, f.otherProject)
	requireMilestoneSQLState(t, err, "23503")
	_, err = f.db.Exec(f.ctx, `UPDATE milestones SET deleted_at=now() WHERE id=$1`, f.milestone)
	requireMilestoneSQLState(t, err, "23503")
	_, err = f.db.Exec(f.ctx, `UPDATE milestone_issues SET project_id=$2 WHERE issue_id=$1`, f.item, f.otherProject)
	requireMilestoneSQLState(t, err, "23503")
	f.exec(t, `UPDATE milestone_issues SET deleted_at=now() WHERE issue_id=$1`, f.item)
	f.exec(t, `UPDATE issues SET project_id=$2 WHERE id=$1`, f.item, f.otherProject)
	_, err = f.db.Exec(f.ctx, `UPDATE milestone_issues SET deleted_at=NULL WHERE issue_id=$1`, f.item)
	requireMilestoneSQLState(t, err, "23503")
	f.exec(t, `UPDATE issues SET project_id=$2 WHERE id=$1`, f.item, f.project)
	f.exec(t, `UPDATE milestones SET deleted_at=now() WHERE id=$1`, f.milestone)
	_, err = f.db.Exec(f.ctx, insertMilestoneLink, f.milestone, f.item, f.workspace, f.project)
	requireMilestoneSQLState(t, err, "23503")
	f.exec(t, insertMilestoneLink, f.otherMilestone, f.item, f.workspace, f.project)
	var distinctIDs int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(DISTINCT id) FROM milestone_issues WHERE issue_id=$1`, f.item).Scan(&distinctIDs))
	require.Equal(t, 2, distinctIDs)
}

func TestMilestoneSchemaRevisionsFollowLiveSetAndRollback(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	f.exec(t, `UPDATE milestones SET title='First release' WHERE id=$1`, f.milestone)
	require.EqualValues(t, 2, f.revision(t, f.milestone))
	f.exec(t, `UPDATE milestones SET title=title WHERE id=$1`, f.milestone)
	require.EqualValues(t, 2, f.revision(t, f.milestone))
	f.exec(t, insertMilestoneLink, f.milestone, f.item, f.workspace, f.project)
	require.EqualValues(t, 3, f.revision(t, f.milestone))
	f.exec(t, `UPDATE milestone_issues SET updated_at=now() WHERE issue_id=$1`, f.item)
	require.EqualValues(t, 3, f.revision(t, f.milestone))
	f.exec(t, `UPDATE milestone_issues SET milestone_id=$2 WHERE issue_id=$1`, f.item, f.otherMilestone)
	require.EqualValues(t, 4, f.revision(t, f.milestone))
	require.EqualValues(t, 2, f.revision(t, f.otherMilestone))
	tx, err := f.db.Begin(f.ctx)
	require.NoError(t, err)
	_, err = tx.Exec(f.ctx, `UPDATE milestone_issues SET deleted_at=now() WHERE issue_id=$1`, f.item)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(f.ctx))
	require.EqualValues(t, 2, f.revision(t, f.otherMilestone))
	f.exec(t, `UPDATE milestone_issues SET deleted_at=now() WHERE issue_id=$1`, f.item)
	require.EqualValues(t, 3, f.revision(t, f.otherMilestone))
	f.exec(t, `DELETE FROM milestone_issues WHERE issue_id=$1`, f.item)
	require.EqualValues(t, 3, f.revision(t, f.otherMilestone))
	f.exec(t, insertMilestoneLink, f.otherMilestone, f.item, f.workspace, f.project)
	f.exec(t, `DELETE FROM issues WHERE id=$1`, f.item)
	require.EqualValues(t, 5, f.revision(t, f.otherMilestone))
	f.exec(t, `UPDATE milestones SET deleted_at=now() WHERE id=$1`, f.otherMilestone)
	require.EqualValues(t, 6, f.revision(t, f.otherMilestone))
	f.exec(t, `DELETE FROM projects WHERE id=$1`, f.project)
	var count int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM milestone_edit_revisions`).Scan(&count))
	require.Zero(t, count)
}

func TestMilestoneSchemaPhysicalDeletionPreservesItems(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	f.exec(t, insertMilestoneLink, f.milestone, f.item, f.workspace, f.project)
	f.exec(t, `DELETE FROM milestones WHERE id=$1`, f.milestone)
	var itemExists bool
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM issues WHERE id=$1)`, f.item).Scan(&itemExists))
	require.True(t, itemExists)
	var links, revisions int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM milestone_issues`).Scan(&links))
	require.Zero(t, links)
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM milestone_edit_revisions WHERE milestone_id=$1`, f.milestone).Scan(&revisions))
	require.Zero(t, revisions)
	require.EqualValues(t, 1, f.revision(t, f.otherMilestone))
}
