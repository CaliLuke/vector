package instancemanagement

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/instanceadminsession"
	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/CaliLuke/go-plane/internal/workspacevalidation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConfigCipher interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

type Repository struct {
	db     *pgxpool.Pool
	admins *instanceadminsession.Repository
	cipher ConfigCipher
}

type WorkspaceValidationError struct {
	Field   string
	Message string
	Fields  map[string][]string
}

func (err *WorkspaceValidationError) Error() string {
	if err.Message != "" {
		return err.Message
	}
	return "workspace serializer validation failed"
}

func addWorkspaceFieldError(fieldErrors map[string][]string, field, message string) {
	fieldErrors[field] = append(fieldErrors[field], message)
}

func addWorkspaceValidationError(fieldErrors map[string][]string, err error) {
	var validation *WorkspaceValidationError
	if errors.As(err, &validation) {
		if validation.Field != "" {
			addWorkspaceFieldError(fieldErrors, validation.Field, validation.Message)
		}
		for field, messages := range validation.Fields {
			fieldErrors[field] = append(fieldErrors[field], messages...)
		}
	}
}

type WorkspacePaginationError struct{ Message string }

func (err *WorkspacePaginationError) Error() string { return err.Message }

func NewRepository(db *pgxpool.Pool, cipher ConfigCipher) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("instance management database is required")
	}
	if cipher == nil {
		return nil, fmt.Errorf("instance management config cipher is required")
	}
	admins, err := instanceadminsession.NewRepository(db)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db, admins: admins, cipher: cipher}, nil
}

func (r *Repository) IsAdmin(ctx context.Context, user uuid.UUID) (bool, error) {
	return r.admins.IsAdmin(ctx, user)
}

const instanceJSON = `json_build_object(
 'id', i.id, 'instance_name', i.instance_name, 'whitelist_emails', i.whitelist_emails,
 'instance_id', i.instance_id, 'current_version', i.current_version, 'latest_version', i.latest_version,
 'edition', i.edition, 'domain', i.domain, 'last_checked_at', i.last_checked_at,
 'namespace', i.namespace, 'is_telemetry_enabled', i.is_telemetry_enabled,
 'is_support_required', i.is_support_required, 'is_setup_done', i.is_setup_done,
 'is_signup_screen_visited', i.is_signup_screen_visited, 'is_verified', i.is_verified,
 'is_test', i.is_test, 'is_current_version_deprecated', i.is_current_version_deprecated,
 'created_at', i.created_at, 'updated_at', i.updated_at, 'deleted_at', i.deleted_at,
 'created_by', i.created_by_id, 'updated_by', i.updated_by_id)`

func (r *Repository) Root(ctx context.Context) (json.RawMessage, error) {
	var instance []byte
	var workspaceExists bool
	err := r.db.QueryRow(ctx, `SELECT `+instanceJSON+`, EXISTS(SELECT 1 FROM workspaces WHERE deleted_at IS NULL) FROM instances i WHERE i.deleted_at IS NULL ORDER BY i.created_at DESC LIMIT 1`).Scan(&instance, &workspaceExists)
	if errors.Is(err, pgx.ErrNoRows) {
		return json.RawMessage(`{"is_activated":false,"is_setup_done":false}`), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read instance management root: %w", err)
	}
	var projected map[string]any
	if err := json.Unmarshal(instance, &projected); err != nil {
		return nil, err
	}
	projected["workspaces_exist"] = workspaceExists
	config, err := r.rootConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{"instance": projected, "config": config})
	return body, err
}

