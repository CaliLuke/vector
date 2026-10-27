//go:build integration

package workspaceexports

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestCreateIssueExportPersistsDjangoUUIDArrayValues(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE users (
  id uuid PRIMARY KEY, first_name text NOT NULL, last_name text NOT NULL,
  display_name text NOT NULL, avatar text NOT NULL, avatar_asset_id uuid,
  is_bot boolean NOT NULL
);
CREATE TABLE exporters (
  id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  type text NOT NULL, workspace_id uuid NOT NULL, project uuid[], provider text NOT NULL,
  status text NOT NULL, reason text NOT NULL, key text NOT NULL, url text, initiated_by_id uuid NOT NULL, token text NOT NULL,
  name text, filters jsonb, rich_filters jsonb, deleted_at timestamptz,
  created_by_id uuid, updated_by_id uuid
);
INSERT INTO workspaces(id,slug) VALUES ('00000000-0000-4000-8000-000000000001','demo');
INSERT INTO users(id,first_name,last_name,display_name,avatar,is_bot)
VALUES ('00000000-0000-4000-8000-000000000002','Test','User','Test User','',FALSE);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close postgres fixture: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	original := []any{json.Number("123"), "123e4567-e89b-12d3-a456-426614174000", nil}
	_, _, _, queued, err := repository.CreateIssueExport(ctx, "00000000-0000-4000-8000-000000000002", "demo", "csv", original, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != len(original) || queued[0] != original[0] || queued[1] != original[1] || queued[2] != nil {
		t.Fatalf("queued values = %#v, want original %#v", queued, original)
	}
	var first, second, createdBy, initiatedBy, exportType, provider, status, reason, key, token string
	var richFilters []byte
	var thirdIsNull, nameIsNull, filtersIsNull, urlIsNull, deletedAtIsNull, updatedByIsNull bool
	if err := connection.QueryRow(ctx, `SELECT project[1]::text,project[2]::text,project[3] IS NULL,created_by_id::text,initiated_by_id::text,rich_filters,type,provider,status,reason,key,token,name IS NULL,filters IS NULL,url IS NULL,deleted_at IS NULL,updated_by_id IS NULL FROM exporters`).Scan(&first, &second, &thirdIsNull, &createdBy, &initiatedBy, &richFilters, &exportType, &provider, &status, &reason, &key, &token, &nameIsNull, &filtersIsNull, &urlIsNull, &deletedAtIsNull, &updatedByIsNull); err != nil {
		t.Fatal(err)
	}
	if first != "00000000-0000-0000-0000-00000000007b" || second != "123e4567-e89b-12d3-a456-426614174000" || !thirdIsNull {
		t.Fatalf("stored UUID array = [%q,%q,null=%v]", first, second, thirdIsNull)
	}
	if createdBy != "00000000-0000-4000-8000-000000000002" || initiatedBy != createdBy {
		t.Fatalf("creator/initiator = %q/%q, want actor %q", createdBy, initiatedBy, "00000000-0000-4000-8000-000000000002")
	}
	if string(richFilters) != `{}` {
		t.Fatalf("rich_filters = %s, want empty object", richFilters)
	}
	if exportType != "issue_exports" || provider != "csv" || status != "queued" || reason != "" || key != "" || token == "" {
		t.Fatalf("export fields = type:%q provider:%q status:%q reason:%q key:%q token-empty:%v", exportType, provider, status, reason, key, token == "")
	}
	if !nameIsNull || !filtersIsNull || !urlIsNull || !deletedAtIsNull || !updatedByIsNull {
		t.Fatalf("nullable defaults = name:%v filters:%v url:%v deleted_at:%v updated_by:%v, want all NULL", nameIsNull, filtersIsNull, urlIsNull, deletedAtIsNull, updatedByIsNull)
	}
	t.Run("custom properties are explicitly persisted", func(t *testing.T) {
		_, _, customToken, _, err := repository.CreateIssueExport(ctx, "00000000-0000-4000-8000-000000000002", "demo", "json", original, true)
		if err != nil {
			t.Fatal(err)
		}
		var enabled bool
		if err = connection.QueryRow(ctx, `SELECT rich_filters->>'include_custom_properties'='true' FROM exporters WHERE token=$1`, customToken).Scan(&enabled); err != nil {
			t.Fatal(err)
		}
		if !enabled {
			t.Fatal("custom opt-in was not persisted")
		}
	})
}

func TestAuthorizedRejectsSoftDeletedMembership(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE workspace_members (
  id uuid PRIMARY KEY, workspace_id uuid NOT NULL, member_id uuid NOT NULL,
  role smallint NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz
);
INSERT INTO workspaces(id,slug) VALUES ('00000000-0000-4000-8000-000000000001','demo');
INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active,deleted_at) VALUES
  ('00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',20,true,NULL),
  ('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000003',20,true,clock_timestamp());
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close postgres fixture: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		userID string
		want   bool
	}{
		{name: "active member", userID: "00000000-0000-4000-8000-000000000002", want: true},
		{name: "soft-deleted member", userID: "00000000-0000-4000-8000-000000000003", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := repository.Authorized(ctx, test.userID, "demo")
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("Authorized(%s) = %v, want %v", test.userID, got, test.want)
			}
		})
	}
}

