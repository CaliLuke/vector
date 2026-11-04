//go:build integration

package workitemtypes_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

var typeSchema = postgresfixture.NewTemplate(seedTypeSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := typeSchema.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close work-item type test template:", err)
		status = 1
	}
	os.Exit(status)
}

// Only immutable fixture data belongs in the template. Each test still gets a
// private database, so writes, triggers, concurrency and rollback stay isolated.
const (
	typeOwner     = "10000000-0000-4000-8000-000000000001"
	typeMember    = "10000000-0000-4000-8000-000000000002"
	typeOutsider  = "10000000-0000-4000-8000-000000000003"
	typeWorkspace = "10000000-0000-4000-8000-000000000004"
	typeProject   = "10000000-0000-4000-8000-000000000005"
)

func seedTypeSchema(ctx context.Context, databaseURL string) (resultErr error) {
	if err := bootstrap.MigrateSchema(ctx, databaseURL); err != nil {
		return err
	}
	db, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, db.Close(context.WithoutCancel(ctx))) }()
	owner, member, outsider, workspace := typeOwner, typeMember, typeOutsider, typeWorkspace
	batch := &pgx.Batch{}
	for _, user := range []string{owner, member, outsider} {
		batch.Queue(`INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required) VALUES($1::uuid,$1::text,$1::text||'@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Types',true,false)`, user)
	}
	batch.Queue(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Types',$2,'types','UTC','',now(),now())`, workspace, owner)
	for _, user := range []string{owner, member} {
		role := 15
		if user == owner {
			role = 20
		}
		batch.Queue(`INSERT INTO workspace_members(id,workspace_id,member_id,role,created_at,updated_at,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features) VALUES(gen_random_uuid(),$1,$2,$3,now(),now(),'{}','{}','{}',true,'{}','{}','{}')`, workspace, user, role)
	}
	batch.Queue(`INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES($1::uuid,$1::text,'',0,$2,left($1::text,12),false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, typeProject, workspace)
	return db.SendBatch(ctx, batch).Close()
}