func (r *Repository) rootConfiguration(ctx context.Context) (map[string]any, error) {
	keys := []string{"ENABLE_SIGNUP", "DISABLE_WORKSPACE_CREATION", "IS_GOOGLE_ENABLED", "IS_GITHUB_ENABLED", "GITHUB_APP_NAME", "IS_GITLAB_ENABLED", "IS_GITEA_ENABLED", "EMAIL_HOST", "ENABLE_MAGIC_LINK_LOGIN", "ENABLE_EMAIL_PASSWORD", "SLACK_CLIENT_ID", "POSTHOG_API_KEY", "POSTHOG_HOST", "UNSPLASH_ACCESS_KEY", "LLM_API_KEY"}
	values, err := r.configurationValues(ctx, keys)
	if err != nil {
		return nil, err
	}
	defaults := map[string]string{
		"ENABLE_SIGNUP": "0", "DISABLE_WORKSPACE_CREATION": "0", "IS_GOOGLE_ENABLED": "0", "IS_GITHUB_ENABLED": "0",
		"GITHUB_APP_NAME": "", "IS_GITLAB_ENABLED": "0", "IS_GITEA_ENABLED": "0", "EMAIL_HOST": "",
		"ENABLE_MAGIC_LINK_LOGIN": "1", "ENABLE_EMAIL_PASSWORD": "1", "SLACK_CLIENT_ID": "", "POSTHOG_API_KEY": "",
		"POSTHOG_HOST": "", "UNSPLASH_ACCESS_KEY": "", "LLM_API_KEY": "",
	}
	for key, value := range defaults {
		if _, ok := values[key]; !ok {
			values[key] = os.Getenv(key)
			if values[key] == "" {
				values[key] = value
			}
		}
	}
	boolean := func(key string) bool { return values[key] == "1" }
	fileSize, err := strconv.ParseFloat(envOr("FILE_SIZE_LIMIT", "5242880"), 64)
	if err != nil {
		fileSize = 5242880
	}
	return map[string]any{
		"enable_signup": boolean("ENABLE_SIGNUP"), "is_workspace_creation_disabled": boolean("DISABLE_WORKSPACE_CREATION"),
		"is_google_enabled": boolean("IS_GOOGLE_ENABLED"), "is_github_enabled": boolean("IS_GITHUB_ENABLED"),
		"is_gitlab_enabled": boolean("IS_GITLAB_ENABLED"), "is_gitea_enabled": boolean("IS_GITEA_ENABLED"),
		"is_magic_login_enabled": boolean("ENABLE_MAGIC_LINK_LOGIN"), "is_email_password_enabled": boolean("ENABLE_EMAIL_PASSWORD"),
		"github_app_name": values["GITHUB_APP_NAME"], "slack_client_id": nullableString(values["SLACK_CLIENT_ID"]),
		"posthog_api_key": nullableString(values["POSTHOG_API_KEY"]), "posthog_host": nullableString(values["POSTHOG_HOST"]),
		"has_unsplash_configured": values["UNSPLASH_ACCESS_KEY"] != "", "has_llm_configured": values["LLM_API_KEY"] != "",
		"file_size_limit": fileSize, "is_smtp_configured": values["EMAIL_HOST"] != "",
		"admin_base_url": envOr("ADMIN_BASE_URL", ""), "space_base_url": envOr("SPACE_BASE_URL", ""),
		"app_base_url": envOr("APP_BASE_URL", ""), "instance_changelog_url": envOr("INSTANCE_CHANGELOG_URL", ""),
		"is_self_managed": envOr("IS_SELF_MANAGED", "false") == "true",
	}, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *Repository) PatchRoot(ctx context.Context, actor uuid.UUID, fields map[string]json.RawMessage) (json.RawMessage, error) {
	var name, whitelist *string
	var telemetry, support, signup *bool
	if value, ok := fields["instance_name"]; ok {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			return nil, err
		}
		name = &parsed
	}
	if value, ok := fields["whitelist_emails"]; ok {
		if string(value) == "null" {
			whitelist = nil
		} else {
			var parsed string
			if err := json.Unmarshal(value, &parsed); err != nil {
				return nil, err
			}
			whitelist = &parsed
		}
	}
	if value, ok := fields["is_telemetry_enabled"]; ok {
		var parsed bool
		if err := json.Unmarshal(value, &parsed); err != nil {
			return nil, err
		}
		telemetry = &parsed
	}
	if value, ok := fields["is_support_required"]; ok {
		var parsed bool
		if err := json.Unmarshal(value, &parsed); err != nil {
			return nil, err
		}
		support = &parsed
	}
	if value, ok := fields["is_signup_screen_visited"]; ok {
		var parsed bool
		if err := json.Unmarshal(value, &parsed); err != nil {
			return nil, err
		}
		signup = &parsed
	}
	var result []byte
	err := r.db.QueryRow(ctx, `UPDATE instances AS i SET
 instance_name=COALESCE(@instance_name, i.instance_name),
 whitelist_emails=CASE WHEN @has_whitelist THEN @whitelist::text ELSE i.whitelist_emails END,
 is_telemetry_enabled=COALESCE(@telemetry, i.is_telemetry_enabled),
 is_support_required=COALESCE(@support, i.is_support_required),
 is_signup_screen_visited=COALESCE(@signup, i.is_signup_screen_visited),
 updated_by_id=@actor, updated_at=NOW()
 WHERE i.id=(SELECT id FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1)
 RETURNING `+instanceJSON,
		pgx.StrictNamedArgs{"instance_name": name, "has_whitelist": has(fields, "whitelist_emails"), "whitelist": whitelist,
			"telemetry": telemetry, "support": support, "signup": signup, "actor": actor}).Scan(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) ListAdmins(ctx context.Context) (json.RawMessage, error) {
	var result []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(item ORDER BY created_at DESC), '[]'::json) FROM (
SELECT json_build_object('id', a.id, 'created_at', a.created_at, 'updated_at', a.updated_at, 'deleted_at', a.deleted_at,
 'created_by', a.created_by_id, 'updated_by', a.updated_by_id, 'instance', a.instance_id, 'user', a.user_id,
 'role', a.role, 'is_verified', a.is_verified,
 'user_detail', json_build_object('id', u.id, 'avatar', u.avatar, 'avatar_url', NULLIF(u.avatar,''),
 'display_name', u.display_name, 'email', u.email, 'first_name', u.first_name, 'last_name', u.last_name,
 'is_bot', u.is_bot, 'last_login_medium', u.last_login_medium)) AS item, a.created_at
FROM instance_admins a LEFT JOIN users u ON u.id=a.user_id
WHERE a.deleted_at IS NULL AND a.instance_id=(SELECT id FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1)
) admins`).Scan(&result)
	return result, err
}

func (r *Repository) CreateAdmin(ctx context.Context, actor uuid.UUID, body map[string]json.RawMessage) (json.RawMessage, error) {
	email := readString(body["email"])
	if email == "" {
		return nil, errors.New("email required")
	}
	role := 20
	if raw := body["role"]; len(raw) != 0 {
		_ = json.Unmarshal(raw, &role)
	}
	var result []byte
	err := r.db.QueryRow(ctx, `INSERT INTO instance_admins(id, instance_id, user_id, role, is_verified, created_by_id, created_at, updated_at)
