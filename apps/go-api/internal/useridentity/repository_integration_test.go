//go:build integration

package useridentity

import (
	"context"
	"encoding/json"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"uuid"
)

func TestSessionUserMatchesDjangoUserTimezoneSerialization(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (
  id uuid PRIMARY KEY, slug text NOT NULL
);
CREATE TABLE file_assets (
  id uuid PRIMARY KEY, entity_type text, workspace_id uuid, project_id uuid, issue_id uuid
);
CREATE TABLE users (
  id uuid PRIMARY KEY, avatar text NOT NULL, avatar_asset_id uuid,
  cover_image text NOT NULL, cover_image_asset_id uuid, date_joined timestamptz NOT NULL,
  display_name text NOT NULL, email text, first_name text NOT NULL,
  last_name text NOT NULL, is_active boolean NOT NULL, is_bot boolean NOT NULL,
  is_email_verified boolean NOT NULL, user_timezone text NOT NULL, username text NOT NULL,
  is_password_autoset boolean NOT NULL, last_login_medium text, last_login_time timestamptz
);
INSERT INTO workspaces VALUES ('00000000-0000-4000-8000-000000000b10', 'demo');
INSERT INTO file_assets VALUES (
  '00000000-0000-4000-8000-000000000b11', 'USER_AVATAR',
  '00000000-0000-4000-8000-000000000b10',
  '00000000-0000-4000-8000-000000000b12',
  '00000000-0000-4000-8000-000000000b13'
);
INSERT INTO users VALUES (
  '00000000-0000-4000-8000-000000000b01', 'https://example.com/avatar.png', '00000000-0000-4000-8000-000000000b11',
  'https://example.com/cover.png', '00000000-0000-4000-8000-000000000b11', '2026-07-02 03:04:05.123456+00',
  'Current User', 'user-me@example.com', 'Current', 'User', true, false, true,
  'America/Los_Angeles', 'user-me@example.com', true, 'magic-code',
  '2026-07-02 03:04:05.123456+00'
), (
  '00000000-0000-4000-8000-000000000b02', '', NULL,
  '', NULL, '2026-07-02 03:04:05.123000+00',
  'UTC User', NULL, 'UTC', 'User', true, false, true,
  'UTC', 'utc-user@example.com', false, NULL,
  '2026-07-02 03:04:05.123000+00'
);`)
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
	repository, _ := NewRepository(connection)
	body, err := repository.SessionUser(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000b01"))
	if err != nil {
		t.Fatal(err)
	}
	var user struct {
		DateJoined    string  `json:"date_joined"`
		LastLoginTime string  `json:"last_login_time"`
		AvatarURL     *string `json:"avatar_url"`
		CoverImageURL *string `json:"cover_image_url"`
		Email         *string `json:"email"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatal(err)
	}
	const expected = "2026-07-01T20:04:05.123456-07:00"
	if user.DateJoined != expected || user.LastLoginTime != expected {
		t.Fatalf("timezone timestamps = %#v, want %q", user, expected)
	}
	const staticURL = "/api/assets/v2/static/00000000-0000-4000-8000-000000000b11/"
	if user.AvatarURL == nil || *user.AvatarURL != staticURL || user.CoverImageURL == nil || *user.CoverImageURL != staticURL {
		t.Fatalf("static asset URLs = %#v", user)
	}
	assetCases := []struct {
		name        string
		entityType  string
		workspaceID any
		projectID   any
		issueID     any
		wantURL     *string
		omitURL     bool
	}{
		{name: "issue attachment", entityType: "ISSUE_ATTACHMENT", workspaceID: "00000000-0000-4000-8000-000000000b10", projectID: "00000000-0000-4000-8000-000000000b12", issueID: "00000000-0000-4000-8000-000000000b13", wantURL: stringPointer("/api/assets/v2/workspaces/demo/projects/00000000-0000-4000-8000-000000000b12/issues/00000000-0000-4000-8000-000000000b13/attachments/00000000-0000-4000-8000-000000000b11/")},
		{name: "description", entityType: "ISSUE_DESCRIPTION", workspaceID: "00000000-0000-4000-8000-000000000b10", projectID: "00000000-0000-4000-8000-000000000b12", issueID: "00000000-0000-4000-8000-000000000b13", wantURL: stringPointer("/api/assets/v2/workspaces/demo/projects/00000000-0000-4000-8000-000000000b12/00000000-0000-4000-8000-000000000b11/")},
		{name: "nullable issue references", entityType: "ISSUE_ATTACHMENT", workspaceID: "00000000-0000-4000-8000-000000000b10", wantURL: stringPointer("/api/assets/v2/workspaces/demo/projects/None/issues/None/attachments/00000000-0000-4000-8000-000000000b11/")},
		{name: "nullable description project", entityType: "ISSUE_DESCRIPTION", workspaceID: "00000000-0000-4000-8000-000000000b10", wantURL: stringPointer("/api/assets/v2/workspaces/demo/projects/None/00000000-0000-4000-8000-000000000b11/")},
		{name: "unsupported", entityType: "DRAFT_ISSUE_ATTACHMENT"},
		{name: "missing scoped workspace", entityType: "ISSUE_ATTACHMENT", projectID: "00000000-0000-4000-8000-000000000b12", issueID: "00000000-0000-4000-8000-000000000b13", omitURL: true},
	}
	for _, testCase := range assetCases {
		if _, err := connection.Exec(ctx, "UPDATE file_assets SET entity_type=$1, workspace_id=$2, project_id=$3, issue_id=$4", testCase.entityType, testCase.workspaceID, testCase.projectID, testCase.issueID); err != nil {
			t.Fatal(err)
		}
		assetBody, err := repository.SessionUser(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000b01"))
		if err != nil {
			t.Fatalf("%s SessionUser(): %v", testCase.name, err)
		}
		if err := json.Unmarshal(assetBody, &user); err != nil {
			t.Fatal(err)
		}
		if testCase.omitURL {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(assetBody, &fields); err != nil {
				t.Fatal(err)
			}
			if _, avatarExists := fields["avatar_url"]; avatarExists {
				t.Fatalf("%s unexpectedly returned avatar_url: %s", testCase.name, assetBody)
			}
			if _, coverExists := fields["cover_image_url"]; coverExists {
				t.Fatalf("%s unexpectedly returned cover_image_url: %s", testCase.name, assetBody)
			}
		}
		if !equalOptionalString(user.AvatarURL, testCase.wantURL) || !equalOptionalString(user.CoverImageURL, testCase.wantURL) {
			t.Fatalf("%s asset URLs = %#v, want %v", testCase.name, user, testCase.wantURL)
		}
	}
	utcBody, err := repository.SessionUser(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000b02"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(utcBody, &user); err != nil {
		t.Fatal(err)
	}
	const utcExpected = "2026-07-02T03:04:05.123000Z"
	if user.DateJoined != utcExpected || user.LastLoginTime != utcExpected {
		t.Fatalf("UTC timestamps = %#v, want %q", user, utcExpected)
	}
	if user.Email != nil {
		t.Fatalf("nullable email = %q, want null", *user.Email)
	}
}

func stringPointer(value string) *string { return &value }

func equalOptionalString(left, right *string) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}
