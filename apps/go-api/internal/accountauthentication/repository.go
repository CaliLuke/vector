package accountauthentication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInstanceNotConfigured = errors.New("instance is not configured")
	ErrUserNotFound          = errors.New("account user not found")
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type ConfigCipher interface {
	Decrypt(string) (string, error)
}

type User struct {
	ID                 string
	Username           string
	Email              string
	FirstName          string
	Password           string
	PasswordAutoset    bool
	PasswordReset      bool
	Active             bool
	EmailVerified      bool
	LastLogin          *time.Time
	LastLoginTime      *time.Time
	LastLoginIP        string
	LastLoginUserAgent string
	LastLogoutTime     *time.Time
	IsBot              bool
}

type Repository struct {
	db        Database
	cipher    ConfigCipher
	useDBVars bool
}

func NewRepository(db Database, cipher ConfigCipher, useDBVars bool) (*Repository, error) {
	if db == nil {
		return nil, errors.New("account authentication database is required")
	}
	if useDBVars && cipher == nil {
		return nil, errors.New("account authentication configuration cipher is required when database configuration is enabled")
	}
	return &Repository{db: db, cipher: cipher, useDBVars: useDBVars}, nil
}

func NewPoolRepository(pool *pgxpool.Pool, cipher ConfigCipher, useDBVars bool) (*Repository, error) {
	return NewRepository(pool, cipher, useDBVars)
}

func (repository *Repository) InstanceConfigured(ctx context.Context) (bool, error) {
	var configured bool
	err := repository.db.QueryRow(ctx, `SELECT is_setup_done FROM instances WHERE deleted_at IS NULL ORDER BY created_at ASC LIMIT 1`).Scan(&configured)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read account authentication instance setup state: %w", err)
	}
	return configured, nil
}

// InstanceExists follows the admin sign-in source, which checks for an
// instance row rather than requiring setup to be complete.
func (repository *Repository) InstanceExists(ctx context.Context) (bool, error) {
	var exists bool
	err := repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM instances WHERE deleted_at IS NULL)`).Scan(&exists)
	return exists, err
}

// ConfigValue follows get_configuration_value's source switch: database
// settings are authoritative when SKIP_ENV_VAR is set; otherwise use only the
// process environment and fallback value.
func (repository *Repository) ConfigValue(ctx context.Context, key, fallback string) (string, error) {
	if !repository.useDBVars {
		if value, ok := os.LookupEnv(key); ok {
			return value, nil
		}
		return fallback, nil
	}
	var value string
	var encrypted bool
	err := repository.db.QueryRow(ctx, `SELECT value,is_encrypted FROM instance_configurations WHERE key=@key ORDER BY created_at ASC LIMIT 1`, pgx.StrictNamedArgs{"key": key}).Scan(&value, &encrypted)
	if errors.Is(err, pgx.ErrNoRows) {
		return fallback, nil
	}
	if err != nil {
		return "", fmt.Errorf("read account authentication configuration %s: %w", key, err)
	}
	if encrypted {
		value, err = repository.cipher.Decrypt(value)
		if err != nil {
			return "", fmt.Errorf("decrypt account authentication configuration %s: %w", key, err)
		}
	}
	return value, nil
}

func (repository *Repository) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := scanUser(repository.db.QueryRow(ctx, userColumns+` WHERE email=@email ORDER BY created_at ASC LIMIT 1`, pgx.StrictNamedArgs{"email": email}), &user)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("read account authentication user: %w", err)
	}
	return user, nil
}

func (repository *Repository) UserByID(ctx context.Context, id string) (User, error) {
	var user User
	err := scanUser(repository.db.QueryRow(ctx, userColumns+` WHERE id=@id`, pgx.StrictNamedArgs{"id": id}), &user)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("read account authentication user: %w", err)
	}
	return user, nil
}

// UserJSON mirrors the all-model-fields UserSerializer response returned after
// SetUserPasswordEndpoint. That serializer intentionally differs from the
// compact UserMeSerializer used by ordinary authenticated profile reads.
func (repository *Repository) UserJSON(ctx context.Context, id string) (json.RawMessage, error) {
	var body json.RawMessage
	err := repository.db.QueryRow(ctx, `SELECT
  (to_jsonb(users) - 'password' - 'avatar_asset_id' - 'cover_image_asset_id') ||
  jsonb_build_object(
    'avatar_asset', avatar_asset_id,
    'cover_image_asset', cover_image_asset_id
  )
