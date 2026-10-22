package useridentity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"uuid"
)

var ErrNotFound = errors.New("user identity record not found")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("user identity database is required")
	}
	return &Repository{db: db}, nil
}

const avatarURLSQL = `CASE
  WHEN u.avatar_asset_id IS NULL THEN NULLIF(u.avatar,'')
  WHEN avatar_asset.entity_type IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/' || avatar_asset.id::text || '/'
  WHEN avatar_asset.entity_type='ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/' || avatar_workspace.slug || '/projects/' || COALESCE(avatar_asset.project_id::text,'None') || '/issues/' || COALESCE(avatar_asset.issue_id::text,'None') || '/attachments/' || avatar_asset.id::text || '/'
  WHEN avatar_asset.entity_type IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/' || avatar_workspace.slug || '/projects/' || COALESCE(avatar_asset.project_id::text,'None') || '/' || avatar_asset.id::text || '/'
  ELSE NULL END`

const sessionUserJSON = `json_build_object(
'id',u.id,
'avatar',u.avatar,
'cover_image',u.cover_image,
'avatar_url',` + avatarURLSQL + `,
'cover_image_url',CASE
  WHEN u.cover_image_asset_id IS NULL THEN NULLIF(u.cover_image,'')
  WHEN cover_asset.entity_type IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/' || cover_asset.id::text || '/'
  WHEN cover_asset.entity_type='ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/' || cover_workspace.slug || '/projects/' || COALESCE(cover_asset.project_id::text,'None') || '/issues/' || COALESCE(cover_asset.issue_id::text,'None') || '/attachments/' || cover_asset.id::text || '/'
  WHEN cover_asset.entity_type IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/' || cover_workspace.slug || '/projects/' || COALESCE(cover_asset.project_id::text,'None') || '/' || cover_asset.id::text || '/'
  ELSE NULL END,
'date_joined',to_char(u.date_joined,'YYYY-MM-DD"T"HH24:MI:SS') || CASE WHEN to_char(u.date_joined,'US') <> '000000' THEN '.' || to_char(u.date_joined,'US') ELSE '' END || CASE WHEN to_char(u.date_joined,'OF') IN ('+00','-00') THEN 'Z' WHEN length(to_char(u.date_joined,'OF'))=3 THEN to_char(u.date_joined,'OF') || ':00' ELSE to_char(u.date_joined,'OF') END,
'display_name',u.display_name,
'email',u.email,
'first_name',u.first_name,
'last_name',u.last_name,
'is_active',u.is_active,
'is_bot',u.is_bot,
'is_email_verified',u.is_email_verified,
'user_timezone',u.user_timezone,
'username',u.username,
'is_password_autoset',u.is_password_autoset,
'last_login_medium',u.last_login_medium,
'last_login_time',CASE WHEN u.last_login_time IS NULL THEN NULL ELSE to_char(u.last_login_time,'YYYY-MM-DD"T"HH24:MI:SS') || CASE WHEN to_char(u.last_login_time,'US') <> '000000' THEN '.' || to_char(u.last_login_time,'US') ELSE '' END || CASE WHEN to_char(u.last_login_time,'OF') IN ('+00','-00') THEN 'Z' WHEN length(to_char(u.last_login_time,'OF'))=3 THEN to_char(u.last_login_time,'OF') || ':00' ELSE to_char(u.last_login_time,'OF') END END)`

