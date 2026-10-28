//go:build integration

package accountauthentication

import (
	"context"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryCreatesDjangoAccountDefaultsAndUpdatesLoginMetadata(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE users (
  id uuid PRIMARY KEY, username varchar(128) UNIQUE NOT NULL, email varchar(255) UNIQUE NOT NULL,
  display_name varchar(255) NOT NULL, first_name varchar(255) NOT NULL, last_name varchar(255) NOT NULL,
  password varchar(128) NOT NULL, last_login timestamptz, avatar text NOT NULL, avatar_asset_id uuid,
  cover_image varchar(800), cover_image_asset_id uuid, date_joined timestamptz NOT NULL,
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, last_location varchar(255) NOT NULL,
  created_location varchar(255) NOT NULL, is_superuser boolean NOT NULL, is_managed boolean NOT NULL,
  is_password_expired boolean NOT NULL, is_active boolean NOT NULL, is_staff boolean NOT NULL,
  is_email_verified boolean NOT NULL, is_password_autoset boolean NOT NULL,
  is_password_reset_required boolean NOT NULL, token varchar(64) NOT NULL, last_active timestamptz,
  last_login_time timestamptz, last_logout_time timestamptz, last_login_ip varchar(255) NOT NULL,
  last_logout_ip varchar(255) NOT NULL, last_login_medium varchar(20) NOT NULL,
  last_login_uagent text NOT NULL, token_updated_at timestamptz, is_bot boolean NOT NULL,
  bot_type varchar(30), user_timezone varchar(255) NOT NULL, is_email_valid boolean NOT NULL,
  masked_at timestamptz
);
CREATE TABLE profiles (
  id uuid PRIMARY KEY, user_id uuid UNIQUE NOT NULL REFERENCES users(id), theme jsonb NOT NULL,
  is_app_rail_docked boolean NOT NULL, is_tour_completed boolean NOT NULL, onboarding_step jsonb NOT NULL,
  use_case text, role varchar(300), is_onboarded boolean NOT NULL, last_workspace_id uuid,
  billing_address_country varchar(255) NOT NULL, billing_address jsonb, has_billing_address boolean NOT NULL,
  company_name varchar(255) NOT NULL, notification_view_mode varchar(255) NOT NULL,
  is_smooth_cursor_enabled boolean NOT NULL, is_mobile_onboarded boolean NOT NULL,
  mobile_onboarding_step jsonb NOT NULL, mobile_timezone_auto_set boolean NOT NULL,
  language varchar(255) NOT NULL, start_of_the_week smallint NOT NULL, goals jsonb NOT NULL,
  background_color varchar(255) NOT NULL, is_navigation_tour_completed boolean NOT NULL,
  has_marketing_email_consent boolean NOT NULL, is_subscribed_to_changelog boolean NOT NULL,
  product_tour jsonb NOT NULL, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL
);
CREATE TABLE user_notification_preferences (
  id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id), workspace_id uuid, project_id uuid,
  property_change boolean NOT NULL, state_change boolean NOT NULL, comment boolean NOT NULL,
  mention boolean NOT NULL, issue_completed boolean NOT NULL, created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);
CREATE TABLE workspaces (
  id uuid PRIMARY KEY, slug text NOT NULL, created_at timestamptz NOT NULL, deleted_at timestamptz
);
CREATE TABLE workspace_members (
  workspace_id uuid NOT NULL REFERENCES workspaces(id), member_id uuid NOT NULL REFERENCES users(id),
  is_active boolean NOT NULL, deleted_at timestamptz
);
CREATE TABLE workspace_member_invites (
  id uuid PRIMARY KEY, email varchar(255) NOT NULL, accepted boolean NOT NULL, deleted_at timestamptz
);
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
	repository, err := NewRepository(connection, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	created, err := repository.CreateUser(ctx, "new@example.test", "generated-user-name", "pbkdf2_sha256$hash", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Username != "generated-user-name" || created.Email != "new@example.test" || !created.Active {
		t.Fatalf("created user defaults = %#v", created)
	}
	var profileCount, preferencesCount int
	var displayName, country, language string
	var preferenceFlags int
	err = connection.QueryRow(ctx, `
SELECT (SELECT count(*) FROM profiles WHERE user_id=$1),
       (SELECT count(*) FROM user_notification_preferences WHERE user_id=$1),
       (SELECT display_name FROM users WHERE id=$1),
       (SELECT billing_address_country FROM profiles WHERE user_id=$1),
       (SELECT language FROM profiles WHERE user_id=$1),
       (SELECT (property_change::int + state_change::int + comment::int + mention::int + issue_completed::int)
          FROM user_notification_preferences WHERE user_id=$1)`, created.ID).Scan(
		&profileCount, &preferencesCount, &displayName, &country, &language, &preferenceFlags,
	)
	if err != nil {
		t.Fatal(err)
	}
	if profileCount != 1 || preferencesCount != 1 || displayName != "new" || country != "INDIA" || language != "en" || preferenceFlags != 5 {
		t.Fatalf("created sidecars profile=%d prefs=%d display=%q country=%q language=%q flags=%d", profileCount, preferencesCount, displayName, country, language, preferenceFlags)
	}

	earliestWorkspaceID := "00000000-0000-4000-8000-000000000011"
	lastWorkspaceID := "00000000-0000-4000-8000-000000000012"
	inactiveWorkspaceID := "00000000-0000-4000-8000-000000000013"
	_, err = connection.Exec(ctx, `INSERT INTO workspaces(id,slug,created_at,deleted_at) VALUES
($1,'earliest-active',TIMESTAMPTZ '2026-01-01 00:00:00+00',NULL),
($2,'last-active',TIMESTAMPTZ '2026-02-01 00:00:00+00',NULL),
($3,'inactive-membership',TIMESTAMPTZ '2025-01-01 00:00:00+00',NULL)`, earliestWorkspaceID, lastWorkspaceID, inactiveWorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = connection.Exec(ctx, `INSERT INTO workspace_members(workspace_id,member_id,is_active,deleted_at) VALUES
($1,$4,TRUE,NULL),($2,$4,TRUE,NULL),($3,$4,FALSE,NULL)`, earliestWorkspaceID, lastWorkspaceID, inactiveWorkspaceID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = connection.Exec(ctx, `UPDATE profiles SET is_onboarded=TRUE,last_workspace_id=$2 WHERE user_id=$1`, created.ID, lastWorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = connection.Exec(ctx, `INSERT INTO workspace_member_invites(id,email,accepted,deleted_at) VALUES
('00000000-0000-4000-8000-000000000021','new@example.test',TRUE,NULL),
('00000000-0000-4000-8000-000000000022','new@example.test',FALSE,NULL),
('00000000-0000-4000-8000-000000000023','new@example.test',FALSE,NOW()),
('00000000-0000-4000-8000-000000000024','deleted-only@example.test',FALSE,NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	if invited, hasWorkspaceInviteErr := repository.HasWorkspaceInvite(ctx, created.Email); hasWorkspaceInviteErr != nil || !invited {
		t.Fatalf("active invite existence=%v err=%v", invited, hasWorkspaceInviteErr)
	}
	if invited, hasWorkspaceInviteErr := repository.HasWorkspaceInvite(ctx, "deleted-only@example.test"); hasWorkspaceInviteErr != nil || invited {
		t.Fatalf("deleted-only invite existence=%v err=%v", invited, hasWorkspaceInviteErr)
	}
	onboarded, lastWorkspace, earliestWorkspace, inviteCount, err := repository.RedirectProfile(ctx, created.ID)
	if err != nil || !onboarded || lastWorkspace != "last-active" || earliestWorkspace != "earliest-active" || inviteCount != 2 {
		t.Fatalf("redirect profile onboarded=%v last=%q earliest=%q invites=%d err=%v", onboarded, lastWorkspace, earliestWorkspace, inviteCount, err)
	}
	if ensureProfileErr := repository.EnsureProfile(ctx, created.ID); ensureProfileErr != nil {
		t.Fatal(ensureProfileErr)
	}
	var preservedOnboarded bool
	if scanErr := connection.QueryRow(ctx, `SELECT is_onboarded FROM profiles WHERE user_id=$1`, created.ID).Scan(&preservedOnboarded); scanErr != nil || !preservedOnboarded {
		t.Fatalf("ensure changed existing profile onboarded=%v err=%v", preservedOnboarded, scanErr)
	}
	if _, execErr := connection.Exec(ctx, `DELETE FROM profiles WHERE user_id=$1`, created.ID); execErr != nil {
		t.Fatal(execErr)
	}
	if ensureProfileErr := repository.EnsureProfile(ctx, created.ID); ensureProfileErr != nil {
		t.Fatal(ensureProfileErr)
	}
	var defaultCountry, defaultLanguage string
	if scanErr := connection.QueryRow(ctx, `SELECT billing_address_country,language FROM profiles WHERE user_id=$1`, created.ID).Scan(&defaultCountry, &defaultLanguage); scanErr != nil {
		t.Fatal(scanErr)
	}
	if defaultCountry != "INDIA" || defaultLanguage != "en" {
		t.Fatalf("ensured profile defaults country=%q language=%q", defaultCountry, defaultLanguage)
	}

	logoutAt := time.Date(2026, 10, 21, 13, 0, 0, 0, time.UTC)
	if updateLogoutMetadataErr := repository.UpdateLogoutMetadata(ctx, created.ID, "192.0.2.9", logoutAt); updateLogoutMetadataErr != nil {
		t.Fatal(updateLogoutMetadataErr)
	}
	var logoutIP string
	var lastLogout time.Time
	if scanErr := connection.QueryRow(ctx, `SELECT last_logout_ip,last_logout_time FROM users WHERE id=$1`, created.ID).Scan(&logoutIP, &lastLogout); scanErr != nil {
		t.Fatal(scanErr)
	}
	if logoutIP != "192.0.2.9" || !lastLogout.Equal(logoutAt) {
		t.Fatalf("logout metadata ip=%q time=%v", logoutIP, lastLogout)
	}

	provisionedID := "00000000-0000-4000-8000-000000000001"
	_, err = connection.Exec(ctx, `INSERT INTO users (
id,username,email,display_name,first_name,last_name,password,avatar,date_joined,created_at,updated_at,last_location,
created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,
is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid
) VALUES ($1,'provisioned','provisioned@example.test','Provisioned','','','!', '',NOW(),NOW(),NOW(),'','',FALSE,FALSE,FALSE,FALSE,FALSE,FALSE,FALSE,FALSE,'','','','email','',FALSE,'UTC',FALSE)`, provisionedID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	activated, err := repository.UpdateLoginMetadata(ctx, provisionedID, now, "192.0.2.4", "Fixture Agent", "github", now)
	if err != nil || !activated {
		t.Fatalf("provisioned activation=%v err=%v", activated, err)
	}
	var active bool
	var token string
	var medium, ip, userAgent string
	var lastActive, tokenUpdated time.Time
	if err := connection.QueryRow(ctx, `SELECT is_active,token,last_login_medium,last_login_ip,last_login_uagent,last_active,token_updated_at FROM users WHERE id=$1`, provisionedID).Scan(
		&active, &token, &medium, &ip, &userAgent, &lastActive, &tokenUpdated,
	); err != nil {
		t.Fatal(err)
	}
	if !active || len(token) != 64 || medium != "github" || ip != "192.0.2.4" || userAgent != "Fixture Agent" || !lastActive.Equal(now) || !tokenUpdated.Equal(now) {
		t.Fatalf("login metadata active=%t token-len=%d medium=%q ip=%q ua=%q last_active=%v token_updated=%v", active, len(token), medium, ip, userAgent, lastActive, tokenUpdated)
	}
}
