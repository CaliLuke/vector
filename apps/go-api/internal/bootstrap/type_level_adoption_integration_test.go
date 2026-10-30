//go:build integration

package bootstrap

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestAdoptionRepairsNonfiniteTypeLevelBeforeCatalogueReads(t *testing.T) {
	url := postgresfixture.EmptyDatabase(t)
	db, err := pgx.Connect(t.Context(), url)
	require.NoError(t, err)
	t.Cleanup(func() { testsupport.CloseContext(t, db, context.WithoutCancel(t.Context())) })
	baseline, err := schemaFiles.ReadFile("schema_migrations/000001_django_baseline.up.sql")
	require.NoError(t, err)
	_, err = db.Exec(t.Context(), string(baseline))
	require.NoError(t, err)
	_, err = db.Exec(t.Context(), `INSERT INTO public.users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required)
 VALUES('00000000-0000-4000-8000-000000000001','level-adoption','level-adoption@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Legacy',true,false);
 INSERT INTO public.workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at)
 VALUES('00000000-0000-4000-8000-000000000002','Legacy','00000000-0000-4000-8000-000000000001','legacy','UTC','',now(),now());
 INSERT INTO public.issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,is_epic,level,created_at,updated_at,external_id,external_source)
 VALUES('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002','Legacy type','retained','{}',true,false,false,'NaN','2020-01-02','2020-01-03','legacy-type','tracker');`)
	require.NoError(t, err)
	var metadata string
	require.NoError(t, db.QueryRow(t.Context(), `SELECT (to_jsonb(t)-'level')::text FROM public.issue_types t`).Scan(&metadata))
	require.NoError(t, MigrateSchema(t.Context(), url))
	var level int
	var preserved bool
	require.NoError(t, db.QueryRow(t.Context(), `SELECT level::integer,(to_jsonb(t)-'level') @> $1::jsonb FROM public.issue_types t`, metadata).Scan(&level, &preserved))
	require.Zero(t, level)
	require.True(t, preserved, "adoption must retain every original metadata field")
}