func TestCreateIssueExportRejectsSoftDeletedWorkspace(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE users (
  id uuid PRIMARY KEY, first_name text NOT NULL, last_name text NOT NULL,
  display_name text NOT NULL, avatar text NOT NULL, avatar_asset_id uuid,
  is_bot boolean NOT NULL
);
CREATE TABLE exporters (
  id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  type text NOT NULL, workspace_id uuid NOT NULL, project uuid[], provider text NOT NULL,
  status text NOT NULL, reason text NOT NULL, key text NOT NULL, url text, initiated_by_id uuid NOT NULL,
  token text NOT NULL, name text, filters jsonb, rich_filters jsonb, deleted_at timestamptz,
  created_by_id uuid, updated_by_id uuid
);
INSERT INTO workspaces(id,slug,deleted_at)
VALUES ('00000000-0000-4000-8000-000000000001','demo',clock_timestamp());
INSERT INTO users(id,first_name,last_name,display_name,avatar,is_bot)
VALUES ('00000000-0000-4000-8000-000000000002','Test','User','Test User','',FALSE);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close postgres fixture: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := repository.CreateIssueExport(ctx, "00000000-0000-4000-8000-000000000002", "demo", "csv", []any{"00000000-0000-4000-8000-000000000003"}, false); !errors.Is(err, ErrWorkspaceMissing) {
		t.Fatalf("CreateIssueExport error = %v, want ErrWorkspaceMissing", err)
	}
	var count int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM exporters`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("exporter rows = %d, want 0 for soft-deleted workspace", count)
	}
}

func TestCreateIssueExportDefaultsExcludeSoftDeletedProjects(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, archived_at timestamptz, deleted_at timestamptz);
CREATE TABLE project_members (project_id uuid NOT NULL, member_id uuid NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz);
CREATE TABLE users (
  id uuid PRIMARY KEY, first_name text NOT NULL, last_name text NOT NULL,
  display_name text NOT NULL, avatar text NOT NULL, avatar_asset_id uuid,
  is_bot boolean NOT NULL
);
CREATE TABLE exporters (
  id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  type text NOT NULL, workspace_id uuid NOT NULL, project uuid[], provider text NOT NULL,
  status text NOT NULL, reason text NOT NULL, key text NOT NULL, url text, initiated_by_id uuid NOT NULL, token text NOT NULL,
  name text, filters jsonb, rich_filters jsonb, deleted_at timestamptz,
  created_by_id uuid, updated_by_id uuid
);
INSERT INTO workspaces(id,slug) VALUES ('00000000-0000-4000-8000-000000000001','demo');
INSERT INTO users(id,first_name,last_name,display_name,avatar,is_bot)
VALUES ('00000000-0000-4000-8000-000000000002','Test','User','Test User','',FALSE);
INSERT INTO projects(id,workspace_id) VALUES
  ('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000001'),
  ('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001');
UPDATE projects SET deleted_at=clock_timestamp() WHERE id='00000000-0000-4000-8000-000000000004';
INSERT INTO project_members(project_id,member_id,is_active,deleted_at) VALUES
  ('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002',true,NULL),
  ('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000002',true,NULL);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close postgres fixture: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, projects, err := repository.CreateIssueExport(ctx, "00000000-0000-4000-8000-000000000002", "demo", "csv", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0] != "00000000-0000-4000-8000-000000000003" {
		t.Fatalf("default projects = %#v, want only the non-deleted project", projects)
	}
}

func TestHistoryExcludesSoftDeletedExporterRows(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, historyVisibilityFixtureDDL+`
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE users (id uuid PRIMARY KEY, first_name text, last_name text, display_name text, avatar text, avatar_asset_id uuid, is_bot boolean);
CREATE TABLE exporters (
  id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  type text NOT NULL, workspace_id uuid NOT NULL, project uuid[], provider text NOT NULL,
  status text NOT NULL, reason text NOT NULL, key text NOT NULL, url text, initiated_by_id uuid,
  token text NOT NULL, created_by_id uuid, updated_by_id uuid, deleted_at timestamptz, rich_filters jsonb
);
INSERT INTO workspaces(id,slug) VALUES ('00000000-0000-4000-8000-000000000001','demo');
INSERT INTO exporters(id,created_at,updated_at,type,workspace_id,project,provider,status,reason,key,url,initiated_by_id,token,created_by_id,updated_by_id,deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000011',clock_timestamp(),clock_timestamp(),'issue_exports','00000000-0000-4000-8000-000000000001',ARRAY[]::uuid[],'csv','queued','','',NULL,NULL,'token-live',NULL,NULL,NULL),
 ('00000000-0000-4000-8000-000000000012',clock_timestamp(),clock_timestamp(),'issue_exports','00000000-0000-4000-8000-000000000001',ARRAY[]::uuid[],'csv','queued','','',NULL,NULL,'token-deleted',NULL,NULL,clock_timestamp());
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close postgres fixture: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	data, err := repository.History(ctx, "00000000-0000-4000-8000-000000000002", "demo", djangoapi.Query{Params: map[string]string{"per_page": "20", "cursor": "20:1:0", "order_by": "-created_at"}})
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 || response.Results[0]["token"] != "token-live" {
		t.Fatalf("history results = %#v, want only token-live", response.Results)
	}
}