FROM users WHERE id=@id`, pgx.StrictNamedArgs{"id": id}).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read account user serializer projection: %w", err)
	}
	return body, nil
}

const userColumns = `SELECT id::text,username,email,first_name,password,is_password_autoset,is_password_reset_required,is_active,is_email_verified,last_login,last_login_time,COALESCE(last_login_ip,''),COALESCE(last_login_uagent,''),last_logout_time,is_bot FROM users`

func scanUser(row pgx.Row, user *User) error {
	return row.Scan(
		&user.ID, &user.Username, &user.Email, &user.FirstName, &user.Password, &user.PasswordAutoset,
		&user.PasswordReset, &user.Active, &user.EmailVerified, &user.LastLogin, &user.LastLoginTime,
		&user.LastLoginIP, &user.LastLoginUserAgent, &user.LastLogoutTime, &user.IsBot,
	)
}

// CreateUser mirrors the user-creation defaults used by Django's signup path,
// including the post-save notification-preference row. Username is supplied by
// the auth workflow so the same generated identity value is available to audit.
func (repository *Repository) CreateUser(ctx context.Context, email, username, passwordHash string, passwordAutoset, emailVerified bool) (result User, resultErr error) {
	tx, err := repository.db.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin account user creation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()

	userID := uuid.NewString()
	profileID := uuid.NewString()
	preferenceID := uuid.NewString()
	now := time.Now().UTC()
	displayName := email
	if local, _, found := strings.Cut(email, "@"); found {
		displayName = local
	}
	_, err = tx.Exec(ctx, `
INSERT INTO users (
  id,username,email,display_name,first_name,last_name,password,last_login,
  avatar,avatar_asset_id,cover_image,cover_image_asset_id,date_joined,created_at,updated_at,
  last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,
  is_email_verified,is_password_autoset,is_password_reset_required,token,last_active,last_login_time,
  last_logout_time,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,token_updated_at,
  is_bot,bot_type,user_timezone,is_email_valid,masked_at
) VALUES (
  @id,@username,@email,@display_name,'','',@password,NULL,
  '',NULL,NULL,NULL,@now,@now,@now,
  '','',FALSE,FALSE,FALSE,TRUE,FALSE,
  @email_verified,@password_autoset,FALSE,'',@now,NULL,
  NULL,'','','email','',NULL,
  FALSE,NULL,'UTC',FALSE,NULL
)`, pgx.StrictNamedArgs{
		"id": userID, "username": username, "email": email, "display_name": displayName,
		"password": passwordHash, "email_verified": emailVerified, "password_autoset": passwordAutoset, "now": now,
	})
	if err != nil {
		return User{}, fmt.Errorf("create account user: %w", err)
	}
	if err = insertDefaultProfile(ctx, tx, userID, now, profileID); err != nil {
		return User{}, fmt.Errorf("create account profile: %w", err)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO user_notification_preferences (
  id,user_id,workspace_id,project_id,property_change,state_change,comment,mention,issue_completed,created_at,updated_at
) VALUES (@id,@user_id,NULL,NULL,TRUE,TRUE,TRUE,TRUE,TRUE,@now,@now)`, pgx.StrictNamedArgs{
		"id": preferenceID, "user_id": userID, "now": now,
	})
	if err != nil {
		return User{}, fmt.Errorf("create account notification preferences: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit account user creation: %w", err)
	}
	return User{
		ID: userID, Username: username, Email: email, FirstName: "", Password: passwordHash,
		PasswordAutoset: passwordAutoset, Active: true, EmailVerified: emailVerified, IsBot: false,
	}, nil
}

