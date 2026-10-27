//go:build integration

package milestones_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type writeFixture struct {
	ctx  context.Context
	db   *pgx.Conn
	repo *milestones.Repository
	opts milestones.WriteOptions
}

func newWriteFixture(t *testing.T) writeFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close(context.WithoutCancel(ctx))) })
	config, err := pgxpool.ParseConfig(pair.SeedURL())
	require.NoError(t, err)
	config.MaxConns = 2
	config.ConnConfig.RuntimeParams["application_name"] = "milestone-writes"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	repo, err := milestones.NewRepository(pool)
	require.NoError(t, err)
	_, err = db.Exec(ctx, milestoneWriteSeed)
	require.NoError(t, err)
	return writeFixture{ctx, db, repo, milestones.WriteOptions{Scope: milestones.Scope{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "milestones", Timezone: "UTC"}, Origin: "https://example.test"}}
}

func textField(value string) milestones.StringField {
	return milestones.StringField{Present: true, Value: &value}
}

func (f writeFixture) counts(t *testing.T, records, effects, receipts int) {
	t.Helper()
	var gotRecords, gotEffects, gotReceipts int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM milestones),(SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM mutation_receipts)`).Scan(&gotRecords, &gotEffects, &gotReceipts))
	require.Equal(t, records, gotRecords)
	require.Equal(t, effects, gotEffects)
	require.Equal(t, receipts, gotReceipts)
}

const milestoneWriteSeed = `
INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required)
VALUES('00000000-0000-4000-8000-000000000003','writer','writer@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Writer',true,false);
INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at)
VALUES('00000000-0000-4000-8000-000000000001','Milestones','00000000-0000-4000-8000-000000000003','milestones','UTC','',now(),now());
INSERT INTO workspace_members(id,workspace_id,member_id,role,created_at,updated_at,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features)
VALUES(gen_random_uuid(),'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000003',15,now(),now(),'{}','{}','{}',true,'{}','{}','{}');
INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at)
VALUES('00000000-0000-4000-8000-000000000002','Milestones','',0,'00000000-0000-4000-8000-000000000001','MILE',false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now());
INSERT INTO project_members(id,workspace_id,project_id,member_id,role,created_at,updated_at,view_props,default_props,sort_order,preferences,is_active)
VALUES(gen_random_uuid(),'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003',15,now(),now(),'{}','{}',0,'{}',true);
`
