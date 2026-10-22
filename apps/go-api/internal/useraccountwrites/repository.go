package useraccountwrites

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/useridentity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound      = errors.New("user account not found")
	ErrEmailTaken    = errors.New("email already exists")
	ErrInvalid       = errors.New("invalid account input")
	ErrCannotDisable = errors.New("account cannot be deactivated")
)

type Database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct {
	db       Database
	identity interface {
		SessionUser(context.Context, uuid.UUID) (json.RawMessage, error)
	}
}

func NewRepository(db Database, identity interface {
	SessionUser(context.Context, uuid.UUID) (json.RawMessage, error)
}) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("user account writes database is required")
	}
	if identity == nil {
		return nil, fmt.Errorf("user identity repository is required")
	}
	return &Repository{db: db, identity: identity}, nil
}

const userSerializerJSON = `json_build_object(
'id',u.id,'last_login',u.last_login,'is_superuser',u.is_superuser,'username',u.username,'mobile_number',u.mobile_number,
'email',u.email,'display_name',u.display_name,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,
'avatar_asset',u.avatar_asset_id,'cover_image',u.cover_image,'cover_image_asset',u.cover_image_asset_id,
'date_joined',u.date_joined,'created_at',u.created_at,'updated_at',u.updated_at,'last_location',u.last_location,
'created_location',u.created_location,'is_managed',u.is_managed,'is_password_expired',u.is_password_expired,
'is_active',u.is_active,'is_staff',u.is_staff,'is_email_verified',u.is_email_verified,'is_password_autoset',u.is_password_autoset,
'is_password_reset_required',u.is_password_reset_required,'token',u.token,'last_active',u.last_active,
'last_login_time',u.last_login_time,'last_logout_time',u.last_logout_time,'last_login_ip',u.last_login_ip,
'last_logout_ip',u.last_logout_ip,'last_login_medium',u.last_login_medium,'last_login_uagent',u.last_login_uagent,
'token_updated_at',u.token_updated_at,'is_bot',u.is_bot,'bot_type',u.bot_type,'user_timezone',u.user_timezone,
'is_email_valid',u.is_email_valid,'masked_at',u.masked_at)`