type accountExec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertDefaultProfile(ctx context.Context, executor accountExec, userID string, now time.Time, profileID string) error {
	_, err := executor.Exec(ctx, `
INSERT INTO profiles (
  id,user_id,theme,is_app_rail_docked,is_tour_completed,onboarding_step,use_case,role,is_onboarded,
  last_workspace_id,billing_address_country,billing_address,has_billing_address,company_name,
  notification_view_mode,is_smooth_cursor_enabled,is_mobile_onboarded,mobile_onboarding_step,
  mobile_timezone_auto_set,language,start_of_the_week,goals,background_color,
  is_navigation_tour_completed,has_marketing_email_consent,is_subscribed_to_changelog,product_tour,
  created_at,updated_at
) VALUES (
  @id,@user_id,'{}'::jsonb,TRUE,FALSE,
  '{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":false}'::jsonb,
  NULL,NULL,FALSE,NULL,'INDIA',NULL,FALSE,'','full',FALSE,FALSE,
  '{"profile_complete":false,"workspace_create":false,"workspace_join":false}'::jsonb,
  FALSE,'en',0,'{}'::jsonb,'#' || substr(md5(random()::text),1,6),FALSE,FALSE,FALSE,
  '{"work_items":false,"cycles":false,"modules":false,"intake":false,"pages":false}'::jsonb,
  @now,@now
) ON CONFLICT (user_id) DO NOTHING`, pgx.StrictNamedArgs{"id": profileID, "user_id": userID, "now": now})
	return err
}

// EnsureProfile mirrors Profile.objects.get_or_create(user=user), including
// Django's model defaults when an older user has no profile row.
func (repository *Repository) EnsureProfile(ctx context.Context, userID string) (resultErr error) {
	tx, err := repository.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin account profile ensure: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if err := insertDefaultProfile(ctx, tx, userID, time.Now().UTC(), uuid.NewString()); err != nil {
		return fmt.Errorf("ensure account profile: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit account profile ensure: %w", err)
	}
	return nil
}

func (repository *Repository) HasWorkspaceInvite(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := repository.db.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM workspace_member_invites
  WHERE email=@email AND deleted_at IS NULL
)`, pgx.StrictNamedArgs{"email": email}).Scan(&exists); err != nil {
		return false, fmt.Errorf("check account workspace invitations: %w", err)
	}
	return exists, nil
}

func (repository *Repository) RedirectProfile(ctx context.Context, userID string) (bool, string, string, int, error) {
	if err := repository.EnsureProfile(ctx, userID); err != nil {
		return false, "", "", 0, err
	}
	var result struct {
		Onboarded         bool
		LastWorkspace     string
		EarliestWorkspace string
		InviteCount       int
	}
	err := repository.db.QueryRow(ctx, `
SELECT profile.is_onboarded,
       COALESCE((
         SELECT workspace.slug
         FROM workspaces workspace
         JOIN workspace_members membership ON membership.workspace_id=workspace.id
         WHERE workspace.id=profile.last_workspace_id
           AND workspace.deleted_at IS NULL
           AND membership.member_id=profile.user_id
           AND membership.is_active
           AND membership.deleted_at IS NULL
         ORDER BY workspace.created_at DESC
         LIMIT 1
       ), ''),
       COALESCE((
         SELECT workspace.slug
         FROM workspaces workspace
         JOIN workspace_members membership ON membership.workspace_id=workspace.id
         WHERE workspace.deleted_at IS NULL
           AND membership.member_id=profile.user_id
           AND membership.is_active
           AND membership.deleted_at IS NULL
         ORDER BY workspace.created_at ASC
         LIMIT 1
       ), ''),
       (SELECT count(*)::int
        FROM workspace_member_invites invitation
        JOIN users account ON account.id=profile.user_id
        WHERE invitation.email=account.email AND invitation.deleted_at IS NULL)
