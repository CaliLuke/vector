package systemapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	accountauth "github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errOAuthSignupDisabled = &oauthFailure{code: 5015, name: "SIGNUP_DISABLED"}
	errOAuthInactive       = &oauthFailure{code: 5019, name: "USER_ACCOUNT_DEACTIVATED"}
	errOAuthBot            = &oauthFailure{code: 5017, name: "BOT_USER_LOGIN_FORBIDDEN"}
	errOAuthInvalidEmail   = &oauthFailure{code: 5005, name: "INVALID_EMAIL"}
)

type oauthRedirectPostgresRepository struct {
	db       *pgxpool.Pool
	accounts *accountauth.Repository
}

// NewOAuthRedirectPostgresRepository connects the generated OAuth service to
// Plane's PostgreSQL identity tables and the shared instance configuration
// reader. Configuration encryption follows the account-authentication service.
func NewOAuthRedirectPostgresRepository(db *pgxpool.Pool, cipher accountauth.ConfigCipher, useDBVars bool) (*oauthRedirectPostgresRepository, error) {
	if db == nil {
		return nil, errors.New("OAuth redirect database is required")
	}
	accounts, err := accountauth.NewPoolRepository(db, cipher, useDBVars)
	if err != nil {
		return nil, err
	}
	return &oauthRedirectPostgresRepository{db: db, accounts: accounts}, nil
}

func (repository *oauthRedirectPostgresRepository) InstanceConfigured(ctx context.Context) (bool, error) {
	return repository.accounts.InstanceConfigured(ctx)
}

func (repository *oauthRedirectPostgresRepository) ConfigValue(ctx context.Context, key, fallback string) (string, error) {
	return repository.accounts.ConfigValue(ctx, key, fallback)
}

