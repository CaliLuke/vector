//go:build integration

package instancemanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorkspacesUsesDjangoCursorAndEscapedSearchAgainstPostgres(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY,email text,first_name text,last_name text,is_bot boolean NOT NULL DEFAULT false);
CREATE TABLE workspaces (id uuid PRIMARY KEY,created_at timestamptz NOT NULL,updated_at timestamptz NOT NULL,deleted_at timestamptz,name text NOT NULL,logo text,logo_asset_id uuid,owner_id uuid NOT NULL,slug text NOT NULL,organization_size text,timezone text,background_color text,created_by_id uuid,updated_by_id uuid);
CREATE TABLE file_assets (id uuid PRIMARY KEY,entity_type text,workspace_id uuid,project_id uuid,issue_id uuid);
CREATE TABLE projects (id uuid PRIMARY KEY,workspace_id uuid NOT NULL,deleted_at timestamptz);
CREATE TABLE workspace_members (id uuid PRIMARY KEY,workspace_id uuid NOT NULL,member_id uuid NOT NULL,deleted_at timestamptz,is_active boolean NOT NULL DEFAULT true);
INSERT INTO users(id,email,first_name,last_name,is_bot) VALUES('00000000-0000-4000-8000-000000000001','admin@example.test','Admin','User',false);
`)
		if err != nil {
			return err
		}
		for index := 0; index < 12; index++ {
			_, err = pool.Exec(ctx, `INSERT INTO workspaces(id,created_at,updated_at,name,logo,owner_id,slug,timezone,background_color) VALUES($1,$2,$2,$3,'','00000000-0000-4000-8000-000000000001',$4,'UTC','#3f76ff')`,
				fmt.Sprintf("00000000-0000-4000-8000-%012d", index+10), time.Date(2026, 10, 21, 0, index, 0, 0, time.UTC), fmt.Sprintf("Managedworkspace %02d", index), fmt.Sprintf("managed-%02d", index))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, pair)

	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repository, err := NewRepository(pool, timestampTestCipher{})
	if err != nil {
		t.Fatal(err)
	}

	page, err := repository.Workspaces(ctx, "", "10", "5:1:0")
	if err != nil {
		t.Fatalf("Workspaces with mismatched cursor page size: %v", err)
	}
	var pageResult struct {
		Count      int              `json:"count"`
		TotalCount int              `json:"total_count"`
		NextCursor string           `json:"next_cursor"`
		Results    []map[string]any `json:"results"`
	}
	if unmarshalErr := json.Unmarshal(page, &pageResult); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if pageResult.TotalCount != 12 || pageResult.Count != 2 || pageResult.NextCursor != "10:2:0" || len(pageResult.Results) != 2 {
		t.Fatalf("cursor page projection = %#v, want 2 rows from page two of 12 with next cursor 10:2:0", pageResult)
	}

	literalPercent, err := repository.Workspaces(ctx, "%", "10", "")
	if err != nil {
		t.Fatalf("Workspaces with literal percent search: %v", err)
	}
	var filtered struct {
		TotalCount int `json:"total_count"`
		Count      int `json:"count"`
	}
	if unmarshalErr := json.Unmarshal(literalPercent, &filtered); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if filtered.TotalCount != 0 || filtered.Count != 0 {
		t.Fatalf("literal percent search = %#v, want zero rows", filtered)
	}

	_, err = pool.Exec(ctx, `INSERT INTO workspaces(id,created_at,updated_at,name,logo,owner_id,slug,timezone,background_color) VALUES('00000000-0000-4000-8000-000000000099',NOW(),NOW(),'literal%workspace','','00000000-0000-4000-8000-000000000001','literal-percent','UTC','#3f76ff')`)
	if err != nil {
		t.Fatal(err)
	}
	literalMatch, err := repository.Workspaces(ctx, "literal%", "10", "")
	if err != nil {
		t.Fatalf("Workspaces with escaped percent search: %v", err)
	}
	if err := json.Unmarshal(literalMatch, &filtered); err != nil {
		t.Fatal(err)
	}
	if filtered.TotalCount != 1 || filtered.Count != 1 {
		t.Fatalf("literal substring search = %#v, want only the row containing percent", filtered)
	}
}

func TestCreateWorkspacePersistsDjangoTextFieldCompanyRoleValues(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY,email text NOT NULL,first_name text,last_name text);
CREATE TABLE workspaces (id uuid PRIMARY KEY,created_at timestamptz NOT NULL,updated_at timestamptz NOT NULL,deleted_at timestamptz,created_by_id uuid,updated_by_id uuid,owner_id uuid NOT NULL,name text NOT NULL,slug text NOT NULL,logo text,logo_asset_id uuid,organization_size text,timezone text,background_color text);
CREATE TABLE file_assets (id uuid PRIMARY KEY,entity_type text,workspace_id uuid,project_id uuid,issue_id uuid);
CREATE TABLE workspace_members (id uuid PRIMARY KEY,workspace_id uuid NOT NULL,member_id uuid NOT NULL,role integer NOT NULL,view_props jsonb,default_props jsonb,issue_props jsonb,is_active boolean NOT NULL,company_role text,getting_started_checklist jsonb,tips jsonb,explored_features jsonb,created_at timestamptz NOT NULL,updated_at timestamptz NOT NULL,created_by_id uuid);
INSERT INTO users(id,email,first_name,last_name) VALUES('00000000-0000-4000-8000-000000000001','admin@example.test','Admin','User');`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, pair)

	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repository := &Repository{db: pool}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	for index, test := range []struct {
		name string
		raw  json.RawMessage
		want *string
	}{
		{name: "number", raw: json.RawMessage(`123`), want: stringPointer("123")},
		{name: "boolean", raw: json.RawMessage(`true`), want: stringPointer("True")},
		{name: "list", raw: json.RawMessage(`[1,"ops"]`), want: stringPointer(`[1, 'ops']`)},
		{name: "object", raw: json.RawMessage(`{"team":"ops"}`), want: stringPointer(`{'team': 'ops'}`)},
		{name: "null", raw: json.RawMessage(`null`), want: nil},
		{name: "omitted", want: stringPointer("")},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields := map[string]json.RawMessage{
				"name": json.RawMessage(fmt.Sprintf(`"Company role %d"`, index)),
				"slug": json.RawMessage(fmt.Sprintf(`"company-role-%d"`, index)),
			}
			if test.raw != nil {
				fields["company_role"] = test.raw
			}
			response, err := repository.CreateWorkspace(ctx, actor, fields, "UTC")
			if err != nil {
				t.Fatalf("CreateWorkspace(): %v", err)
			}
			var projected struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(response, &projected); err != nil {
				t.Fatal(err)
			}
			var got pgtype.Text
			if err := pool.QueryRow(ctx, `SELECT company_role FROM workspace_members WHERE workspace_id=$1`, projected.ID).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got.Valid != (test.want != nil) || got.Valid && got.String != *test.want {
				t.Fatalf("persisted company_role = %#v, want %#v", got, test.want)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