SELECT gen_random_uuid(), i.id, u.id, @role, false, @actor, NOW(), NOW()
FROM instances i CROSS JOIN users u
WHERE i.id=(SELECT id FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1) AND u.email=@email
RETURNING json_build_object('id', id, 'created_at', created_at, 'updated_at', updated_at, 'deleted_at', deleted_at,
 'created_by', created_by_id, 'updated_by', updated_by_id, 'instance', instance_id, 'user', user_id,
 'role', role, 'is_verified', is_verified,
 'user_detail', (SELECT json_build_object('id',users.id,'avatar',users.avatar,'avatar_url',NULLIF(users.avatar,''),'display_name',users.display_name,'email',users.email,'first_name',users.first_name,'last_name',users.last_name,'is_bot',users.is_bot,'last_login_medium',users.last_login_medium) FROM users WHERE users.id=user_id))`, pgx.StrictNamedArgs{"role": role, "actor": actor, "email": email}).Scan(&result)
	return result, err
}

func (r *Repository) DeleteAdmin(ctx context.Context, admin uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE instance_admins SET deleted_at=NOW(), updated_at=NOW() WHERE id=@id AND deleted_at IS NULL AND instance_id=(SELECT id FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1)`, pgx.StrictNamedArgs{"id": admin})
	return err
}