func (repository *oauthRedirectPostgresRepository) AuthenticateOAuth(ctx context.Context, request oauthLoginRequest) (oauthLoginUser, error) {
	email := strings.ToLower(strings.TrimSpace(request.Identity.Email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, "@") {
		return oauthLoginUser{}, &oauthFailure{code: errOAuthInvalidEmail.code, name: errOAuthInvalidEmail.name, email: email}
	}
	var user oauthLoginUser
	var passwordHash string
	var active, bot bool
	var lastLogout *time.Time
	err = repository.db.QueryRow(ctx, `SELECT id::text,email,password,is_active,is_bot,last_logout_time FROM users WHERE email=@email ORDER BY created_at ASC LIMIT 1`, pgx.StrictNamedArgs{"email": email}).Scan(&user.ID, &user.Email, &passwordHash, &active, &bot, &lastLogout)
	if errors.Is(err, pgx.ErrNoRows) {
		user, err = repository.createOAuthUser(ctx, request, email)
		if err != nil {
			return oauthLoginUser{}, err
		}
		passwordHash = user.PasswordHash
		active = true
		user.WasNew = true
	} else if err != nil {
		return oauthLoginUser{}, fmt.Errorf("read OAuth user: %w", err)
	}
	if !active && lastLogout != nil {
		return oauthLoginUser{}, &oauthFailure{code: errOAuthInactive.code, name: errOAuthInactive.name, email: email}
	}
	if bot {
		return oauthLoginUser{}, &oauthFailure{code: errOAuthBot.code, name: errOAuthBot.name, email: email}
	}
	user.Active = active
	user.LastLogoutTimeSet = lastLogout != nil
	user.PasswordHash = passwordHash
	if !user.WasNew {
		if user.ID == "" {
			return oauthLoginUser{}, errors.New("OAuth user query returned an empty ID")
		}
		providerSync, configValueErr := repository.ConfigValue(ctx, "ENABLE_"+strings.ToUpper(request.Provider)+"_SYNC", "0")
		if configValueErr != nil {
			return oauthLoginUser{}, configValueErr
		}
		if providerSync == "1" {
			if _, execErr := repository.db.Exec(ctx, `UPDATE users SET first_name=@first_name,last_name=@last_name,display_name=COALESCE(NULLIF(@display_name,''),split_part(email,'@',1)),avatar=@avatar,updated_at=@now WHERE id=@user_id`, pgx.StrictNamedArgs{
				"first_name": request.Identity.FirstName, "last_name": request.Identity.LastName,
				"display_name": strings.SplitN(user.Email, "@", 2)[0], "avatar": request.Identity.Avatar, "now": request.Now, "user_id": user.ID,
			}); execErr != nil {
				return oauthLoginUser{}, fmt.Errorf("sync OAuth profile: %w", execErr)
			}
		}
	}
	_, err = repository.db.Exec(ctx, `UPDATE users SET last_login_medium=@provider,last_active=@now,last_login_time=@now,last_login_ip=@client_ip,last_login_uagent=@user_agent,token_updated_at=@now,is_active=true,updated_at=@now WHERE id=@user_id`, pgx.StrictNamedArgs{
		"provider": request.Provider, "now": request.Now, "client_ip": request.ClientIP,
		"user_agent": request.UserAgent, "user_id": user.ID,
	})
	if err != nil {
		return oauthLoginUser{}, fmt.Errorf("record OAuth login metadata: %w", err)
	}
	user.Active = true
	return user, nil
}

func (repository *oauthRedirectPostgresRepository) createOAuthUser(ctx context.Context, request oauthLoginRequest, email string) (oauthLoginUser, error) {
	allowSignup, err := repository.ConfigValue(ctx, "ENABLE_SIGNUP", "1")
	if err != nil {
		return oauthLoginUser{}, err
	}
	if allowSignup == "0" {
		var invited bool
		if scanErr := repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_member_invites WHERE lower(email)=@email)`, pgx.StrictNamedArgs{"email": email}).Scan(&invited); scanErr != nil {
			return oauthLoginUser{}, fmt.Errorf("check OAuth signup invitation: %w", scanErr)
		}
		if !invited {
			return oauthLoginUser{}, &oauthFailure{code: errOAuthSignupDisabled.code, name: errOAuthSignupDisabled.name, email: email}
		}
	}
	randomPassword, err := oauthRandomToken()
	if err != nil {
		return oauthLoginUser{}, err
	}
	hasher := password.Default()
	passwordHash, err := hasher.Encode(randomPassword)
	if err != nil {
		return oauthLoginUser{}, err
	}
	userID := uuid.New()
	username, err := oauthRandomToken()
	if err != nil {
		return oauthLoginUser{}, err
	}
	tokenPart, err := oauthRandomToken()
	if err != nil {
		return oauthLoginUser{}, err
	}
	tokenPart2, err := oauthRandomToken()
	if err != nil {
		return oauthLoginUser{}, err
	}
	_, err = repository.db.Exec(ctx, `INSERT INTO users (id,username,email,password,mobile_number,first_name,last_name,display_name,avatar,avatar_asset_id,cover_image,cover_image_asset_id,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_active,last_login_time,last_logout_time,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,token_updated_at,is_bot,bot_type,user_timezone,is_email_valid,masked_at) VALUES (@id,@username,@email,@password,NULL,@first_name,@last_name,split_part(@email,'@',1),@avatar,NULL,NULL,NULL,@now,@now,@now,'','',false,false,false,true,false,true,true,false,@token,@now,@now,NULL,@client_ip,'',@provider,@user_agent,@now,false,NULL,'UTC',false,NULL)`, pgx.StrictNamedArgs{
		"id": userID, "username": username, "email": email, "password": passwordHash,
		"first_name": request.Identity.FirstName, "last_name": request.Identity.LastName,
		"avatar": request.Identity.Avatar, "token": tokenPart + tokenPart2,
		"provider": request.Provider, "now": request.Now, "client_ip": request.ClientIP, "user_agent": request.UserAgent,
	})
	if err != nil {
		return oauthLoginUser{}, fmt.Errorf("create OAuth user: %w", err)
	}
	if createDefaultProfileErr := repository.createDefaultProfile(ctx, userID, request.Now); createDefaultProfileErr != nil {
		return oauthLoginUser{}, createDefaultProfileErr
	}
	// User.post_save creates this preference row for non-bot accounts.
	_, err = repository.db.Exec(ctx, `INSERT INTO user_notification_preferences (id,user_id,workspace_id,project_id,property_change,state,comment,mention,issue_completed,created_at,updated_at) VALUES (@id,@user_id,NULL,NULL,true,true,true,true,true,@now,@now)`, pgx.StrictNamedArgs{"id": uuid.New(), "user_id": userID, "now": request.Now})
	if err != nil {
		return oauthLoginUser{}, fmt.Errorf("create OAuth notification preference: %w", err)
	}
	return oauthLoginUser{ID: userID.String(), Email: email, PasswordHash: passwordHash, Active: true, WasNew: true}, nil
}

func (repository *oauthRedirectPostgresRepository) createDefaultProfile(ctx context.Context, userID uuid.UUID, now time.Time) error {
	color, err := oauthRandomColor()
	if err != nil {
		return fmt.Errorf("generate OAuth profile color: %w", err)
	}
	_, err = repository.db.Exec(ctx, `INSERT INTO profiles (id,user_id,theme,is_app_rail_docked,is_tour_completed,onboarding_step,use_case,role,is_onboarded,last_workspace_id,billing_address_country,billing_address,has_billing_address,company_name,notification_view_mode,is_smooth_cursor_enabled,is_mobile_onboarded,mobile_onboarding_step,mobile_timezone_auto_set,language,start_of_the_week,goals,background_color,is_navigation_tour_completed,has_marketing_email_consent,is_subscribed_to_changelog,product_tour,created_at,updated_at) VALUES (@id,@user_id,'{}'::jsonb,true,false,'{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":false}'::jsonb,NULL,NULL,false,NULL,'INDIA',NULL,false,'','full',false,false,'{"profile_complete":false,"workspace_create":false,"workspace_join":false}'::jsonb,false,'en',0,'{}'::jsonb,@background_color,false,false,false,'{"work_items":false,"cycles":false,"modules":false,"intake":false,"pages":false}'::jsonb,@now,@now)`, pgx.StrictNamedArgs{"id": uuid.New(), "user_id": userID, "background_color": color, "now": now})
	if err != nil {
		return fmt.Errorf("create OAuth profile: %w", err)
	}
	return nil
}

func (repository *oauthRedirectPostgresRepository) SaveOAuthAccount(ctx context.Context, user oauthLoginUser, provider, providerID string, tokens oauthTokens, now time.Time) error {
	var accountID uuid.UUID
	err := repository.db.QueryRow(ctx, `SELECT id FROM accounts WHERE user_id=@user_id AND provider=@provider AND provider_account_id=@provider_account_id ORDER BY created_at DESC LIMIT 1`, pgx.StrictNamedArgs{"user_id": user.ID, "provider": provider, "provider_account_id": providerID}).Scan(&accountID)
	args := pgx.StrictNamedArgs{"id": uuid.New(), "user_id": user.ID, "provider": provider, "provider_account_id": providerID, "access_token": tokens.AccessToken, "refresh_token": nullableString(tokens.RefreshToken), "access_token_expired_at": tokens.ExpiresAt, "refresh_token_expired_at": tokens.RefreshExpiresAt, "last_connected_at": now, "id_token": tokens.IDToken, "created_at": now, "updated_at": now}
	if err == nil {
		args["id"] = accountID
		_, err = repository.db.Exec(ctx, `UPDATE accounts SET access_token=@access_token,refresh_token=@refresh_token,access_token_expired_at=@access_token_expired_at,refresh_token_expired_at=@refresh_token_expired_at,last_connected_at=@last_connected_at,id_token=@id_token,updated_at=@updated_at WHERE id=@id`, args)
		if err != nil {
			return fmt.Errorf("update OAuth account: %w", err)
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read OAuth account: %w", err)
	}
	_, err = repository.db.Exec(ctx, `INSERT INTO accounts (id,user_id,provider,provider_account_id,access_token,refresh_token,access_token_expired_at,refresh_token_expired_at,last_connected_at,id_token,metadata,created_at,updated_at) VALUES (@id,@user_id,@provider,@provider_account_id,@access_token,@refresh_token,@access_token_expired_at,@refresh_token_expired_at,@last_connected_at,@id_token,'{}'::jsonb,@created_at,@updated_at)`, args)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			// Django logs account uniqueness failures and still completes the login.
			return nil
		}
		return fmt.Errorf("create OAuth account: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (repository *oauthRedirectPostgresRepository) LoginRedirectPath(ctx context.Context, user oauthLoginUser) (string, error) {
	var onboarded bool
	var lastWorkspace *uuid.UUID
	err := repository.db.QueryRow(ctx, `SELECT is_onboarded,last_workspace_id FROM profiles WHERE user_id=@user_id`, pgx.StrictNamedArgs{"user_id": user.ID}).Scan(&onboarded, &lastWorkspace)
	if errors.Is(err, pgx.ErrNoRows) {
		if createDefaultProfileErr := repository.createDefaultProfile(ctx, uuid.MustParse(user.ID), time.Now()); createDefaultProfileErr != nil {
			return "", createDefaultProfileErr
		}
		onboarded = false
	} else if err != nil {
		return "", fmt.Errorf("read OAuth redirect profile: %w", err)
	}
	if !onboarded {
		return "onboarding", nil
	}
	if lastWorkspace != nil {
		var slug string
		scanErr := repository.db.QueryRow(ctx, `SELECT w.slug FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE w.id=@workspace_id AND wm.member_id=@user_id AND wm.is_active=true LIMIT 1`, pgx.StrictNamedArgs{"workspace_id": *lastWorkspace, "user_id": user.ID}).Scan(&slug)
		if scanErr == nil {
			return slug, nil
		}
		if !errors.Is(scanErr, pgx.ErrNoRows) {
			return "", fmt.Errorf("resolve OAuth last workspace: %w", scanErr)
		}
	}
	var slug string
	err = repository.db.QueryRow(ctx, `SELECT w.slug FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE wm.member_id=@user_id AND wm.is_active=true ORDER BY w.created_at ASC LIMIT 1`, pgx.StrictNamedArgs{"user_id": user.ID}).Scan(&slug)
	if err == nil {
		return slug, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("resolve OAuth first workspace: %w", err)
	}
	var invited bool
	if err := repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_member_invites WHERE email=@email)`, pgx.StrictNamedArgs{"email": user.Email}).Scan(&invited); err != nil {
		return "", fmt.Errorf("check OAuth redirect invitations: %w", err)
	}
	if invited {
		return "invitations", nil
	}
	return "create-workspace", nil
}

func oauthRandomToken() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func oauthRandomColor() (string, error) {
	const alphabet = "0123456789abcdefABCDEF"
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	var color strings.Builder
	color.Grow(7)
	color.WriteByte('#')
	for _, value := range bytes {
		color.WriteByte(alphabet[int(value)%len(alphabet)])
	}
	return color.String(), nil
}