func (repository *Repository) UpdateMe(ctx context.Context, userID uuid.UUID, fields map[string]json.RawMessage) (json.RawMessage, error) {
	columns := map[string]string{
		"last_login": "last_login", "display_name": "display_name", "first_name": "first_name", "last_name": "last_name",
		"avatar": "avatar", "avatar_asset": "avatar_asset_id", "cover_image": "cover_image", "cover_image_asset": "cover_image_asset_id",
		"is_password_expired": "is_password_expired", "is_password_reset_required": "is_password_reset_required",
		"user_timezone": "user_timezone", "is_email_valid": "is_email_valid", "masked_at": "masked_at", "bot_type": "bot_type",
		"last_location": "last_location", "created_location": "created_location",
	}
	args := pgx.StrictNamedArgs{"user_id": userID}
	updates := make([]string, 0, len(fields)+2)
	for field, raw := range fields {
		column, ok := columns[field]
		if !ok {
			continue
		}
		value, err := dbValue(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", field, err)
		}
		if field == "avatar_asset" || field == "cover_image_asset" {
			column += ""
			args[field] = value
			updates = append(updates, column+"=CASE WHEN @"+field+" IS NULL THEN NULL ELSE @"+field+"::uuid END")
			continue
		}
		args[field] = value
		if field == "display_name" {
			updates = append(updates, column+"=COALESCE(NULLIF(@display_name,''),split_part(email,'@',1))")
		} else {
			updates = append(updates, column+"=@"+field)
		}
	}
	if value, ok := fields["first_name"]; ok {
		name := rawString(value)
		if containsURL(name) {
			return nil, FieldError{"first_name": {"First name cannot contain a URL."}}
		}
	}
	if value, ok := fields["last_name"]; ok {
		name := rawString(value)
		if containsURL(name) {
			return nil, FieldError{"last_name": {"Last name cannot contain a URL."}}
		}
	}
	updates = append(updates, "updated_at=NOW()")
	query := `UPDATE users u SET ` + strings.Join(updates, ",") + ` FROM users base
LEFT JOIN file_assets fa ON fa.id=base.avatar_asset_id
LEFT JOIN file_assets ca ON ca.id=base.cover_image_asset_id
WHERE u.id=@user_id AND base.id=u.id RETURNING ` + userSerializerJSON
	var body []byte
	if err := repository.db.QueryRow(ctx, query, args).Scan(&body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (repository *Repository) Me(ctx context.Context, userID uuid.UUID) (json.RawMessage, error) {
	body, err := repository.identity.SessionUser(ctx, userID)
	if errors.Is(err, useridentity.ErrNotFound) {
		return nil, ErrNotFound
	}
	return body, err
}

func (repository *Repository) EmailExists(ctx context.Context, userID uuid.UUID, email string) (bool, error) {
	var exists bool
	err := repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1 AND id<>$2)`, email, userID).Scan(&exists)
	return exists, err
}

func (repository *Repository) UpdateEmail(ctx context.Context, userID uuid.UUID, email string) (string, error) {
	var oldEmail string
	if err := repository.db.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, userID).Scan(&oldEmail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	_, err := repository.db.Exec(ctx, `UPDATE users SET email=$2,is_email_verified=false,updated_at=NOW() WHERE id=$1`, userID, email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", ErrEmailTaken
		}
		return "", err
	}
	return oldEmail, nil
}

func (repository *Repository) Logout(ctx context.Context, sessionID string) error {
	_, err := repository.db.Exec(ctx, `DELETE FROM sessions WHERE session_key=$1`, sessionID)
	return err
}

func (repository *Repository) DeactivationBlocker(ctx context.Context, userID uuid.UUID) (string, error) {
	var isInstanceAdmin bool
	if err := repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM instance_admins WHERE user_id=$1)`, userID).Scan(&isInstanceAdmin); err != nil {
		return "", err
	}
	if isInstanceAdmin {
		return "You cannot deactivate your account since you are an instance admin", nil
	}
	rows, err := repository.db.Query(ctx, `
SELECT mine.role, COUNT(all_members.id), COUNT(all_members.id) FILTER (WHERE all_members.role=20 AND all_members.member_id<>mine.member_id)
FROM project_members mine
JOIN project_members all_members ON all_members.project_id=mine.project_id AND all_members.is_active=true AND all_members.deleted_at IS NULL
WHERE mine.member_id=$1 AND mine.is_active=true AND mine.deleted_at IS NULL
GROUP BY mine.id,mine.role,mine.member_id`, userID)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var role int
		var members, otherAdmins int
		if err := rows.Scan(&role, &members, &otherAdmins); err != nil {
			rows.Close()
			return "", err
		}
		if role == 20 && otherAdmins == 0 && members > 1 {
			rows.Close()
			return "You cannot deactivate account as you are the only admin in some projects.", nil
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()
	rows, err = repository.db.Query(ctx, `
SELECT mine.role, COUNT(all_members.id), COUNT(all_members.id) FILTER (WHERE all_members.role=20 AND all_members.member_id<>mine.member_id)
FROM workspace_members mine
JOIN workspace_members all_members ON all_members.workspace_id=mine.workspace_id AND all_members.is_active=true AND all_members.deleted_at IS NULL
WHERE mine.member_id=$1 AND mine.is_active=true AND mine.deleted_at IS NULL
GROUP BY mine.id,mine.role,mine.member_id`, userID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var role int
		var members, otherAdmins int
		if err := rows.Scan(&role, &members, &otherAdmins); err != nil {
			return "", err
		}
		if role == 20 && otherAdmins == 0 && members > 1 {
			return "You cannot deactivate account as you are the only admin in some workspaces.", nil
		}
	}
	return "", rows.Err()
}

func (repository *Repository) Deactivate(ctx context.Context, userID uuid.UUID, passwordHash, lastLogoutIP string) error {
	if _, err := repository.db.Exec(ctx, `UPDATE project_members SET is_active=false WHERE member_id=$1 AND is_active=true AND deleted_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := repository.db.Exec(ctx, `UPDATE workspace_members SET is_active=false WHERE member_id=$1 AND is_active=true AND deleted_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := repository.db.Exec(ctx, `UPDATE workspace_member_invites SET deleted_at=NOW(),updated_at=NOW() WHERE email=(SELECT email FROM users WHERE id=$1) AND deleted_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := repository.db.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, userID.String()); err != nil {
		return err
	}
	profile, err := repository.db.Exec(ctx, `UPDATE profiles SET last_workspace_id=NULL,is_tour_completed=false,is_onboarded=false,onboarding_step='{"workspace_join":false,"profile_complete":false,"workspace_create":false,"workspace_invite":false}'::jsonb,updated_at=NOW() WHERE user_id=$1`, userID)
	if err != nil {
		return err
	}
	if profile.RowsAffected() == 0 {
		return ErrNotFound
	}
	user, err := repository.db.Exec(ctx, `UPDATE users SET password=$2,is_password_autoset=true,is_active=false,last_logout_ip=$3,last_logout_time=NOW(),updated_at=NOW() WHERE id=$1`, userID, passwordHash, lastLogoutIP)
	if err != nil {
		return err
	}
	if user.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func dbValue(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func rawString(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}

func containsURL(value string) bool {
	if len(value) > 1000 {
		return false
	}
	for _, line := range strings.Split(value, "\n") {
		if len(line) > 500 {
			line = line[:500]
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") || strings.Contains(lower, "www.") {
			return true
		}
		for i := 0; i < len(lower); i++ {
			if lower[i] == '.' && i > 0 && i+1 < len(lower) && strings.Contains("abcdefghijklmnopqrstuvwxyz", string(lower[i+1])) {
				return true
			}
		}
	}
	return false
}

type FieldError map[string][]string

func (err FieldError) Error() string { return "user account fields are invalid" }