func (r *Repository) Configurations(ctx context.Context) (json.RawMessage, error) {
	var configs []struct {
		ID        uuid.UUID `json:"id"`
		Key       string    `json:"key"`
		Value     *string   `json:"value"`
		Category  string    `json:"category"`
		Encrypted bool      `json:"is_encrypted"`
		Created   any       `json:"created_at"`
		Updated   any       `json:"updated_at"`
		Deleted   any       `json:"deleted_at"`
		CreatedBy any       `json:"created_by"`
		UpdatedBy any       `json:"updated_by"`
	}
	rows, err := r.db.Query(ctx, `SELECT id,key,value,category,is_encrypted,created_at,updated_at,deleted_at,created_by_id,updated_by_id FROM instance_configurations WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c struct {
			ID        uuid.UUID `json:"id"`
			Key       string    `json:"key"`
			Value     *string   `json:"value"`
			Category  string    `json:"category"`
			Encrypted bool      `json:"is_encrypted"`
			Created   any       `json:"created_at"`
			Updated   any       `json:"updated_at"`
			Deleted   any       `json:"deleted_at"`
			CreatedBy any       `json:"created_by"`
			UpdatedBy any       `json:"updated_by"`
		}
		if err := rows.Scan(&c.ID, &c.Key, &c.Value, &c.Category, &c.Encrypted, &c.Created, &c.Updated, &c.Deleted, &c.CreatedBy, &c.UpdatedBy); err != nil {
			return nil, err
		}
		if c.Encrypted && c.Value != nil {
			plaintext, err := r.cipher.Decrypt(*c.Value)
			if err != nil {
				c.Value = strPtr("")
			} else {
				c.Value = &plaintext
			}
		}
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(configs)
}

func (r *Repository) configurationValues(ctx context.Context, keys []string) (map[string]string, error) {
	if os.Getenv("SKIP_ENV_VAR") == "0" || strings.EqualFold(os.Getenv("SKIP_ENV_VAR"), "false") {
		return map[string]string{}, nil
	}
	rows, err := r.db.Query(ctx, `SELECT key, value, is_encrypted FROM instance_configurations WHERE deleted_at IS NULL AND key=ANY($1::text[])`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make(map[string]string)
	for rows.Next() {
		var key, value string
		var encrypted bool
		if err := rows.Scan(&key, &value, &encrypted); err != nil {
			return nil, err
		}
		if encrypted {
			plain, err := r.cipher.Decrypt(value)
			if err != nil {
				value = ""
			} else {
				value = plain
			}
		}
		values[key] = value
	}
	return values, rows.Err()
}

func (r *Repository) SMTPConfiguration(ctx context.Context) (instancemail.SMTPConfig, string, error) {
	keys := []string{"EMAIL_HOST", "EMAIL_HOST_USER", "EMAIL_HOST_PASSWORD", "EMAIL_PORT", "EMAIL_USE_TLS", "EMAIL_USE_SSL", "EMAIL_FROM"}
	values, err := r.configurationValues(ctx, keys)
	if err != nil {
		return instancemail.SMTPConfig{}, "", err
	}
	defaults := map[string]string{"EMAIL_PORT": "587", "EMAIL_USE_TLS": "1", "EMAIL_USE_SSL": "0", "EMAIL_FROM": "Vector <noreply@localhost>"}
	for _, key := range keys {
		if _, ok := values[key]; !ok {
			values[key] = os.Getenv(key)
			if values[key] == "" {
				values[key] = defaults[key]
			}
		}
	}
	port, err := strconv.Atoi(values["EMAIL_PORT"])
	if err != nil {
		return instancemail.SMTPConfig{}, "", err
	}
	return instancemail.SMTPConfig{Host: values["EMAIL_HOST"], Port: port, Username: values["EMAIL_HOST_USER"], Password: values["EMAIL_HOST_PASSWORD"], UseTLS: values["EMAIL_USE_TLS"] == "1", UseSSL: values["EMAIL_USE_SSL"] == "1"}, values["EMAIL_FROM"], nil
}

func (r *Repository) PatchConfigurations(ctx context.Context, fields map[string]json.RawMessage) (json.RawMessage, error) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return json.RawMessage(`[]`), nil
	}
	rows, err := r.db.Query(ctx, `SELECT id,key,value,category,is_encrypted FROM instance_configurations WHERE deleted_at IS NULL AND key=ANY($1::text[])`, keys)
	if err != nil {
		return nil, err
	}
	type config struct {
		id        uuid.UUID
		key       string
		value     *string
		category  string
		encrypted bool
	}
	var configs []config
	for rows.Next() {
		var c config
		if err := rows.Scan(&c.id, &c.key, &c.value, &c.category, &c.encrypted); err != nil {
			rows.Close()
			return nil, err
		}
		configs = append(configs, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	for _, c := range configs {
		value := pythonString(fields[c.key])
		if c.encrypted {
			encrypted, err := r.cipher.Encrypt(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			value = encrypted
		} else {
			value = strings.TrimSpace(value)
		}
		if _, err := tx.Exec(ctx, `UPDATE instance_configurations SET value=@value WHERE id=@id`, pgx.StrictNamedArgs{"value": value, "id": c.id}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.configurationsForKeys(ctx, keys)
}

func (r *Repository) configurationsForKeys(ctx context.Context, keys []string) (json.RawMessage, error) {
	rows, err := r.db.Query(ctx, `SELECT id,key,value,category,is_encrypted,created_at,updated_at,deleted_at,created_by_id,updated_by_id FROM instance_configurations WHERE deleted_at IS NULL AND key=ANY($1::text[]) ORDER BY created_at DESC`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []struct {
		ID        uuid.UUID `json:"id"`
		Key       string    `json:"key"`
		Value     *string   `json:"value"`
		Category  string    `json:"category"`
		Encrypted bool      `json:"is_encrypted"`
		Created   any       `json:"created_at"`
		Updated   any       `json:"updated_at"`
		Deleted   any       `json:"deleted_at"`
		CreatedBy any       `json:"created_by"`
		UpdatedBy any       `json:"updated_by"`
	}
	for rows.Next() {
		var c struct {
			ID        uuid.UUID `json:"id"`
			Key       string    `json:"key"`
			Value     *string   `json:"value"`
			Category  string    `json:"category"`
			Encrypted bool      `json:"is_encrypted"`
			Created   any       `json:"created_at"`
			Updated   any       `json:"updated_at"`
			Deleted   any       `json:"deleted_at"`
			CreatedBy any       `json:"created_by"`
			UpdatedBy any       `json:"updated_by"`
		}
		if err := rows.Scan(&c.ID, &c.Key, &c.Value, &c.Category, &c.Encrypted, &c.Created, &c.Updated, &c.Deleted, &c.CreatedBy, &c.UpdatedBy); err != nil {
			return nil, err
		}
		if c.Encrypted && c.Value != nil {
			value, err := r.cipher.Decrypt(*c.Value)
			if err != nil {
				value = ""
			}
			c.Value = &value
		}
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(configs)
}

func (r *Repository) DisableEmail(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `UPDATE instance_configurations SET value=CASE WHEN key='ENABLE_SMTP' THEN '0' ELSE '' END WHERE deleted_at IS NULL AND key=ANY($1::text[])`, []string{"EMAIL_HOST", "EMAIL_HOST_USER", "EMAIL_HOST_PASSWORD", "ENABLE_SMTP", "EMAIL_PORT", "EMAIL_FROM"})
	return err
}

func (r *Repository) SignupVisited(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `UPDATE instances SET is_signup_screen_visited=true,updated_at=NOW() WHERE id=(SELECT id FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1)`)
	return err
}

func (r *Repository) Workspaces(ctx context.Context, search, perPage, cursor string) (json.RawMessage, error) {
	limit, pageIndex, offset, err := workspacePageWindow(perPage, cursor)
	if err != nil {
		return nil, err
	}
	pattern := "%" + issuequery.EscapeLikePattern(search) + "%"
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM workspaces WHERE deleted_at IS NULL AND (@search='' OR name ILIKE @pattern ESCAPE '\')`, pgx.StrictNamedArgs{"search": search, "pattern": pattern}).Scan(&count); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT json_build_object('id',w.id,'created_at',w.created_at,'updated_at',w.updated_at,'deleted_at',w.deleted_at,'name',w.name,'logo',w.logo,'logo_asset',w.logo_asset_id,'logo_url',CASE WHEN w.logo_asset_id IS NULL THEN NULLIF(w.logo,'') WHEN f.entity_type IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/'||w.logo_asset_id::text||'/' WHEN f.entity_type='ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/'||asset_workspace.slug||'/projects/'||COALESCE(f.project_id::text,'None')||'/issues/'||COALESCE(f.issue_id::text,'None')||'/attachments/'||f.id::text||'/' WHEN f.entity_type IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/'||asset_workspace.slug||'/projects/'||COALESCE(f.project_id::text,'None')||'/'||f.id::text||'/' ELSE NULL END,'owner',json_build_object('id',u.id,'email',u.email,'first_name',u.first_name,'last_name',u.last_name),'slug',w.slug,'organization_size',w.organization_size,'timezone',w.timezone,'background_color',w.background_color,'created_by',w.created_by_id,'updated_by',w.updated_by_id,'total_projects',(SELECT count(*) FROM projects p WHERE p.workspace_id=w.id AND p.deleted_at IS NULL),'total_members',(SELECT count(*) FROM workspace_members wm JOIN users mu ON mu.id=wm.member_id WHERE wm.workspace_id=w.id AND wm.deleted_at IS NULL AND wm.is_active AND NOT mu.is_bot)) FROM workspaces w JOIN users u ON u.id=w.owner_id LEFT JOIN file_assets f ON f.id=w.logo_asset_id LEFT JOIN workspaces asset_workspace ON asset_workspace.id=f.workspace_id WHERE w.deleted_at IS NULL AND (@search='' OR w.name ILIKE @pattern ESCAPE '\') ORDER BY w.created_at DESC LIMIT @limit OFFSET @offset`, pgx.StrictNamedArgs{"search": search, "pattern": pattern, "limit": limit, "offset": offset})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]json.RawMessage, 0)
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		results = append(results, body)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if limit == 0 {
		return nil, errors.New("division by zero")
	}
	response := djangoapi.Envelope(rawMessagesAsValues(results), int(count), int(limit), int(pageIndex))
	return json.Marshal(response)
}

func (r *Repository) CreateWorkspace(ctx context.Context, actor uuid.UUID, fields map[string]json.RawMessage, requestTimezone string) (json.RawMessage, error) {
	name, namePresent, nameTruthy, nameErr := workspaceCharField(fields["name"])
	slug, slugPresent, slugTruthy, slugErr := workspaceCharField(fields["slug"])
	if !namePresent || !slugPresent || !nameTruthy || !slugTruthy {
		return nil, &WorkspaceValidationError{Message: "Both name and slug are required"}
	}
	if workspaceRawFieldLength(fields["name"]) > 80 || workspaceRawFieldLength(fields["slug"]) > 48 {
		return nil, &WorkspaceValidationError{Message: "The maximum length for name is 80 and for slug is 48"}
	}
	if nameTruthy && !workspaceRawString(fields["name"]) {
		nameErr = "Not a valid string."
	}
	if slugTruthy && !workspaceRawString(fields["slug"]) {
		slugErr = "Not a valid string."
	}
	fieldErrors := map[string][]string{}
	if nameErr != "" {
		addWorkspaceFieldError(fieldErrors, "name", nameErr)
	} else if workspacevalidation.ContainsURL(name) {
		addWorkspaceFieldError(fieldErrors, "name", "Name must not contain URLs")
	} else if !hasAlphaNumeric(name) {
		addWorkspaceFieldError(fieldErrors, "name", "Name must contain at least one letter or number")
	}
	if slugErr != "" {
		addWorkspaceFieldError(fieldErrors, "slug", slugErr)
	} else if !workspacevalidation.ValidSlug(slug) || workspacevalidation.RestrictedSlug(slug) {
		message := `Enter a valid "slug" consisting of letters, numbers, underscores or hyphens.`
		if workspacevalidation.RestrictedSlug(slug) {
			message = "Slug is not valid"
		}
		addWorkspaceFieldError(fieldErrors, "slug", message)
	} else {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE lower(slug)=lower(@slug) AND deleted_at IS NULL)`, pgx.StrictNamedArgs{"slug": slug}).Scan(&exists); err != nil {
			return nil, err
		}
		if exists {
			addWorkspaceFieldError(fieldErrors, "slug", "Slug is already in use")
		}
	}
	logo, logoErr := workspaceOptionalText(fields, "logo", true, true, 0)
	if logoErr != nil {
		addWorkspaceValidationError(fieldErrors, logoErr)
	}
	var assetID any
	if raw := fields["logo_asset"]; len(raw) > 0 && string(raw) != "null" {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			addWorkspaceFieldError(fieldErrors, "logo_asset", "Must be a valid UUID.")
		} else if id, err := uuid.Parse(s); err != nil {
			addWorkspaceFieldError(fieldErrors, "logo_asset", "Must be a valid UUID.")
		} else {
			var assetExists bool
			if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=@id AND deleted_at IS NULL)`, pgx.StrictNamedArgs{"id": id}).Scan(&assetExists); err != nil {
				return nil, err
			}
			if !assetExists {
				addWorkspaceFieldError(fieldErrors, "logo_asset", fmt.Sprintf("Invalid pk %q - object does not exist.", s))
			} else {
				assetID = id
			}
		}
	}
	org, orgErr := workspaceOptionalText(fields, "organization_size", true, true, 20)
	if orgErr != nil {
		addWorkspaceValidationError(fieldErrors, orgErr)
	}
	timezone := "UTC"
	if raw, present := fields["timezone"]; present {
		zone, _, _, fieldError := workspaceCharField(raw)
		if fieldError != "" {
			addWorkspaceFieldError(fieldErrors, "timezone", fieldError)
		} else if !workspaceprojectreads.ValidProjectTimezone(zone) {
			addWorkspaceFieldError(fieldErrors, "timezone", fmt.Sprintf("%q is not a valid choice.", zone))
		} else {
			timezone = zone
		}
	}
	backgroundColor, colorErr := workspaceBackgroundColor(fields)
	if colorErr != nil {
		var validation *WorkspaceValidationError
		if !errors.As(colorErr, &validation) {
			return nil, colorErr
		}
		addWorkspaceValidationError(fieldErrors, validation)
	}
	var deletedAt any
	if raw, present := fields["deleted_at"]; present && string(raw) != "null" {
		value, ok := workspaceDecodedField(raw)
		text, stringInput := value.(string)
		if !ok || !stringInput {
			addWorkspaceFieldError(fieldErrors, "deleted_at", "Datetime has wrong format.")
		} else {
			location := time.UTC
			if requestTimezone != "" {
				if loaded, err := time.LoadLocation(requestTimezone); err == nil {
					location = loaded
				}
			}
			parsed, err := pythoncompat.DjangoDateTime(text, location)
			if err != nil {
				addWorkspaceFieldError(fieldErrors, "deleted_at", "Datetime has wrong format.")
			} else {
				deletedAt = parsed
			}
		}
	}
	if len(fieldErrors) > 0 {
		return nil, &WorkspaceValidationError{Fields: fieldErrors}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var result []byte
	var workspaceID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO workspaces(id,created_at,updated_at,deleted_at,created_by_id,owner_id,name,slug,logo,logo_asset_id,organization_size,timezone,background_color) VALUES(gen_random_uuid(),NOW(),NOW(),@deleted_at,@actor,@actor,@name,@slug,@logo,@logo_asset,@organization_size,@timezone,@background_color) RETURNING id,json_build_object('id',id,'created_at',created_at,'updated_at',updated_at,'deleted_at',deleted_at,'name',name,'logo',logo,'logo_asset',logo_asset_id,'logo_url',CASE WHEN logo_asset_id IS NULL THEN NULLIF(logo,'') WHEN (SELECT entity_type FROM file_assets WHERE id=logo_asset_id) IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/'||logo_asset_id::text||'/' ELSE NULL END,'owner',(SELECT json_build_object('id',id,'email',email,'first_name',first_name,'last_name',last_name) FROM users WHERE id=owner_id),'slug',slug,'organization_size',organization_size,'timezone',timezone,'background_color',background_color,'created_by',created_by_id,'updated_by',updated_by_id)`, pgx.StrictNamedArgs{"actor": actor, "deleted_at": deletedAt, "name": name, "slug": slug, "logo": logo, "logo_asset": assetID, "organization_size": org, "timezone": timezone, "background_color": backgroundColor}).Scan(&workspaceID, &result)
	if err != nil {
		return nil, err
	}
	companyRole, err := workspaceCompanyRole(fields)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO workspace_members(id,workspace_id,member_id,role,view_props,default_props,issue_props,is_active,company_role,getting_started_checklist,tips,explored_features,created_at,updated_at,created_by_id) VALUES(gen_random_uuid(),@workspace,@actor,20,@view_props,@default_props,@issue_props,true,@company_role,'{}','{}','{}',NOW(),NOW(),@actor)`, pgx.StrictNamedArgs{"actor": actor, "workspace": workspaceID, "company_role": companyRole, "view_props": projectmembershipreads.WorkspaceDefaultProps, "default_props": projectmembershipreads.WorkspaceDefaultProps, "issue_props": projectmembershipreads.WorkspaceIssueProps})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// WorkspaceRole returns the active workspace role for a user, or zero when
// the user has no active membership. Workspace lifecycle actions use the same
// membership row that the Django permission class checks.
func (r *Repository) WorkspaceRole(ctx context.Context, actor uuid.UUID, slug string) (int, error) {
	var role int
	err := r.db.QueryRow(ctx, `SELECT COALESCE((
 SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id
 WHERE wm.member_id=@actor AND w.slug=@slug AND w.deleted_at IS NULL
   AND wm.is_active=true AND wm.deleted_at IS NULL
 ORDER BY wm.created_at DESC LIMIT 1),0)`, pgx.StrictNamedArgs{"actor": actor, "slug": slug}).Scan(&role)
	return role, err
}

func (r *Repository) WorkspaceCreationDisabled(ctx context.Context) (bool, error) {
	values, err := r.configurationValues(ctx, []string{"DISABLE_WORKSPACE_CREATION"})
	if err != nil {
		return false, err
	}
	value, ok := values["DISABLE_WORKSPACE_CREATION"]
	if !ok {
		value = os.Getenv("DISABLE_WORKSPACE_CREATION")
	}
	return value == "1", nil
}

// UpdateWorkspace applies the shared WorkSpaceSerializer fields to one
// workspace. PUT and PATCH use the same serializer update operation in Plane.
func (r *Repository) UpdateWorkspace(ctx context.Context, actor uuid.UUID, slug string, fields map[string]json.RawMessage) (json.RawMessage, error) {
	fieldErrors := map[string][]string{}
	var name, newSlug, logo, organizationSize, timezone, backgroundColor any
	set := map[string]bool{}
	if raw, ok := fields["name"]; ok {
		value, present, truthy, fieldError := workspaceCharField(raw)
		if fieldError != "" || !present || !truthy {
			addWorkspaceFieldError(fieldErrors, "name", "This field may not be blank.")
		} else if workspaceRawFieldLength(raw) > 80 {
			addWorkspaceFieldError(fieldErrors, "name", "Ensure this field has no more than 80 characters.")
		} else if !workspaceRawString(raw) {
			addWorkspaceFieldError(fieldErrors, "name", "Not a valid string.")
		} else if workspacevalidation.ContainsURL(value) {
			addWorkspaceFieldError(fieldErrors, "name", "Name must not contain URLs")
		} else if !hasAlphaNumeric(value) {
			addWorkspaceFieldError(fieldErrors, "name", "Name must contain at least one letter or number")
		} else {
			name, set["name"] = value, true
		}
	}
	if raw, ok := fields["slug"]; ok {
		value, present, truthy, fieldError := workspaceCharField(raw)
		if fieldError != "" || !present || !truthy {
			addWorkspaceFieldError(fieldErrors, "slug", "This field may not be blank.")
		} else if workspaceRawFieldLength(raw) > 48 {
			addWorkspaceFieldError(fieldErrors, "slug", "Ensure this field has no more than 48 characters.")
		} else if !workspaceRawString(raw) {
			addWorkspaceFieldError(fieldErrors, "slug", "Not a valid string.")
		} else if !workspacevalidation.ValidSlug(value) || workspacevalidation.RestrictedSlug(value) {
			message := "Slug can only contain letters, numbers, hyphens (-), and underscores (_)"
			if workspacevalidation.RestrictedSlug(value) {
				message = "Slug is not valid"
			}
			addWorkspaceFieldError(fieldErrors, "slug", message)
		} else {
			var exists bool
			if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE lower(slug)=lower(@new_slug) AND slug<>@slug AND deleted_at IS NULL)`, pgx.StrictNamedArgs{"new_slug": value, "slug": slug}).Scan(&exists); err != nil {
				return nil, err
			}
			if exists {
				addWorkspaceFieldError(fieldErrors, "slug", "Slug is already in use")
			} else {
				newSlug, set["slug"] = value, true
			}
		}
	}
	if _, ok := fields["logo"]; ok {
		value, err := workspaceOptionalText(fields, "logo", true, true, 0)
		if err != nil {
			addWorkspaceValidationError(fieldErrors, err)
		} else {
			logo, set["logo"] = value, true
		}
	}
	if _, ok := fields["organization_size"]; ok {
		value, err := workspaceOptionalText(fields, "organization_size", true, true, 20)
		if err != nil {
			addWorkspaceValidationError(fieldErrors, err)
		} else {
			organizationSize, set["organization_size"] = value, true
		}
	}
	if raw, ok := fields["timezone"]; ok {
		value, present, _, fieldError := workspaceCharField(raw)
		if fieldError != "" || !present || !workspaceprojectreads.ValidProjectTimezone(value) {
			addWorkspaceFieldError(fieldErrors, "timezone", fmt.Sprintf("%q is not a valid choice.", value))
		} else {
			timezone, set["timezone"] = value, true
		}
	}
	if _, ok := fields["background_color"]; ok {
		value, err := workspaceBackgroundColor(fields)
		if err != nil {
			addWorkspaceValidationError(fieldErrors, err)
		} else {
			backgroundColor, set["background_color"] = value, true
		}
	}
	if len(fieldErrors) != 0 {
		return nil, &WorkspaceValidationError{Fields: fieldErrors}
	}
	var result []byte
	err := r.db.QueryRow(ctx, `UPDATE workspaces w SET
 name=CASE WHEN @set_name THEN @name::text ELSE w.name END,
 slug=CASE WHEN @set_slug THEN @new_slug::text ELSE w.slug END,
 logo=CASE WHEN @set_logo THEN @logo::text ELSE w.logo END,
 organization_size=CASE WHEN @set_organization_size THEN @organization_size::text ELSE w.organization_size END,
 timezone=CASE WHEN @set_timezone THEN @timezone::text ELSE w.timezone END,
 background_color=CASE WHEN @set_background_color THEN @background_color::text ELSE w.background_color END,
 updated_by_id=@actor, updated_at=NOW()
 WHERE w.slug=@slug AND w.deleted_at IS NULL
   AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=@actor AND wm.role=20 AND wm.is_active AND wm.deleted_at IS NULL)
 RETURNING json_build_object('id',w.id,'created_at',w.created_at,'updated_at',w.updated_at,'deleted_at',w.deleted_at,'name',w.name,'logo',w.logo,'logo_asset',w.logo_asset_id,'logo_url',CASE WHEN w.logo_asset_id IS NULL THEN NULLIF(w.logo,'') WHEN (SELECT entity_type FROM file_assets WHERE id=w.logo_asset_id) IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/'||w.logo_asset_id::text||'/' ELSE NULL END,'owner',w.owner_id,'slug',w.slug,'organization_size',w.organization_size,'timezone',w.timezone,'background_color',w.background_color,'created_by',w.created_by_id,'updated_by',w.updated_by_id,'total_members',(SELECT count(*) FROM workspace_members wm JOIN users mu ON mu.id=wm.member_id WHERE wm.workspace_id=w.id AND wm.deleted_at IS NULL AND wm.is_active AND NOT mu.is_bot))
`, pgx.StrictNamedArgs{
		"actor": actor, "slug": slug, "set_name": set["name"], "name": name,
		"set_slug": set["slug"], "new_slug": newSlug, "set_logo": set["logo"], "logo": logo,
		"set_organization_size": set["organization_size"], "organization_size": organizationSize,
		"set_timezone": set["timezone"], "timezone": timezone,
		"set_background_color": set["background_color"], "background_color": backgroundColor,
	}).Scan(&result)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

