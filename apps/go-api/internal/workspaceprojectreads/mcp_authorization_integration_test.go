//go:build integration

package workspaceprojectreads_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const projectReadSchema = `
CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean,user_timezone text,is_bot boolean DEFAULT false);
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,role int,is_active boolean,deleted_at timestamptz,created_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,name text,identifier text,description text,emoji text,icon_prop jsonb,cover_image text,cover_image_asset_id uuid,archived_at timestamptz,created_at timestamptz,updated_at timestamptz,deleted_at timestamptz,network int,
archive_in int,close_in int,created_by_id uuid,cycle_view boolean,default_assignee_id uuid,default_state_id uuid,description_html text,description_text text,estimate_id uuid,external_id text,external_source text,guest_view_all_features boolean,intake_view boolean,is_issue_type_enabled boolean,is_time_tracking_enabled boolean,issue_views_view boolean,logo_props jsonb,module_view boolean,page_view boolean,project_lead_id uuid,timezone text,updated_by_id uuid);
CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,sort_order double precision,is_active boolean,deleted_at timestamptz,created_at timestamptz,role int);
CREATE TABLE file_assets(id uuid PRIMARY KEY,entity_type text,workspace_id uuid,project_id uuid,issue_id uuid);
CREATE TABLE deploy_boards(anchor text,entity_name text,entity_identifier uuid,workspace_id uuid,deleted_at timestamptz,created_at timestamptz);
CREATE TABLE user_favorites(user_id uuid,entity_identifier uuid,entity_type text,project_id uuid,deleted_at timestamptz);
CREATE TABLE issue_sequences(project_id uuid,sequence bigint,deleted_at timestamptz);
CREATE TABLE project_user_properties(project_id uuid,workspace_id uuid,user_id uuid,deleted_at timestamptz,created_at timestamptz,sort_order double precision);
CREATE TABLE cycles(project_id uuid,deleted_at timestamptz);
CREATE TABLE modules(project_id uuid,deleted_at timestamptz);
INSERT INTO users VALUES('10000000-0000-4000-8000-000000000001',true,'UTC',false);
INSERT INTO workspaces VALUES('20000000-0000-4000-8000-000000000001','space',NULL);
INSERT INTO workspace_members VALUES('20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001',15,true,NULL,now());
INSERT INTO projects(id,workspace_id,name,identifier,created_at,network) VALUES('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','Workspace project','PROJ',now(),2);
`

type changeProjectAuthorization struct {
	db     *pgx.Conn
	t      *testing.T
	change string
	calls  int
}

func (r *changeProjectAuthorization) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r.calls++
	if r.calls == 2 {
		_, err := r.db.Exec(ctx, r.change)
		require.NoError(r.t, err)
	}
	return r.db.QueryRow(ctx, sql, args...)
}

type projectRead func(context.Context, *workspaceprojectreads.Repository) (json.RawMessage, error)

func projectReadOperations() map[string]projectRead {
	actor := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("30000000-0000-4000-8000-000000000001")
	return map[string]projectRead{
		"lite": func(ctx context.Context, reader *workspaceprojectreads.Repository) (json.RawMessage, error) {
			return reader.APIProjects(ctx, actor, "space", workspaceprojectreads.APIProjectListOptions{})
		},
		"list": func(ctx context.Context, reader *workspaceprojectreads.Repository) (json.RawMessage, error) {
			return reader.APIProjectList(ctx, actor, "space", workspaceprojectreads.APIProjectListOptions{})
		},
		"detail": func(ctx context.Context, reader *workspaceprojectreads.Repository) (json.RawMessage, error) {
			return reader.APIProject(ctx, actor, "space", project, workspaceprojectreads.APIProjectDetailOptions{})
		},
	}
}

func TestProjectReadsRecheckAuthorizationInDataSnapshot(t *testing.T) {
	for name, read := range projectReadOperations() {
		t.Run(name, func(t *testing.T) {
			for _, scenario := range []struct{ name, change string }{
				{"membership inactive", `UPDATE workspace_members SET is_active=false`},
				{"membership deleted", `UPDATE workspace_members SET deleted_at=now()`},
				{"membership role removed", `UPDATE workspace_members SET role=0`},
				{"workspace deleted", `UPDATE workspaces SET deleted_at=now()`},
				{"account inactive", `UPDATE users SET is_active=false`},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					ctx, db := postgres.OpenSQL(t, projectReadSchema)
					reader, err := workspaceprojectreads.NewRepository(db)
					require.NoError(t, err)
					before, err := read(ctx, reader)
					require.NoError(t, err)
					require.Contains(t, string(before), "Workspace project")
					changed := &changeProjectAuthorization{db: db, t: t, change: scenario.change}
					reader, err = workspaceprojectreads.NewRepository(changed)
					require.NoError(t, err)

					body, err := read(ctx, reader)

					require.ErrorIs(t, err, workspaceprojectreads.ErrForbidden)
					require.Empty(t, body)
				})
			}
		})
	}
}

