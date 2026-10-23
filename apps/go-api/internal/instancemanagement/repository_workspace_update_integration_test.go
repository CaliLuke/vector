//go:build integration

package instancemanagement

import (
	"context"
	"encoding/json"
	"testing"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUpdateWorkspacePersistsChangesAndProjectsOptionalLogo(t *testing.T) {
	ctx := t.Context()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY,email text,first_name text,last_name text,is_bot boolean NOT NULL DEFAULT false);
CREATE TABLE workspaces (id uuid PRIMARY KEY,created_at timestamptz NOT NULL,updated_at timestamptz NOT NULL,deleted_at timestamptz,name text NOT NULL,logo text,logo_asset_id uuid,owner_id uuid NOT NULL,slug text NOT NULL,organization_size text,timezone text,background_color text,created_by_id uuid,updated_by_id uuid);
CREATE TABLE file_assets (id uuid PRIMARY KEY,entity_type text);
CREATE TABLE workspace_members (workspace_id uuid NOT NULL,member_id uuid NOT NULL,role integer NOT NULL,is_active boolean NOT NULL,deleted_at timestamptz);
INSERT INTO users VALUES
('00000000-0000-4000-8000-000000000001','owner@example.test','Workspace','Owner',false),
('00000000-0000-4000-8000-000000000004','bot@example.test','Bot','Member',true),
('00000000-0000-4000-8000-000000000005','inactive@example.test','Inactive','Member',false),
('00000000-0000-4000-8000-000000000006','deleted@example.test','Deleted','Member',false);
INSERT INTO file_assets VALUES ('00000000-0000-4000-8000-000000000002','WORKSPACE_LOGO'),('00000000-0000-4000-8000-000000000003','ISSUE_ATTACHMENT');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repository := &Repository{db: pool}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	for _, test := range []struct {
		slug    string
		logo    string
		assetID *string
		wantURL *string
	}{
		{slug: "no-logo"},
		{slug: "legacy-logo", logo: "https://example.test/logo.png", wantURL: stringPointer("https://example.test/logo.png")},
		{slug: "static-logo", assetID: stringPointer("00000000-0000-4000-8000-000000000002"), wantURL: stringPointer("/api/assets/v2/static/00000000-0000-4000-8000-000000000002/")},
		{slug: "non-static-logo", assetID: stringPointer("00000000-0000-4000-8000-000000000003")},
	} {
		t.Run(test.slug, func(t *testing.T) {
			var id uuid.UUID
			err := pool.QueryRow(ctx, `INSERT INTO workspaces(id,created_at,updated_at,name,slug,logo,logo_asset_id,owner_id,timezone) VALUES(gen_random_uuid(),NOW(),NOW(),'Original',@slug,@logo,@asset,@actor,'UTC') RETURNING id`, pgx.StrictNamedArgs{"slug": test.slug, "logo": test.logo, "asset": test.assetID, "actor": actor}).Scan(&id)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO workspace_members VALUES(@workspace,@actor,20,true,NULL),(@workspace,'00000000-0000-4000-8000-000000000004',15,true,NULL),(@workspace,'00000000-0000-4000-8000-000000000005',15,false,NULL),(@workspace,'00000000-0000-4000-8000-000000000006',15,true,NOW())`, pgx.StrictNamedArgs{"workspace": id, "actor": actor}); err != nil {
				t.Fatal(err)
			}
			body, err := repository.UpdateWorkspace(ctx, actor, test.slug, map[string]json.RawMessage{"name": json.RawMessage(`"Updated workspace"`)})
			if err != nil {
				t.Fatalf("update workspace: %v", err)
			}
			var got struct {
				Name         string  `json:"name"`
				LogoURL      *string `json:"logo_url"`
				Owner        string  `json:"owner"`
				TotalMembers int     `json:"total_members"`
			}
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatal(err)
			}
			if got.Name != "Updated workspace" || got.Owner != actor.String() || got.TotalMembers != 1 {
				t.Fatalf("updated response = %s", body)
			}
			if (got.LogoURL == nil) != (test.wantURL == nil) || got.LogoURL != nil && *got.LogoURL != *test.wantURL {
				t.Fatalf("logo projection = %s", body)
			}
			var name, timezone string
			var updatedBy uuid.UUID
			if err := pool.QueryRow(ctx, `SELECT name,timezone,updated_by_id FROM workspaces WHERE id=$1`, id).Scan(&name, &timezone, &updatedBy); err != nil {
				t.Fatal(err)
			}
			if name != "Updated workspace" || timezone != "UTC" || updatedBy != actor {
				t.Fatalf("persisted name=%q timezone=%q updated_by=%s", name, timezone, updatedBy)
			}
		})
	}
}
