//go:build integration

package workspaceexports

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const historyVisibilityFixtureDDL = `
CREATE TABLE workspace_members (workspace_id uuid,member_id uuid,role smallint,is_active boolean,deleted_at timestamptz);
CREATE TABLE projects (id uuid,workspace_id uuid,deleted_at timestamptz,guest_view_all_features boolean);
CREATE TABLE project_members (project_id uuid,workspace_id uuid,member_id uuid,role smallint,is_active boolean,deleted_at timestamptz);
CREATE TABLE issues (id uuid,workspace_id uuid,project_id uuid,created_by_id uuid,deleted_at timestamptz,is_draft boolean);
`

func TestCustomExportHistoryRequiresOwnerAndCurrentSourceVisibility(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := connection.Close(context.WithoutCancel(ctx)); closeErr != nil {
				t.Error(closeErr)
			}
		}()
		_, err = connection.Exec(ctx, historyVisibilityFixtureDDL+`
CREATE TABLE workspaces (id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE users (id uuid PRIMARY KEY,first_name text,last_name text,display_name text,avatar text,avatar_asset_id uuid,is_bot boolean);
CREATE TABLE exporters (id uuid PRIMARY KEY,created_at timestamptz,updated_at timestamptz,type text,workspace_id uuid,project uuid[],provider text,status text,reason text,key text,url text,initiated_by_id uuid,token text,created_by_id uuid,updated_by_id uuid,deleted_at timestamptz,rich_filters jsonb);
INSERT INTO workspaces VALUES ('00000000-0000-4000-8000-000000000001','demo',NULL);
INSERT INTO users(id) VALUES ('00000000-0000-4000-8000-000000000002'),('00000000-0000-4000-8000-000000000003');
INSERT INTO workspace_members VALUES ('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',15,true,NULL),('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000003',20,true,NULL);
INSERT INTO projects VALUES ('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001',NULL,false);
INSERT INTO project_members VALUES ('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',15,true,NULL);
INSERT INTO issues VALUES ('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000003',NULL,false);
INSERT INTO exporters(id,created_at,updated_at,type,workspace_id,project,provider,status,initiated_by_id,token,url,rich_filters) VALUES
 ('00000000-0000-4000-8000-000000000011',NOW(),NOW(),'issue_exports','00000000-0000-4000-8000-000000000001',ARRAY['00000000-0000-4000-8000-000000000004']::uuid[],'csv','completed','00000000-0000-4000-8000-000000000002','custom','private-url','{"include_custom_properties":true,"exported_item_ids":["00000000-0000-4000-8000-000000000005"]}'),
 ('00000000-0000-4000-8000-000000000012',NOW(),NOW(),'issue_exports','00000000-0000-4000-8000-000000000001',ARRAY['00000000-0000-4000-8000-000000000004']::uuid[],'csv','completed','00000000-0000-4000-8000-000000000002','standard','standard-url','{}');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := connection.Close(context.WithoutCancel(ctx)); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	for _, test := range []struct {
		name, actor, change string
		wantCustom          bool
	}{
		{name: "owner retains source access", actor: "00000000-0000-4000-8000-000000000002", wantCustom: true},
		{name: "other workspace admin", actor: "00000000-0000-4000-8000-000000000003"},
		{name: "project membership revoked", actor: "00000000-0000-4000-8000-000000000002", change: `UPDATE project_members SET is_active=false`},
		{name: "project guest cannot see another author's item", actor: "00000000-0000-4000-8000-000000000002", change: `UPDATE project_members SET role=5`},
		{name: "workspace guest cannot export", actor: "00000000-0000-4000-8000-000000000002", change: `UPDATE workspace_members SET role=5`},
		{name: "item deleted", actor: "00000000-0000-4000-8000-000000000002", change: `UPDATE issues SET deleted_at=NOW()`},
		{name: "completed snapshot missing IDs fails closed", actor: "00000000-0000-4000-8000-000000000002", change: `UPDATE exporters SET rich_filters=rich_filters-'exported_item_ids'`},
		{name: "queued snapshot has no IDs yet", actor: "00000000-0000-4000-8000-000000000002", change: `UPDATE exporters SET status='queued',rich_filters=rich_filters-'exported_item_ids'`, wantCustom: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, err := connection.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if rollbackErr := tx.Rollback(context.WithoutCancel(ctx)); rollbackErr != nil {
					t.Error(rollbackErr)
				}
			}()
			if test.change != "" {
				if _, err = tx.Exec(ctx, test.change); err != nil {
					t.Fatal(err)
				}
			}
			repository, err := NewRepository(tx)
			if err != nil {
				t.Fatal(err)
			}
			data, err := repository.History(ctx, test.actor, "demo", djangoapi.Query{Params: map[string]string{"per_page": "20", "cursor": "20:1:0"}})
			if err != nil {
				t.Fatal(err)
			}
			var response struct {
				Total   int `json:"total_results"`
				Results []struct {
					Token string `json:"token"`
				} `json:"results"`
			}
			if err = json.Unmarshal(data, &response); err != nil {
				t.Fatal(err)
			}
			tokens := map[string]bool{}
			for _, row := range response.Results {
				tokens[row.Token] = true
			}
			if tokens["custom"] != test.wantCustom || !tokens["standard"] || response.Total != len(response.Results) {
				t.Fatalf("history=%s wantCustom=%v and standard always present", data, test.wantCustom)
			}
		})
	}
}