const profileJSON = `json_build_object(
'id',p.id,
'created_at',p.created_at,
'updated_at',p.updated_at,
'theme',p.theme,
'is_app_rail_docked',p.is_app_rail_docked,
'is_tour_completed',p.is_tour_completed,
'onboarding_step',p.onboarding_step,
'use_case',p.use_case,
'role',p.role,
'is_onboarded',p.is_onboarded,
'last_workspace_id',p.last_workspace_id,
'billing_address_country',p.billing_address_country,
'billing_address',p.billing_address,
'has_billing_address',p.has_billing_address,
'company_name',p.company_name,
'notification_view_mode',p.notification_view_mode,
'is_smooth_cursor_enabled',p.is_smooth_cursor_enabled,
'is_mobile_onboarded',p.is_mobile_onboarded,
'mobile_onboarding_step',p.mobile_onboarding_step,
'mobile_timezone_auto_set',p.mobile_timezone_auto_set,
'language',p.language,
'start_of_the_week',p.start_of_the_week,
'goals',p.goals,
'background_color',p.background_color,
'is_navigation_tour_completed',p.is_navigation_tour_completed,
'has_marketing_email_consent',p.has_marketing_email_consent,
'is_subscribed_to_changelog',p.is_subscribed_to_changelog,
'product_tour',p.product_tour,
'user',p.user_id)`

