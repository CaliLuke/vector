package accountauthentication

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type InitialAdminResult struct {
	User              User
	Exists            bool
	SetupDone         bool
	AlreadyExists     bool
	UserAlreadyExists bool
}

func (repository *Repository) InitialAdminState(ctx context.Context) (exists, setupDone, hasAdmins bool, err error) {
	err = repository.db.QueryRow(ctx, `SELECT id::text,is_setup_done FROM instances
WHERE deleted_at IS NULL ORDER BY created_at ASC LIMIT 1`).Scan(new(string), &setupDone)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, false, nil
	}
	if err != nil {
		return false, false, false, fmt.Errorf("read initial admin setup state: %w", err)
	}
	if err = repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM instance_admins WHERE deleted_at IS NULL)`).Scan(&hasAdmins); err != nil {
		return false, false, false, fmt.Errorf("check initial admin state: %w", err)
	}
	return true, setupDone, hasAdmins, nil
}

func (repository *Repository) IsInstanceAdmin(ctx context.Context, instanceConfigured bool, userID string) (bool, error) {
	if !instanceConfigured {
		return false, nil
	}
	var allowed bool
	err := repository.db.QueryRow(ctx, `SELECT EXISTS (
SELECT 1 FROM instance_admins a
JOIN instances i ON i.id=a.instance_id
WHERE i.deleted_at IS NULL AND a.deleted_at IS NULL AND a.user_id=@user_id
)`, pgx.StrictNamedArgs{"user_id": userID}).Scan(&allowed)
	return allowed, err
}

// CreateInitialAdmin serializes the first-admin transition on the instance row.
// It uses the same profile defaults as account signup, then creates the admin
// relation and flips setup state in the same transaction.
func (repository *Repository) CreateInitialAdmin(
	ctx context.Context,
	email, firstName, lastName, companyName, passwordHash, clientIP, userAgent string,
	telemetry bool,
	now time.Time,
) (result InitialAdminResult, resultErr error) {
	tx, err := repository.db.Begin(ctx)
	if err != nil {
		return InitialAdminResult{}, fmt.Errorf("begin initial admin creation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()

	var instanceID string
	var setupDone bool
	err = tx.QueryRow(ctx, `SELECT id::text,is_setup_done FROM instances
WHERE deleted_at IS NULL ORDER BY created_at ASC LIMIT 1 FOR UPDATE`).Scan(&instanceID, &setupDone)
	if errors.Is(err, pgx.ErrNoRows) {
		return InitialAdminResult{}, nil
	}
	if err != nil {
		return InitialAdminResult{}, fmt.Errorf("lock instance for initial admin: %w", err)
	}
	var hasAdmins bool
	if scanErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM instance_admins WHERE deleted_at IS NULL)`).Scan(&hasAdmins); scanErr != nil {
		return InitialAdminResult{}, fmt.Errorf("check existing instance admins: %w", scanErr)
	}
	if setupDone || hasAdmins {
		return InitialAdminResult{Exists: true, SetupDone: setupDone, AlreadyExists: hasAdmins}, nil
	}
	var existingUser bool
	if scanErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=@email)`, pgx.StrictNamedArgs{"email": email}).Scan(&existingUser); scanErr != nil {
		return InitialAdminResult{}, fmt.Errorf("check existing admin user: %w", scanErr)
	}
	if existingUser {
		return InitialAdminResult{Exists: true, UserAlreadyExists: true}, nil
	}

	userID := uuid.NewString()
	profileID := uuid.NewString()
	preferenceID := uuid.NewString()
	username := strings.ReplaceAll(uuid.NewString(), "-", "")
	token := strings.ReplaceAll(uuid.NewString()+uuid.NewString(), "-", "")
	displayName := email
	if local, _, found := strings.Cut(email, "@"); found {
		displayName = local
	}
	if now.IsZero() {
		now = time.Now().UTC()
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
 @id,@username,@email,@display_name,@first_name,@last_name,@password,NULL,
 '',NULL,NULL,NULL,@now,@now,@now,'','',FALSE,FALSE,FALSE,TRUE,FALSE,
 FALSE,FALSE,FALSE,@token,@now,@now,NULL,@client_ip,'','email',@user_agent,@now,
 FALSE,NULL,'UTC',FALSE,NULL
)`, pgx.StrictNamedArgs{
		"id": userID, "username": username, "email": email, "display_name": displayName,
		"first_name": firstName, "last_name": lastName, "password": passwordHash, "token": token,
		"now": now, "client_ip": clientIP, "user_agent": userAgent,
	})
	if err != nil {
		return InitialAdminResult{}, fmt.Errorf("create initial admin user: %w", err)
	}
	if err := insertDefaultProfile(ctx, tx, userID, now, profileID); err != nil {
		return InitialAdminResult{}, fmt.Errorf("create initial admin profile: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE profiles SET company_name=@company WHERE user_id=@user_id`,
		pgx.StrictNamedArgs{"company": companyName, "user_id": userID}); err != nil {
		return InitialAdminResult{}, fmt.Errorf("set initial admin company: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_notification_preferences
(id,user_id,workspace_id,project_id,property_change,state_change,comment,mention,issue_completed,created_at,updated_at)
VALUES(@id,@user_id,NULL,NULL,TRUE,TRUE,TRUE,TRUE,TRUE,@now,@now)`,
		pgx.StrictNamedArgs{"id": preferenceID, "user_id": userID, "now": now}); err != nil {
		return InitialAdminResult{}, fmt.Errorf("create initial admin notification preferences: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO instance_admins
(id,instance_id,user_id,role,is_verified,created_at,updated_at)
VALUES(@id,@instance_id,@user_id,20,false,@now,@now)`,
		pgx.StrictNamedArgs{"id": uuid.NewString(), "instance_id": instanceID, "user_id": userID, "now": now}); err != nil {
		return InitialAdminResult{}, fmt.Errorf("create initial admin relation: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE instances SET is_setup_done=true,instance_name=@company,
is_telemetry_enabled=@telemetry,updated_at=@now WHERE id=@instance_id`,
		pgx.StrictNamedArgs{"company": companyName, "telemetry": telemetry, "now": now, "instance_id": instanceID}); err != nil {
		return InitialAdminResult{}, fmt.Errorf("complete initial instance setup: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return InitialAdminResult{}, fmt.Errorf("commit initial admin creation: %w", err)
	}
	return InitialAdminResult{Exists: true, SetupDone: true, User: User{
		ID: userID, Username: username, Email: email, FirstName: firstName, Password: passwordHash,
		Active: true, PasswordAutoset: false, IsBot: false,
	}}, nil
}
