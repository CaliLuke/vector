//go:build integration

package workspacesidebarpreferences

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const (
	sidebarUserID       = "00000000-0000-4000-8000-000000000001"
	otherSidebarUserID  = "00000000-0000-4000-8000-000000000002"
	sidebarWorkspaceID  = "00000000-0000-4000-8000-000000000010"
	otherSidebarSpaceID = "00000000-0000-4000-8000-000000000011"
)

type preferenceValue struct {
	IsPinned  bool    `json:"is_pinned"`
	SortOrder float64 `json:"sort_order"`
}

func startSidebarPreferencesDB(t *testing.T) (*pgx.Conn, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (
	id uuid PRIMARY KEY,
	slug text NOT NULL,
	deleted_at timestamptz
);
CREATE TABLE workspace_user_preferences (
	id uuid PRIMARY KEY,
	created_at timestamptz NOT NULL,
	updated_at timestamptz NOT NULL,
	workspace_id uuid NOT NULL REFERENCES workspaces(id),
	user_id uuid NOT NULL,
	key text NOT NULL,
	is_pinned boolean NOT NULL,
	sort_order double precision NOT NULL,
	deleted_at timestamptz
);
CREATE UNIQUE INDEX workspace_user_preferences_unique_active
	ON workspace_user_preferences(workspace_id,user_id,key) WHERE deleted_at IS NULL;
INSERT INTO workspaces (id,slug) VALUES
	('00000000-0000-4000-8000-000000000010','demo'),
	('00000000-0000-4000-8000-000000000011','other');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeErr := pair.Close()
		if closeErr != nil {
			t.Errorf("close PostgreSQL testcontainer: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })
	return connection, ctx
}

func readSidebarPreferences(t *testing.T, repository *Repository, ctx context.Context, userID, slug string) map[string]preferenceValue {
	t.Helper()
	body, err := repository.List(ctx, uuid.MustParse(userID), slug)
	if err != nil {
		t.Fatalf("List(%q) error = %v", slug, err)
	}
	var preferences map[string]preferenceValue
	if err := json.Unmarshal(body, &preferences); err != nil {
		t.Fatalf("decode List(%q) result %s: %v", slug, body, err)
	}
	return preferences
}

func TestListCreatesSevenDefaultsOnFirstRead(t *testing.T) {
	connection, ctx := startSidebarPreferencesDB(t)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}

	got := readSidebarPreferences(t, repository, ctx, sidebarUserID, "demo")
	want := map[string]preferenceValue{
		"views":         {IsPinned: false, SortOrder: 65535},
		"active_cycles": {IsPinned: false, SortOrder: 75535},
		"analytics":     {IsPinned: false, SortOrder: 85535},
		"drafts":        {IsPinned: true, SortOrder: 95535},
		"your_work":     {IsPinned: true, SortOrder: 105535},
		"archives":      {IsPinned: false, SortOrder: 115535},
		"stickies":      {IsPinned: true, SortOrder: 125535},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("first List() = %#v, want Django defaults %#v", got, want)
	}
	if repeated := readSidebarPreferences(t, repository, ctx, sidebarUserID, "demo"); !reflect.DeepEqual(repeated, want) {
		t.Fatalf("repeated List() = %#v, want unchanged Django defaults %#v", repeated, want)
	}
}

func TestPatchBeforeListCreatesKnownDefaultAndPreservesOmittedField(t *testing.T) {
	connection, ctx := startSidebarPreferencesDB(t)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	pinned := true
	if _, err := repository.Patch(ctx, uuid.MustParse(sidebarUserID), "demo", []Patch{{Key: "views", IsPinned: &pinned}}); err != nil {
		t.Fatalf("Patch() before first List() error = %v", err)
	}

	preferences := readSidebarPreferences(t, repository, ctx, sidebarUserID, "demo")
	if len(preferences) != 7 || !preferences["views"].IsPinned {
		t.Fatalf("first List() after Patch() = %#v, want seven keys and pinned views", preferences)
	}
	sortOrder := 12.0
	if _, err := repository.Patch(ctx, uuid.MustParse(sidebarUserID), "demo", []Patch{{Key: "views", SortOrder: &sortOrder}, {Key: "unknown"}}); err != nil {
		t.Fatalf("partial Patch() error = %v", err)
	}
	preferences = readSidebarPreferences(t, repository, ctx, sidebarUserID, "demo")
	if !preferences["views"].IsPinned || preferences["views"].SortOrder != sortOrder {
		t.Fatalf("partial update of views = %#v, want pinned=true and sort_order=%v", preferences["views"], sortOrder)
	}
	if _, exists := preferences["unknown"]; exists {
		t.Fatalf("unknown preference key was created: %#v", preferences["unknown"])
	}
}

func TestPatchIsScopedToRequestedAccountAndWorkspace(t *testing.T) {
	connection, ctx := startSidebarPreferencesDB(t)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	_, err = connection.Exec(ctx, `INSERT INTO workspace_user_preferences (
	id,created_at,updated_at,workspace_id,user_id,key,is_pinned,sort_order
) VALUES
	('00000000-0000-4000-8000-000000000061',NOW(),NOW(),$1,$2,'views',false,3),
	('00000000-0000-4000-8000-000000000062',NOW(),NOW(),$3,$4,'views',false,4)`, sidebarWorkspaceID, otherSidebarUserID, otherSidebarSpaceID, sidebarUserID)
	if err != nil {
		t.Fatal(err)
	}
	pinned := true
	if _, err := repository.Patch(ctx, uuid.MustParse(sidebarUserID), "demo", []Patch{{Key: "views", IsPinned: &pinned}}); err != nil {
		t.Fatal(err)
	}
	var otherUserPreference, otherWorkspacePreference bool
	if err := connection.QueryRow(ctx, `SELECT is_pinned FROM workspace_user_preferences WHERE workspace_id=$1 AND user_id=$2 AND key='views'`, sidebarWorkspaceID, otherSidebarUserID).Scan(&otherUserPreference); err != nil {
		t.Fatal(err)
	}
	if err := connection.QueryRow(ctx, `SELECT is_pinned FROM workspace_user_preferences WHERE workspace_id=$1 AND user_id=$2 AND key='views'`, otherSidebarSpaceID, sidebarUserID).Scan(&otherWorkspacePreference); err != nil {
		t.Fatal(err)
	}
	if otherUserPreference || otherWorkspacePreference {
		t.Fatalf("patch crossed account/workspace scope: other user value=%v, other workspace value=%v", otherUserPreference, otherWorkspacePreference)
	}
}