func (r *Repository) SessionUser(ctx context.Context, userID uuid.UUID) (json.RawMessage, error) {
	var body []byte
	var omitAvatarURL bool
	var omitCoverImageURL bool
	err := r.db.QueryRow(ctx, `WITH requested_timezone AS MATERIALIZED (
SELECT COALESCE((SELECT user_timezone FROM users WHERE id=$1), 'UTC') AS name
), activated_timezone AS MATERIALIZED (
SELECT set_config('TimeZone', requested_timezone.name, true) FROM requested_timezone
)
SELECT `+sessionUserJSON+`,
  COALESCE(avatar_asset.entity_type IN ('ISSUE_ATTACHMENT','ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') AND avatar_asset.workspace_id IS NULL, false),
  COALESCE(cover_asset.entity_type IN ('ISSUE_ATTACHMENT','ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') AND cover_asset.workspace_id IS NULL, false)
FROM users u
CROSS JOIN activated_timezone
LEFT JOIN file_assets avatar_asset ON avatar_asset.id=u.avatar_asset_id
LEFT JOIN workspaces avatar_workspace ON avatar_workspace.id=avatar_asset.workspace_id
LEFT JOIN file_assets cover_asset ON cover_asset.id=u.cover_image_asset_id
LEFT JOIN workspaces cover_workspace ON cover_workspace.id=cover_asset.workspace_id
WHERE u.id=$1 AND u.is_active=true`, userID).Scan(&body, &omitAvatarURL, &omitCoverImageURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err == nil && (omitAvatarURL || omitCoverImageURL) {
		omitted := make(map[string]bool, 2)
		omitted["avatar_url"] = omitAvatarURL
		omitted["cover_image_url"] = omitCoverImageURL
		body, err = omitTopLevelJSONFields(body, omitted)
	}
	return body, err
}

func omitTopLevelJSONFields(body []byte, omitted map[string]bool) ([]byte, error) {
	skipSpace := func(index int) int {
		for index < len(body) && (body[index] == ' ' || body[index] == '\t' || body[index] == '\r' || body[index] == '\n') {
			index++
		}
		return index
	}
	index := skipSpace(0)
	if index >= len(body) || body[index] != '{' {
		return nil, fmt.Errorf("decode session user JSON object")
	}
	index++
	pairs := make([][]byte, 0, 18)
	for {
		index = skipSpace(index)
		if index >= len(body) {
			return nil, fmt.Errorf("decode session user JSON object")
		}
		if body[index] == '}' {
			index = skipSpace(index + 1)
			if index != len(body) {
				return nil, fmt.Errorf("decode session user JSON object")
			}
			break
		}
		keyStart := index
		keyEnd, ok := jsonStringTokenEnd(body, index)
		if !ok {
			return nil, fmt.Errorf("decode session user JSON key")
		}
		var key string
		if err := json.Unmarshal(body[keyStart:keyEnd], &key); err != nil {
			return nil, fmt.Errorf("decode session user JSON key: %w", err)
		}
		index = skipSpace(keyEnd)
		if index >= len(body) || body[index] != ':' {
			return nil, fmt.Errorf("decode session user JSON field %q", key)
		}
		valueEnd, delimiter, err := topLevelJSONValueEnd(body, skipSpace(index+1))
		if err != nil {
			return nil, fmt.Errorf("decode session user JSON field %q: %w", key, err)
		}
		pairEnd := valueEnd
		for pairEnd > keyStart && (body[pairEnd-1] == ' ' || body[pairEnd-1] == '\t' || body[pairEnd-1] == '\r' || body[pairEnd-1] == '\n') {
			pairEnd--
		}
		if !omitted[key] {
			pairs = append(pairs, body[keyStart:pairEnd])
		}
		index = delimiter
		if body[index] == ',' {
			index++
			continue
		}
		if body[index] != '}' {
			return nil, fmt.Errorf("decode session user JSON object")
		}
	}
	var result bytes.Buffer
	result.WriteByte('{')
	for pairIndex, pair := range pairs {
		if pairIndex > 0 {
			result.WriteByte(',')
		}
		result.Write(pair)
	}
	result.WriteByte('}')
	return result.Bytes(), nil
}

func jsonStringTokenEnd(value []byte, start int) (int, bool) {
	if start >= len(value) || value[start] != '"' {
		return 0, false
	}
	escaped := false
	for index := start + 1; index < len(value); index++ {
		if escaped {
			escaped = false
			continue
		}
		if value[index] == '\\' {
			escaped = true
			continue
		}
		if value[index] == '"' {
			return index + 1, true
		}
	}
	return 0, false
}

func topLevelJSONValueEnd(body []byte, start int) (int, int, error) {
	depth := 0
	for index := start; index < len(body); index++ {
		switch body[index] {
		case '"':
			end, ok := jsonStringTokenEnd(body, index)
			if !ok {
				return 0, 0, fmt.Errorf("unterminated string")
			}
			index = end - 1
		case '{', '[':
			depth++
		case ']':
			if depth == 0 {
				return 0, 0, fmt.Errorf("unexpected array terminator")
			}
			depth--
		case '}':
			if depth == 0 {
				return index, index, nil
			}
			depth--
		case ',':
			if depth == 0 {
				return index, index, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("unterminated value")
}

func (r *Repository) Profile(ctx context.Context, userID uuid.UUID) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT `+profileJSON+` FROM profiles p WHERE p.user_id=$1`, userID).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}

func (r *Repository) UpdateProfile(ctx context.Context, userID uuid.UUID, patch json.RawMessage) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `WITH input AS (SELECT $2::jsonb AS body)
UPDATE profiles p SET
theme=CASE WHEN input.body ? 'theme' THEN input.body->'theme' ELSE p.theme END,
is_app_rail_docked=CASE WHEN input.body ? 'is_app_rail_docked' THEN (input.body->>'is_app_rail_docked')::boolean ELSE p.is_app_rail_docked END,
is_tour_completed=CASE WHEN input.body ? 'is_tour_completed' THEN (input.body->>'is_tour_completed')::boolean ELSE p.is_tour_completed END,
onboarding_step=CASE WHEN input.body ? 'onboarding_step' THEN input.body->'onboarding_step' ELSE p.onboarding_step END,
use_case=CASE WHEN input.body ? 'use_case' THEN input.body->>'use_case' ELSE p.use_case END,
role=CASE WHEN input.body ? 'role' THEN input.body->>'role' ELSE p.role END,
is_onboarded=CASE WHEN input.body ? 'is_onboarded' THEN (input.body->>'is_onboarded')::boolean ELSE p.is_onboarded END,
last_workspace_id=CASE WHEN input.body ? 'last_workspace_id' THEN (input.body->>'last_workspace_id')::uuid ELSE p.last_workspace_id END,
billing_address_country=CASE WHEN input.body ? 'billing_address_country' THEN input.body->>'billing_address_country' ELSE p.billing_address_country END,
billing_address=CASE WHEN input.body ? 'billing_address' THEN CASE WHEN input.body->'billing_address'='null'::jsonb THEN NULL ELSE input.body->'billing_address' END ELSE p.billing_address END,
has_billing_address=CASE WHEN input.body ? 'has_billing_address' THEN (input.body->>'has_billing_address')::boolean ELSE p.has_billing_address END,
company_name=CASE WHEN input.body ? 'company_name' THEN input.body->>'company_name' ELSE p.company_name END,
notification_view_mode=CASE WHEN input.body ? 'notification_view_mode' THEN input.body->>'notification_view_mode' ELSE p.notification_view_mode END,
is_smooth_cursor_enabled=CASE WHEN input.body ? 'is_smooth_cursor_enabled' THEN (input.body->>'is_smooth_cursor_enabled')::boolean ELSE p.is_smooth_cursor_enabled END,
is_mobile_onboarded=CASE WHEN input.body ? 'is_mobile_onboarded' THEN (input.body->>'is_mobile_onboarded')::boolean ELSE p.is_mobile_onboarded END,
mobile_onboarding_step=CASE WHEN input.body ? 'mobile_onboarding_step' THEN input.body->'mobile_onboarding_step' ELSE p.mobile_onboarding_step END,
mobile_timezone_auto_set=CASE WHEN input.body ? 'mobile_timezone_auto_set' THEN (input.body->>'mobile_timezone_auto_set')::boolean ELSE p.mobile_timezone_auto_set END,
language=CASE WHEN input.body ? 'language' THEN input.body->>'language' ELSE p.language END,
start_of_the_week=CASE WHEN input.body ? 'start_of_the_week' THEN (input.body->>'start_of_the_week')::smallint ELSE p.start_of_the_week END,
goals=CASE WHEN input.body ? 'goals' THEN input.body->'goals' ELSE p.goals END,
background_color=CASE WHEN input.body ? 'background_color' THEN input.body->>'background_color' ELSE p.background_color END,
is_navigation_tour_completed=CASE WHEN input.body ? 'is_navigation_tour_completed' THEN (input.body->>'is_navigation_tour_completed')::boolean ELSE p.is_navigation_tour_completed END,
has_marketing_email_consent=CASE WHEN input.body ? 'has_marketing_email_consent' THEN (input.body->>'has_marketing_email_consent')::boolean ELSE p.has_marketing_email_consent END,
is_subscribed_to_changelog=CASE WHEN input.body ? 'is_subscribed_to_changelog' THEN (input.body->>'is_subscribed_to_changelog')::boolean ELSE p.is_subscribed_to_changelog END,
product_tour=CASE WHEN input.body ? 'product_tour' THEN input.body->'product_tour' ELSE p.product_tour END,
updated_at=NOW()
FROM input WHERE p.user_id=$1 RETURNING `+profileJSON, userID, patch).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}

// APIV1Identity projects the API UserLiteSerializer without creating a profile.
func (r *Repository) APIV1Identity(ctx context.Context, userID uuid.UUID) (json.RawMessage, error) {
	var body []byte
	var omitAvatar bool
	err := r.db.QueryRow(ctx, `SELECT json_build_object(
 'id',u.id,'first_name',u.first_name,'last_name',u.last_name,'email',u.email,
 'avatar',u.avatar,'avatar_url',`+avatarURLSQL+`,'display_name',u.display_name),
 COALESCE(avatar_asset.entity_type IN ('ISSUE_ATTACHMENT','ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') AND avatar_asset.workspace_id IS NULL,false)
 FROM users u
 LEFT JOIN file_assets avatar_asset ON avatar_asset.id=u.avatar_asset_id
 LEFT JOIN workspaces avatar_workspace ON avatar_workspace.id=avatar_asset.workspace_id
 WHERE u.id=$1 AND u.is_active=true`, userID).Scan(&body, &omitAvatar)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err == nil && omitAvatar {
		body, err = omitTopLevelJSONFields(body, map[string]bool{"avatar_url": true})
	}
	return body, err
}
