//go:build integration

package projectmembershipreads

import (
	"context"
	"encoding/json"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"uuid"
)

func TestWorkspaceMembersMirrorsFileAssetURLBranches(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (
  id uuid PRIMARY KEY,
  slug text NOT NULL,
  deleted_at timestamptz
);
CREATE TABLE file_assets (
  id uuid PRIMARY KEY,
  entity_type text,
  workspace_id uuid,
  project_id uuid,
  issue_id uuid
);
CREATE TABLE users (
  id uuid PRIMARY KEY,
  user_timezone text,
  first_name text NOT NULL DEFAULT '',
  last_name text NOT NULL DEFAULT '',
  email text,
  avatar text NOT NULL DEFAULT '',
  avatar_asset_id uuid,
  last_login_medium text,
  is_bot boolean NOT NULL DEFAULT false,
  display_name text NOT NULL DEFAULT ''
);
CREATE TABLE workspace_members (
  id uuid PRIMARY KEY,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  created_by_id uuid,
  updated_by_id uuid,
  deleted_at timestamptz,
  workspace_id uuid NOT NULL,
  member_id uuid NOT NULL,
  role integer NOT NULL,
  is_active boolean NOT NULL,
  company_role text,
  view_props jsonb NOT NULL DEFAULT '{}',
  default_props jsonb NOT NULL DEFAULT '{}',
  issue_props jsonb NOT NULL DEFAULT '{}',
  explored_features jsonb NOT NULL DEFAULT '{}',
  getting_started_checklist jsonb NOT NULL DEFAULT '{}',
  tips jsonb NOT NULL DEFAULT '{}'
);
INSERT INTO workspaces (id, slug) VALUES
  ('00000000-0000-4000-8000-000000000001', 'demo');
INSERT INTO file_assets (id, entity_type, workspace_id, project_id, issue_id) VALUES
  ('00000000-0000-4000-8000-000000000002', 'ISSUE_ATTACHMENT',
   '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000003',
   '00000000-0000-4000-8000-000000000004');
INSERT INTO users (id, user_timezone, display_name, avatar_asset_id) VALUES
  ('00000000-0000-4000-8000-000000000005', 'UTC', 'Requester', NULL),
  ('00000000-0000-4000-8000-000000000006', 'UTC', 'Asset Target',
   '00000000-0000-4000-8000-000000000002');
INSERT INTO workspace_members (
  id, created_at, updated_at, workspace_id, member_id, role, is_active
) VALUES
  ('00000000-0000-4000-8000-000000000007', NOW(), NOW(),
   '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000005', 20, true),
  ('00000000-0000-4000-8000-000000000008', NOW(), NOW(),
   '00000000-0000-4000-8000-000000000001',
   '00000000-0000-4000-8000-000000000006', 15, true);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()

	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}

	assetID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	requesterID := uuid.MustParse("00000000-0000-4000-8000-000000000005")
	tests := []struct {
		entityType string
		wantURL    any
	}{
		{
			"ISSUE_ATTACHMENT",
			"/api/assets/v2/workspaces/demo/projects/00000000-0000-4000-8000-000000000003/issues/00000000-0000-4000-8000-000000000004/attachments/00000000-0000-4000-8000-000000000002/",
		},
		{
			"ISSUE_DESCRIPTION",
			"/api/assets/v2/workspaces/demo/projects/00000000-0000-4000-8000-000000000003/00000000-0000-4000-8000-000000000002/",
		},
		{"USER_AVATAR", "/api/assets/v2/static/00000000-0000-4000-8000-000000000002/"},
		{"DRAFT_ISSUE_ATTACHMENT", nil},
	}
	for _, test := range tests {
		t.Run(test.entityType, func(t *testing.T) {
			if _, err := connection.Exec(ctx, "UPDATE file_assets SET entity_type = $1 WHERE id = $2", test.entityType, assetID); err != nil {
				t.Fatal(err)
			}
			data, err := repository.WorkspaceMembers(ctx, requesterID, "demo", nil)
			if err != nil {
				t.Fatal(err)
			}
			var memberships []struct {
				Member struct {
					DisplayName string `json:"display_name"`
					AvatarURL   any    `json:"avatar_url"`
				} `json:"member"`
			}
			if err := json.Unmarshal(data, &memberships); err != nil {
				t.Fatal(err)
			}
			for _, membership := range memberships {
				if membership.Member.DisplayName == "Asset Target" {
					if membership.Member.AvatarURL != test.wantURL {
						t.Fatalf("avatar_url=%#v want %#v; response=%s", membership.Member.AvatarURL, test.wantURL, data)
					}
					return
				}
			}
			t.Fatalf("asset target missing from response: %s", data)
		})
	}
}