type WorkspaceDeletion struct {
	ID, Name, Slug string
	DeletedAt      time.Time
}

func (r *Repository) PrepareWorkspaceDeletion(ctx context.Context, actor uuid.UUID, slug string) (WorkspaceDeletion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return WorkspaceDeletion{}, err
	}
	defer tx.Rollback(ctx)
	var deleted WorkspaceDeletion
	err = tx.QueryRow(ctx, `SELECT w.id::text,w.name,w.slug FROM workspaces w WHERE w.slug=@slug AND w.deleted_at IS NULL AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=@actor AND wm.role=20 AND wm.is_active AND wm.deleted_at IS NULL) FOR UPDATE`, pgx.StrictNamedArgs{"slug": slug, "actor": actor}).Scan(&deleted.ID, &deleted.Name, &deleted.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceDeletion{}, pgx.ErrNoRows
	}
	if err != nil {
		return WorkspaceDeletion{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE profiles SET last_workspace_id=NULL WHERE last_workspace_id=@id`, pgx.StrictNamedArgs{"id": deleted.ID}); err != nil {
		return WorkspaceDeletion{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return WorkspaceDeletion{}, err
	}
	return deleted, nil
}

func (r *Repository) SoftDeleteWorkspace(ctx context.Context, actor uuid.UUID, workspace WorkspaceDeletion, deletedAt time.Time) error {
	command, err := r.db.Exec(ctx, `UPDATE workspaces SET deleted_at=@deleted_at,updated_at=@deleted_at,updated_by_id=@actor,slug=slug||'__'||floor(extract(epoch from @deleted_at::timestamptz))::bigint::text WHERE id=@id AND slug=@slug AND deleted_at IS NULL`, pgx.StrictNamedArgs{"actor": actor, "id": workspace.ID, "slug": workspace.Slug, "deleted_at": deletedAt})
	if errors.Is(err, pgx.ErrNoRows) {
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Django passes this value directly to a TextField on the related member
// model. PostgreSQL preparation stringifies non-null Python values; omitted
// fields use the source view's empty-string default.
func workspaceCompanyRole(fields map[string]json.RawMessage) (any, error) {
	raw, present := fields["company_role"]
	if !present {
		return "", nil
	}
	if string(raw) == "null" {
		return nil, nil
	}
	value, ok := workspaceDecodedField(raw)
	if !ok {
		return nil, errors.New("company_role is not valid JSON")
	}
	return pythoncompat.String(value, raw)
}

func rawMessagesAsValues(messages []json.RawMessage) []any {
	values := make([]any, len(messages))
	for index := range messages {
		values[index] = messages[index]
	}
	return values
}

func hasAlphaNumeric(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

func workspaceFieldsExceedLength(name, slug string) bool {
	return utf8.RuneCountInString(name) > 80 || utf8.RuneCountInString(slug) > 48
}

func workspacePageWindow(perPage, cursor string) (limit int64, page int64, offset int64, err error) {
	limit = 10
	if perPage != "" {
		parsed, parseErr := pythoncompat.DecimalInteger(perPage)
		if parseErr != nil {
			return 0, 0, 0, &WorkspacePaginationError{Message: "Invalid per_page parameter."}
		}
		if parsed.Sign() < 0 {
			return 0, 0, 0, &WorkspacePaginationError{Message: "Invalid per_page parameter."}
		}
		if parsed.Sign() == 0 {
			return 0, 0, 0, &WorkspacePaginationError{Message: "Invalid per_page parameter."}
		}
		if parsed.Sign() > 0 && parsed.Cmp(big.NewInt(10)) > 0 {
			return 0, 0, 0, &WorkspacePaginationError{Message: "Invalid per_page value. Cannot exceed 10."}
		}
		if !parsed.IsInt64() {
			return 0, 0, 0, &WorkspacePaginationError{Message: "Invalid per_page parameter."}
		}
		limit = parsed.Int64()
	}
	request := djangoapi.Query{}
	if cursor != "" {
		request.Params = map[string]string{"cursor": cursor}
		request.Present = map[string]bool{"cursor": true}
	}
	parsedPage, parseErr := djangoapi.CursorPage(request, int(limit))
	if parseErr != nil {
		return 0, 0, 0, &WorkspacePaginationError{Message: "Invalid cursor parameter."}
	}
	page = int64(parsedPage)
	maxInt64 := int64(^uint64(0) >> 1)
	minInt64 := -maxInt64 - 1
	if limit > 0 && (page > maxInt64/limit || page < minInt64/limit) || limit < 0 && (page > minInt64/limit || page < maxInt64/limit) {
		return 0, 0, 0, &WorkspacePaginationError{Message: "Error in parsing"}
	}
	offset = page * limit
	if offset < 0 {
		return 0, 0, 0, &WorkspacePaginationError{Message: "Error in parsing"}
	}
	return limit, page, offset, nil
}

func workspaceCharField(raw json.RawMessage) (value string, present, truthy bool, fieldError string) {
	if len(raw) == 0 {
		return "", false, false, ""
	}
	input, ok := workspaceDecodedField(raw)
	if !ok {
		return "", true, false, "Not a valid string."
	}
	truthy = pythoncompat.Truthy(input)
	value, failures := djangoinput.CharField(input, false, nil)
	if len(failures) > 0 {
		fieldError = failures[0]
	}
	return value, true, truthy, fieldError
}

func workspaceOptionalText(fields map[string]json.RawMessage, key string, allowBlank, nullable bool, maximum int) (any, error) {
	raw, present := fields[key]
	if !present {
		return nil, nil
	}
	if string(raw) == "null" {
		if nullable {
			return nil, nil
		}
		return nil, &WorkspaceValidationError{Field: key, Message: "This field may not be null."}
	}
	value, ok := workspaceDecodedField(raw)
	if !ok {
		return nil, &WorkspaceValidationError{Field: key, Message: "Not a valid string."}
	}
	text, failures := djangoinput.TextField(value, allowBlank, maximum, nil)
	if len(failures) > 0 {
		return nil, &WorkspaceValidationError{Field: key, Message: failures[0]}
	}
	return text, nil
}

func workspaceDecodedField(raw json.RawMessage) (any, bool) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	return value, true
}

func workspaceRawFieldLength(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return 0
	}
	switch typed := value.(type) {
	case string:
		return utf8.RuneCountInString(typed)
	case []any:
		return len(typed)
	case map[string]any:
		return len(typed)
	default:
		return 0
	}
}

func workspaceRawString(raw json.RawMessage) bool {
	value, ok := workspaceDecodedField(raw)
	if !ok {
		return false
	}
	_, isString := value.(string)
	return isString
}

func workspaceBackgroundColor(fields map[string]json.RawMessage) (string, error) {
	if _, present := fields["background_color"]; !present {
		return randomWorkspaceBackgroundColor()
	}
	value, err := workspaceOptionalText(fields, "background_color", false, false, 255)
	if err != nil {
		return "", err
	}
	return value.(string), nil
}

func randomWorkspaceBackgroundColor() (string, error) {
	const alphabet = "0123456789abcdefABCDEF"
	bound := big.NewInt(int64(len(alphabet)))
	color := [7]byte{'#'}
	for index := 1; index < len(color); index++ {
		value, err := rand.Int(rand.Reader, bound)
		if err != nil {
			return "", err
		}
		color[index] = alphabet[value.Int64()]
	}
	return string(color[:]), nil
}
func has(fields map[string]json.RawMessage, key string) bool { _, ok := fields[key]; return ok }
func readString(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
func strPtr(value string) *string { return &value }
func pythonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return string(raw)
	}
	switch item := value.(type) {
	case string:
		return item
	case bool:
		if item {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	default:
		return fmt.Sprint(item)
	}
}
