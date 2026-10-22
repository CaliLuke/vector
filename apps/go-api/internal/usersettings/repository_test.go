package usersettings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"uuid"
)

type settingsRow func(...any) error

func (row settingsRow) Scan(dest ...any) error { return row(dest...) }

type settingsDB struct {
	query string
	row   pgx.Row
}

func (db *settingsDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	db.query = query
	return db.row
}

func settingsJSON(value string) pgx.Row {
	return settingsRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(value)
		return nil
	})
}

func TestRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("expected nil database error")
	}
}

func TestGetMirrorsSerializerWorkspaceSelectionAndLogo(t *testing.T) {
	db := &settingsDB{row: settingsJSON(`{"id":"user","email":"u@example.com","workspace":{}}`)}
	r, _ := NewRepository(db)
	if body, err := r.Get(context.Background(), uuid.New()); err != nil || string(body) != `{"id":"user","email":"u@example.com","workspace":{}}` {
		t.Fatalf("Get() = %s, %v", body, err)
	}
	for _, fragment := range []string{
		"workspace_member_invites",
		"p.last_workspace_id",
		"wm.is_active=true",
		"ORDER BY CASE WHEN w.id=up.last_workspace_id THEN 0 ELSE 1 END,w.created_at ASC",
		"/api/assets/v2/static/",
		"fa.entity_type='WORKSPACE_LOGO'",
		"jsonb_build_object(",
		"'last_workspace_name',sw.name",
		"ELSE '{}'::jsonb END)::json",
	} {
		if !strings.Contains(db.query, fragment) {
			t.Errorf("settings query missing %q", fragment)
		}
	}
	if strings.Contains(db.query, "'last_workspace_name',CASE WHEN") {
		t.Fatal("settings query must not serialize selected-workspace fields as null fallback keys")
	}
}

func TestGetResponseShapeMatchesDjangoFallbackFixture(t *testing.T) {
	db := &settingsDB{row: settingsJSON(`{"id":"<uuid-1>","email":"settings@example.com","workspace":{"fallback_workspace_id":"<uuid-2>","fallback_workspace_slug":"settings-parity","invites":1,"last_workspace_id":null,"last_workspace_slug":null}}`)}
	r, _ := NewRepository(db)
	body, err := r.Get(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Get() returned invalid JSON: %v", err)
	}
	workspace := got["workspace"].(map[string]any)
	for _, key := range []string{"last_workspace_name", "last_workspace_logo"} {
		if _, ok := workspace[key]; ok {
			t.Errorf("fallback workspace unexpectedly contains %q", key)
		}
	}
}

func TestGetResponseShapeMatchesDjangoSelectedFixture(t *testing.T) {
	db := &settingsDB{row: settingsJSON(`{"id":"<uuid-1>","email":"settings@example.com","workspace":{"fallback_workspace_id":"<uuid-2>","fallback_workspace_slug":"settings-parity","invites":1,"last_workspace_id":"<uuid-2>","last_workspace_logo":"","last_workspace_name":"Settings Parity","last_workspace_slug":"settings-parity"}}`)}
	r, _ := NewRepository(db)
	body, err := r.Get(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Get() returned invalid JSON: %v", err)
	}
	workspace := got["workspace"].(map[string]any)
	if workspace["last_workspace_name"] != "Settings Parity" {
		t.Errorf("last_workspace_name = %v, want Settings Parity", workspace["last_workspace_name"])
	}
	if workspace["last_workspace_logo"] != "" {
		t.Errorf("last_workspace_logo = %v, want empty string", workspace["last_workspace_logo"])
	}
}

func TestGetMapsMissingProfileToNotFound(t *testing.T) {
	db := &settingsDB{row: settingsRow(func(...any) error { return pgx.ErrNoRows })}
	r, _ := NewRepository(db)
	if _, err := r.Get(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
}