FROM profiles profile
WHERE profile.user_id=@user_id`, pgx.StrictNamedArgs{"user_id": userID}).Scan(
		&result.Onboarded, &result.LastWorkspace, &result.EarliestWorkspace, &result.InviteCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", "", 0, ErrUserNotFound
	}
	if err != nil {
		return false, "", "", 0, fmt.Errorf("read account redirect profile: %w", err)
	}
	return result.Onboarded, result.LastWorkspace, result.EarliestWorkspace, result.InviteCount, nil
}

func (repository *Repository) UpdateLogoutMetadata(ctx context.Context, userID, ip string, now time.Time) error {
	tag, err := repository.db.Exec(ctx, `
UPDATE users
SET last_logout_ip=@ip,last_logout_time=@now,updated_at=@now
WHERE id=@user_id`, pgx.StrictNamedArgs{"user_id": userID, "ip": ip, "now": now})
	if err != nil {
		return fmt.Errorf("update account logout metadata: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateLoginMetadata persists the provider-specific login fields and activates
// only a provisioned account that has never been explicitly deactivated.
func (repository *Repository) UpdateLoginMetadata(ctx context.Context, userID string, now time.Time, ip, userAgent, medium string, tokenUpdatedAt time.Time) (bool, error) {
	token := strings.ReplaceAll(uuid.NewString(), "-", "") + strings.ReplaceAll(uuid.NewString(), "-", "")
	var activated bool
	err := repository.db.QueryRow(ctx, `
WITH current AS (
  SELECT is_active,last_logout_time FROM users WHERE id=@user_id FOR UPDATE
), updated AS (
  UPDATE users SET last_login_medium=@medium,last_active=@now,last_login_time=@now,
    last_login_ip=@ip,last_login_uagent=@user_agent,token=@token,token_updated_at=@token_updated_at,
    is_active=CASE WHEN last_logout_time IS NULL THEN TRUE ELSE is_active END,updated_at=@now
  WHERE id=@user_id
  RETURNING id
)
SELECT (NOT current.is_active AND current.last_logout_time IS NULL)
FROM current CROSS JOIN updated`, pgx.StrictNamedArgs{
		"user_id": userID, "medium": medium, "now": now, "ip": ip, "user_agent": userAgent,
		"token": token, "token_updated_at": tokenUpdatedAt,
	}).Scan(&activated)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrUserNotFound
	}
	if err != nil {
		return false, fmt.Errorf("update account login metadata: %w", err)
	}
	return activated, nil
}

func (repository *Repository) UpdatePassword(ctx context.Context, userID, encoded string, autoset bool, lastLogin time.Time) error {
	_, err := repository.db.Exec(ctx, `UPDATE users SET password=@password,is_password_autoset=@autoset,last_login=@last_login,updated_at=@last_login WHERE id=@user_id`, pgx.StrictNamedArgs{
		"password": encoded, "autoset": autoset, "last_login": lastLogin, "user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("update account password: %w", err)
	}
	return nil
}

func (repository *Repository) SetPassword(ctx context.Context, userID, encoded string, autoset bool) error {
	_, err := repository.db.Exec(ctx, `UPDATE users SET password=@password,is_password_autoset=@autoset,updated_at=NOW() WHERE id=@user_id`, pgx.StrictNamedArgs{
		"password": encoded, "autoset": autoset, "user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("set account password: %w", err)
	}
	return nil
}

func (repository *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=@email)`, pgx.StrictNamedArgs{"email": email}).Scan(&exists); err != nil {
		return false, fmt.Errorf("check account email: %w", err)
	}
	return exists, nil
}
