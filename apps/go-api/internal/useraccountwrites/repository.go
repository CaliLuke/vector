package useraccountwrites

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound   = errors.New("user account not found")
	ErrEmailTaken = errors.New("email already exists")
	ErrInvalid    = errors.New("invalid account input")
)

type Database interface {
	Begin(context.Context) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct {
	db       Database
	sessions *browsersession.Validator
	identity interface {
		SessionUser(context.Context, uuid.UUID) (json.RawMessage, error)
	}
}

func NewRepository(db Database, identity interface {
	SessionUser(context.Context, uuid.UUID) (json.RawMessage, error)
}, sessions *browsersession.Validator) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("user account writes database is required")
	}
	if identity == nil {
		return nil, fmt.Errorf("user identity repository is required")
	}
	if sessions == nil {
		return nil, errors.New("account session validator is required")
	}
	return &Repository{db: db, identity: identity, sessions: sessions}, nil
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
