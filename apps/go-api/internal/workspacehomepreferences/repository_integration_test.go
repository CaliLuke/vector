//go:build integration

package workspacehomepreferences

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

type listedPreference struct {
	Key       string         `json:"key"`
	IsEnabled bool           `json:"is_enabled"`
	Config    map[string]any `json:"config"`
	SortOrder float64        `json:"sort_order"`
}

const (
	homeUserID       = "00000000-0000-4000-8000-000000000001"
	otherHomeUserID  = "00000000-0000-4000-8000-000000000002"
	homeWorkspaceID  = "00000000-0000-4000-8000-000000000010"
	otherWorkspaceID = "00000000-0000-4000-8000-000000000011"
)

func startHomePreferencesDB(t *testing.T) (*pgx.Conn, context.Context) {
	t.Helper()
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
CREATE TABLE workspace_home_preferences (
	id uuid PRIMARY KEY,
	workspace_id uuid NOT NULL REFERENCES workspaces(id),
	user_id uuid NOT NULL,
	key varchar(255) NOT NULL,
	is_enabled boolean NOT NULL,
	config jsonb NOT NULL,
	sort_order double precision NOT NULL,
	created_at timestamptz NOT NULL,
	updated_at timestamptz NOT NULL,
	created_by_id uuid,
	updated_by_id uuid,
	deleted_at timestamptz,
	UNIQUE (workspace_id,user_id,key,deleted_at)
);
CREATE UNIQUE INDEX workspace_user_home_preferences_unique_workspace_user_key_when_deleted_at_null
	ON workspace_home_preferences(workspace_id,user_id,key) WHERE deleted_at IS NULL;
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
		if err := pair.Close(); err != nil {
			t.Errorf("close PostgreSQL testcontainer: %v", err)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })
	return connection, ctx
}

func listPreferences(t *testing.T, repository *Repository, ctx context.Context, user, slug string) []listedPreference {
	t.Helper()
	data, err := repository.List(ctx, uuid.MustParse(user), slug)
	if err != nil {
		t.Fatalf("List(%q) error = %v", slug, err)
	}
	var preferences []listedPreference
	if err := json.Unmarshal(data, &preferences); err != nil {
		t.Fatalf("decode List(%q) result %s: %v", slug, data, err)
	}
	return preferences
}

func TestListCreatesDjangoDefaultsAndRepeatedReadsPreservePreferences(t *testing.T) {
	connection, ctx := startHomePreferencesDB(t)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.MustParse(homeUserID)
	want := []listedPreference{
		{Key: "my_stickies", IsEnabled: true, Config: map[string]any{}, SortOrder: 997},
		{Key: "recents", IsEnabled: true, Config: map[string]any{}, SortOrder: 998},
		{Key: "quick_links", IsEnabled: true, Config: map[string]any{}, SortOrder: 999},
	}
	if got := listPreferences(t, repository, ctx, homeUserID, "demo"); !reflect.DeepEqual(got, want) {
		t.Fatalf("first List() = %#v, want %#v", got, want)
	}
	if _, err := connection.Exec(ctx, `UPDATE workspace_home_preferences
SET is_enabled=false,config='{"pinned":true}',sort_order=7
WHERE workspace_id=$1 AND user_id=$2 AND key='quick_links' AND deleted_at IS NULL`, homeWorkspaceID, userID); err != nil {
		t.Fatal(err)
	}
	got := listPreferences(t, repository, ctx, homeUserID, "demo")
	want = []listedPreference{
		{Key: "quick_links", IsEnabled: false, Config: map[string]any{"pinned": true}, SortOrder: 7},
		{Key: "my_stickies", IsEnabled: true, Config: map[string]any{}, SortOrder: 997},
		{Key: "recents", IsEnabled: true, Config: map[string]any{}, SortOrder: 998},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("repeated List() = %#v, want preserved preferences %#v", got, want)
	}
	var count int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM workspace_home_preferences WHERE workspace_id=$1 AND user_id=$2 AND deleted_at IS NULL`, homeWorkspaceID, userID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("active preference count after repeated read = %d, want 3", count)
	}
}

func TestListFillsMissingDefaultsAndRecreatesSoftDeletedDefault(t *testing.T) {
	connection, ctx := startHomePreferencesDB(t)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	_, err = connection.Exec(ctx, `INSERT INTO workspace_home_preferences (
	id,workspace_id,user_id,key,is_enabled,config,sort_order,created_at,updated_at,deleted_at
) VALUES
	('00000000-0000-4000-8000-000000000061',$1,$2,'quick_links',false,'{"custom":1}',4,'2026-01-01','2026-01-01',NULL),
	('00000000-0000-4000-8000-000000000062',$1,$2,'recents',false,'{"old":true}',3,'2026-01-01','2026-01-01','2026-01-02')`, homeWorkspaceID, homeUserID)
	if err != nil {
		t.Fatal(err)
	}
	want := []listedPreference{
		{Key: "my_stickies", IsEnabled: true, Config: map[string]any{}, SortOrder: 997},
		{Key: "recents", IsEnabled: true, Config: map[string]any{}, SortOrder: 998},
		{Key: "quick_links", IsEnabled: false, Config: map[string]any{"custom": float64(1)}, SortOrder: 4},
	}
	if got := listPreferences(t, repository, ctx, homeUserID, "demo"); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want only missing active defaults added: %#v", got, want)
	}
	var deletedCount int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM workspace_home_preferences WHERE workspace_id=$1 AND user_id=$2 AND key='recents' AND deleted_at IS NOT NULL`, homeWorkspaceID, homeUserID).Scan(&deletedCount); err != nil {
		t.Fatal(err)
	}
	if deletedCount != 1 {
		t.Fatalf("soft-deleted recents row count = %d, want 1 preserved historical row", deletedCount)
	}
}

func TestListKeepsDefaultsIsolatedByUserAndWorkspace(t *testing.T) {
	connection, ctx := startHomePreferencesDB(t)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `INSERT INTO workspace_home_preferences (
	id,workspace_id,user_id,key,is_enabled,config,sort_order,created_at,updated_at
) VALUES ('00000000-0000-4000-8000-000000000063',$1,$2,'quick_links',false,'{"scope":"other"}',1,'2026-01-01','2026-01-01')`, otherWorkspaceID, homeUserID); err != nil {
		t.Fatal(err)
	}
	if got := listPreferences(t, repository, ctx, homeUserID, "demo"); len(got) != 3 || got[2].Key != "quick_links" || got[2].IsEnabled != true {
		t.Fatalf("demo preferences for original user = %#v, want three independent defaults", got)
	}
	if got := listPreferences(t, repository, ctx, otherHomeUserID, "demo"); len(got) != 3 || got[2].Key != "quick_links" || got[2].IsEnabled != true {
		t.Fatalf("demo preferences for second user = %#v, want three independent defaults", got)
	}
	if got := listPreferences(t, repository, ctx, homeUserID, "other"); len(got) != 3 || got[0].Key != "my_stickies" || got[2].Key != "quick_links" || got[2].IsEnabled != false {
		t.Fatalf("other workspace preferences for original user = %#v, want that workspace's existing quick_links", got)
	}
}
