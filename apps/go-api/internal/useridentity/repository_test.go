package useridentity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"uuid"
)

type identityRow func(...any) error

func (row identityRow) Scan(destinations ...any) error { return row(destinations...) }

type identityDB struct {
	query string
	args  []any
	row   pgx.Row
}

func (database *identityDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.query, database.args = query, args
	return database.row
}

func identityJSONRow(value string) pgx.Row {
	return identityJSONRowWithOmissions(value, false, false)
}

func identityJSONRowWithOmissions(value string, avatarURL bool, coverImageURL bool) pgx.Row {
	return identityRow(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		if len(destinations) > 1 {
			*destinations[1].(*bool) = avatarURL
			*destinations[2].(*bool) = coverImageURL
		}
		return nil
	})
}

func TestIdentityRepositoryOmitsUnavailableScopedAssetURLs(t *testing.T) {
	database := &identityDB{row: identityJSONRowWithOmissions(
		`{"id":"user","avatar_url":null,"nested":{"cover_image_url":"keep"},"cover_image_url":null,"username":"user"}`,
		true,
		true,
	)}
	repository, _ := NewRepository(database)
	body, err := repository.SessionUser(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000010"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"user","nested":{"cover_image_url":"keep"},"username":"user"}`
	if string(body) != want {
		t.Fatalf("SessionUser() = %s, want %s", body, want)
	}
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestIdentityRepositoryReadsSessionAndProfile(t *testing.T) {
	database := &identityDB{row: identityJSONRow(`{"id":"user"}`)}
	repository, _ := NewRepository(database)
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	if body, err := repository.SessionUser(context.Background(), userID); err != nil || string(body) != `{"id":"user"}` {
		t.Fatalf("SessionUser() = %s, %v", body, err)
	}
	for _, fragment := range []string{"avatar_url", "cover_image_url", "avatar_asset.entity_type", "cover_asset.entity_type", "ISSUE_ATTACHMENT", "DRAFT_ISSUE_DESCRIPTION", "u.is_active=true", "set_config('TimeZone'", "user_timezone", "to_char(u.date_joined,'US')", "to_char(u.last_login_time,'US')"} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("session query missing %q", fragment)
		}
	}
	database.row = identityJSONRow(`{"id":"profile"}`)
	if _, err := repository.Profile(context.Background(), userID); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"onboarding_step", "mobile_onboarding_step", "notification_view_mode", "product_tour", "'user',p.user_id"} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("profile query missing %q", fragment)
		}
	}
}

func TestIdentityRepositoryUpdatesOnlyPresentProfileFields(t *testing.T) {
	database := &identityDB{row: identityJSONRow(`{"theme":{"mode":"dark"}}`)}
	repository, _ := NewRepository(database)
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	patch := json.RawMessage(`{"theme":{"mode":"dark"},"billing_address":null}`)
	if _, err := repository.UpdateProfile(context.Background(), userID, patch); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"input.body ? 'theme'",
		"input.body ? 'billing_address'",
		"input.body->'billing_address'='null'::jsonb",
		"input.body ? 'last_workspace_id'",
		"updated_at=NOW()",
	} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("update query missing %q", fragment)
		}
	}
	if len(database.args) != 2 || string(database.args[1].(json.RawMessage)) != string(patch) {
		t.Fatalf("update args = %#v", database.args)
	}
}

func TestIdentityRepositoryMapsMissingRows(t *testing.T) {
	database := &identityDB{row: identityRow(func(...any) error { return pgx.ErrNoRows })}
	repository, _ := NewRepository(database)
	if _, err := repository.Profile(context.Background(), uuid.Nil()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Profile() error = %v, want ErrNotFound", err)
	}
}