func TestProjectReadsKeepPrivateProjectMembershipCurrent(t *testing.T) {
	for name, read := range projectReadOperations() {
		t.Run(name, func(t *testing.T) {
			ctx, db := postgres.OpenSQL(t, projectReadSchema+`
UPDATE projects SET network=0;
INSERT INTO project_members(project_id,workspace_id,member_id,is_active,role,created_at) VALUES('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001',true,15,now());`)
			reader, err := workspaceprojectreads.NewRepository(db)
			require.NoError(t, err)
			before, err := read(ctx, reader)
			require.NoError(t, err)
			require.Contains(t, string(before), "Workspace project")
			changed := &changeProjectAuthorization{db: db, t: t, change: `UPDATE project_members SET is_active=false`}
			reader, err = workspaceprojectreads.NewRepository(changed)
			require.NoError(t, err)

			body, err := read(ctx, reader)

			if name == "detail" {
				require.ErrorIs(t, err, workspaceprojectreads.ErrNotFound)
				require.Empty(t, body)
				return
			}
			require.NoError(t, err)
			var page struct {
				Count   int               `json:"count"`
				Results []json.RawMessage `json:"results"`
			}
			require.NoError(t, json.Unmarshal(body, &page))
			require.Zero(t, page.Count)
			require.Empty(t, page.Results)
		})
	}
}

func TestProjectReadsRejectReassignedWorkspaceSlug(t *testing.T) {
	actor := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	workspace := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("30000000-0000-4000-8000-000000000002")
	reads := map[string]projectRead{
		"lite": func(ctx context.Context, reader *workspaceprojectreads.Repository) (json.RawMessage, error) {
			return reader.APIProjects(ctx, actor, "space", workspaceprojectreads.APIProjectListOptions{ExpectedWorkspaceID: &workspace})
		},
		"list": func(ctx context.Context, reader *workspaceprojectreads.Repository) (json.RawMessage, error) {
			return reader.APIProjectList(ctx, actor, "space", workspaceprojectreads.APIProjectListOptions{ExpectedWorkspaceID: &workspace})
		},
		"detail": func(ctx context.Context, reader *workspaceprojectreads.Repository) (json.RawMessage, error) {
			return reader.APIProject(ctx, actor, "space", project, workspaceprojectreads.APIProjectDetailOptions{ExpectedWorkspaceID: &workspace})
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			ctx, db := postgres.OpenSQL(t, projectReadSchema+`
INSERT INTO workspaces VALUES('20000000-0000-4000-8000-000000000002','other',NULL);
INSERT INTO workspace_members VALUES('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',15,true,NULL,now());
INSERT INTO projects(id,workspace_id,name,identifier,created_at,network) VALUES('30000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000002','Different workspace project','OTHER',now(),2);`)
			changed := &changeProjectAuthorization{db: db, t: t, change: `UPDATE workspaces SET slug='renamed' WHERE slug='space'; UPDATE workspaces SET slug='space' WHERE slug='other'`}
			reader, err := workspaceprojectreads.NewRepository(changed)
			require.NoError(t, err)
			body, err := read(ctx, reader)
			require.ErrorIs(t, err, workspaceprojectreads.ErrForbidden)
			require.Empty(t, body)
		})
	}
}

func TestProjectCollectionsOrderBySupportedNetworkField(t *testing.T) {
	for _, lite := range []bool{false, true} {
		t.Run(fmt.Sprint(lite), func(t *testing.T) {
			ctx, db := postgres.OpenSQL(t, projectReadSchema+`
INSERT INTO projects(id,workspace_id,name,identifier,network,created_at) VALUES('30000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001','Private project','PRIVATE',0,now());
INSERT INTO project_members(project_id,workspace_id,member_id,is_active,role,created_at) VALUES('30000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001',true,15,now());`)
			reader, err := workspaceprojectreads.NewRepository(db)
			require.NoError(t, err)
			actor := uuid.MustParse("10000000-0000-4000-8000-000000000001")
			for _, test := range []struct {
				order string
				want  []string
			}{
				{"network", []string{"Private project", "Workspace project"}},
				{"-network", []string{"Workspace project", "Private project"}},
			} {
				t.Run(test.order, func(t *testing.T) {
					options := workspaceprojectreads.APIProjectListOptions{OrderBy: &test.order}
					var body json.RawMessage
					var err error
					if lite {
						body, err = reader.APIProjects(ctx, actor, "space", options)
					} else {
						body, err = reader.APIProjectList(ctx, actor, "space", options)
					}
					require.NoError(t, err)
					var result struct{ Results []struct{ Name string } }
					require.NoError(t, json.Unmarshal(body, &result))
					require.Len(t, result.Results, 2)
					require.Equal(t, test.want[0], result.Results[0].Name)
					require.Equal(t, test.want[1], result.Results[1].Name)
				})
			}
		})
	}
}
