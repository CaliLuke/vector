package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type invitationDatabase interface {
	database
	Begin(context.Context) (pgx.Tx, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// InvitationsRepository owns invitation and workspace membership reads and writes.
type InvitationsRepository struct {
	reader    database
	writer    invitationDatabase
	publisher TaskPublisher
	baseHost  string
	secretKey string
}

func NewInvitationsRepositoryWithReaderWriter(reader, writer invitationDatabase, publisher TaskPublisher, baseHost, secretKey string) (*InvitationsRepository, error) {
	if reader == nil || writer == nil || publisher == nil {
		return nil, errors.New("invitation reader, writer, and task publisher are required")
	}
	return &InvitationsRepository{reader: reader, writer: writer, publisher: publisher, baseHost: baseHost, secretKey: secretKey}, nil
}

// WorkspaceInvites returns the API v1 workspace invite projection. The API
// serializer deliberately omits the private token and invitation message.
func (repository *Repository) WorkspaceInvites(ctx context.Context, slug string) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE(jsonb_agg(jsonb_build_object(
  'id', invite.id, 'email', invite.email, 'role', invite.role,
  'accepted', invite.accepted, 'responded_at', invite.responded_at,
  'created_at', invite.created_at, 'updated_at', invite.updated_at
) ORDER BY invite.created_at DESC), '[]'::jsonb)
FROM workspace_member_invites invite
JOIN workspaces workspace ON workspace.id = invite.workspace_id
WHERE workspace.slug = $1 AND invite.deleted_at IS NULL
`, slug).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list workspace invitations: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *InvitationsRepository) WorkspaceInvites(ctx context.Context, slug string) (json.RawMessage, error) {
	return (&Repository{database: repository.writer}).WorkspaceInvites(ctx, slug)
}

func (repository *InvitationsRepository) WorkspaceInvite(ctx context.Context, slug string, inviteID uuid.UUID) (json.RawMessage, error) {
	return (&Repository{database: repository.writer}).WorkspaceInvite(ctx, slug, inviteID)
}

func (repository *InvitationsRepository) AuthorizeWorkspaceRole(ctx context.Context, userID uuid.UUID, slug string, roles ...int) error {
	db := database(repository.writer)
	if replica, _ := ctx.Value(invitationReplicaContextKey{}).(bool); replica {
		db = repository.reader
	}
	return (&Repository{database: db}).AuthorizeWorkspaceRole(ctx, userID, slug, roles...)
}

func (repository *InvitationsRepository) WorkspaceExists(ctx context.Context, slug string) (bool, error) {
	return (&Repository{database: repository.reader}).WorkspaceExists(ctx, slug)
}

func (repository *InvitationsRepository) WorkspaceMemberUsers(ctx context.Context, slug string) (json.RawMessage, error) {
	return (&Repository{database: repository.reader}).WorkspaceMemberUsers(ctx, slug)
}

func (repository *InvitationsRepository) WorkspaceMemberLitePage(ctx context.Context, slug string, limit, offset int) (json.RawMessage, int, error) {
	return (&Repository{database: repository.reader}).WorkspaceMemberLitePage(ctx, slug, limit, offset)
}

func invitationHTTP(status int, body map[string]any) error {
	return &djangoapi.HTTPError{Status: status, Body: body}
}

func apiInvitationValues(payload map[string]any, requireEmail bool) (*string, *int, map[string]any) {
	failures := map[string]any{}
	var email *string
	var role *int
	if raw, present := payload["email"]; present {
		value, messages := djangoinput.TextField(raw, false, 255, nil)
		if len(messages) > 0 {
			failures["email"] = messages
		} else if !validInvitationEmail(value) {
			failures["email"] = []string{"Invalid email address"}
		} else {
			email = &value
		}
	} else if requireEmail {
		failures["email"] = []string{"This field is required."}
	}
	if raw, present := payload["role"]; present {
		value, valid := invitationRole(raw)
		if !valid {
			failures["role"] = invitationChoiceMessages(raw)
		} else {
			role = &value
		}
	}
	return email, role, failures
}
func decodeInvitationSerializerPayload(body io.Reader, contentType string) (map[string]any, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	return invitationSerializerData(input)
}

func invitationSerializerData(input djangoinput.Input) (map[string]any, error) {
	if input.Kind != "" {
		message := "Invalid data. Expected a dictionary, but got " + input.Kind + "."
		if input.Null {
			message = "No data provided"
		}
		return nil, invitationHTTP(400, map[string]any{"non_field_errors": []string{message}})
	}
	return input.Fields, nil
}
func (repository *InvitationsRepository) CreateAPIWorkspaceInvite(ctx context.Context, slug string, userID uuid.UUID, body io.Reader, contentType string) (json.RawMessage, error) {
	var workspaceID string
	if err := repository.writer.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, slug).Scan(&workspaceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	payload, err := decodeInvitationSerializerPayload(body, contentType)
	if err != nil {
		return nil, err
	}
	email, role, failures := apiInvitationValues(payload, true)
	if len(failures) > 0 {
		return nil, invitationHTTP(400, failures)
	}
	var duplicate bool
	if err = repository.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_member_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.email=$2 AND i.deleted_at IS NULL)`, slug, *email).Scan(&duplicate); err != nil {
		return nil, err
	}
	if duplicate {
		return nil, invitationHTTP(400, map[string]any{"non_field_errors": []string{"Email already invited"}})
	}
	selectedRole := 5
	if role != nil {
		selectedRole = *role
	}
	return inviteJSON(ctx, repository.writer, `INSERT INTO workspace_member_invites(id,workspace_id,email,role,token,accepted,created_by_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,'',false,$4,clock_timestamp(),clock_timestamp()) RETURNING jsonb_build_object('id',id,'email',email,'role',role,'accepted',accepted,'responded_at',responded_at,'created_at',created_at,'updated_at',updated_at)`, workspaceID, *email, selectedRole, userID)
}
func (repository *InvitationsRepository) PatchAPIWorkspaceInvite(ctx context.Context, slug string, inviteID uuid.UUID, body io.Reader, contentType string) (json.RawMessage, error) {
	current, err := repository.WorkspaceInvite(ctx, slug, inviteID)
	if err != nil {
		return nil, err
	}
	// Source partial_update calls request.data.get before its serializer.
	payload, err := invitationCommandFields(&InvitationCommand{Body: body, ContentType: contentType})
	if err != nil {
		return nil, err
	}
	return repository.updateAPIWorkspaceInvite(ctx, slug, inviteID, current, payload, true)
}
func (repository *InvitationsRepository) updateAPIWorkspaceInvite(ctx context.Context, slug string, inviteID uuid.UUID, current json.RawMessage, payload map[string]any, partial bool) (json.RawMessage, error) {
	var previous struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(current, &previous); err != nil {
		return nil, err
	}
	if raw, present := payload["email"]; present && invitationTruth(raw) {
		same, valid := raw.(string)
		if partial || !valid || same != previous.Email {
			return nil, invitationHTTP(400, map[string]any{"error": "Email cannot be updated after invite is created.", "code": "EMAIL_CANNOT_BE_UPDATED"})
		}
	}
	email, role, failures := apiInvitationValues(payload, !partial)
	if len(failures) > 0 {
		return nil, invitationHTTP(400, failures)
	}
	if email != nil {
		var duplicate bool
		if err := repository.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_member_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.email=$2 AND i.id<>$3 AND i.deleted_at IS NULL)`, slug, *email, inviteID).Scan(&duplicate); err != nil {
			return nil, err
		}
		if duplicate {
			return nil, invitationHTTP(400, map[string]any{"non_field_errors": []string{"Email already invited"}})
		}
	}
	return inviteJSON(ctx, repository.writer, `UPDATE workspace_member_invites i SET email=COALESCE($3,i.email),role=COALESCE($4,i.role),updated_at=clock_timestamp(),updated_by_id=$5 FROM workspaces w WHERE i.workspace_id=w.id AND w.slug=$1 AND i.id=$2 AND i.deleted_at IS NULL RETURNING jsonb_build_object('id',i.id,'email',i.email,'role',i.role,'accepted',i.accepted,'responded_at',i.responded_at,'created_at',i.created_at,'updated_at',i.updated_at)`, slug, inviteID, email, role, invitationActor(ctx))
}

func (repository *InvitationsRepository) DeleteAPIWorkspaceInvite(ctx context.Context, slug string, inviteID uuid.UUID) error {
	var accepted bool
	var respondedAt *time.Time
	err := repository.writer.QueryRow(ctx, `SELECT invite.accepted,invite.responded_at FROM workspace_member_invites invite JOIN workspaces workspace ON workspace.id=invite.workspace_id WHERE workspace.slug=$1 AND invite.id=$2 AND invite.deleted_at IS NULL`, slug, inviteID).Scan(&accepted, &respondedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return invitationHTTP(404, map[string]any{"error": "The required object does not exist."})
	}
	if err != nil {
		return err
	}
	if accepted {
		return invitationHTTP(400, map[string]any{"error": "Invite already accepted", "code": "INVITE_ALREADY_ACCEPTED"})
	}
	if respondedAt != nil {
		return invitationHTTP(400, map[string]any{"error": "Invite already responded", "code": "INVITE_ALREADY_RESPONDED"})
	}
	if _, err := repository.writer.Exec(ctx, `UPDATE workspace_member_invites SET deleted_at=now(),updated_at=now(),updated_by_id=$2 WHERE id=$1`, inviteID, invitationActor(ctx)); err != nil {
		return err
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.deletion_task.soft_delete_related_objects", []any{"db", "workspacememberinvite", map[string]any{"__type__": "uuid", "__value__": map[string]string{"hex": strings.ReplaceAll(inviteID.String(), "-", "")}}}, map[string]any{"using": nil})
}

func (repository *Repository) WorkspaceInvite(ctx context.Context, slug string, inviteID uuid.UUID) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT jsonb_build_object(
  'id', invite.id, 'email', invite.email, 'role', invite.role,
  'accepted', invite.accepted, 'responded_at', invite.responded_at,
  'created_at', invite.created_at, 'updated_at', invite.updated_at
)
FROM workspace_member_invites invite
JOIN workspaces workspace ON workspace.id = invite.workspace_id
WHERE workspace.slug = $1 AND invite.id = $2 AND invite.deleted_at IS NULL
`, slug, inviteID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace invitation: %w", err)
	}
	return json.RawMessage(data), nil
}

// AuthorizeWorkspaceRole matches the API's active-member role admission. The
// API v1 invitation routes require an owner (20); workspace member directories
// admit members and owners (15, 20).
func (repository *Repository) AuthorizeWorkspaceRole(ctx context.Context, userID uuid.UUID, slug string, roles ...int) error {
	var allowed bool
	roleValues := make([]int16, len(roles))
	for index, role := range roles {
		roleValues[index] = int16(role)
	}
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM workspace_members member
  JOIN workspaces workspace ON workspace.id = member.workspace_id
  WHERE workspace.slug = $2
    AND member.member_id = $1 AND member.is_active = true
    AND member.deleted_at IS NULL AND member.role = ANY($3::smallint[])
)
`, userID, slug, roleValues).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize workspace role: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) WorkspaceExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = $1 AND deleted_at IS NULL)
`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check workspace existence: %w", err)
	}
	return exists, nil
}

func (repository *Repository) WorkspaceMemberUsers(ctx context.Context, slug string) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE(jsonb_agg(jsonb_build_object(
  'id', member.id, 'first_name', member.first_name, 'last_name', member.last_name,
  'email', member.email, 'avatar', member.avatar,
  'avatar_url', CASE WHEN member.avatar_asset_id IS NOT NULL
    THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
    ELSE NULLIF(member.avatar, '') END,
  'display_name', member.display_name, 'role', workspace_member.role
 ) ORDER BY workspace_member.created_at DESC), '[]'::jsonb)
FROM workspace_members workspace_member
JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
JOIN users member ON member.id = workspace_member.member_id
WHERE workspace.slug = $1 AND workspace_member.deleted_at IS NULL
`, slug).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list workspace member users: %w", err)
	}
	return json.RawMessage(data), nil
}

// WorkspaceMemberLitePage returns the source serializer projection and count.
// Cursor arithmetic and envelope construction stay in the transport adapter.
func (repository *Repository) WorkspaceMemberLitePage(ctx context.Context, slug string, limit, offset int) (json.RawMessage, int, error) {
	if limit < 1 {
		return nil, 0, fmt.Errorf("workspace member page size must be positive")
	}
	var data []byte
	var count int
	err := repository.database.QueryRow(ctx, `
WITH totals AS (
  SELECT count(*)::int AS total_count
  FROM workspace_members workspace_member
  JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
  WHERE workspace.slug = $1 AND workspace_member.deleted_at IS NULL
), page AS (
  SELECT jsonb_build_object(
    'id', member.id, 'first_name', member.first_name, 'last_name', member.last_name,
    'email', member.email, 'avatar', member.avatar,
    'avatar_url', CASE WHEN member.avatar_asset_id IS NOT NULL
      THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
      ELSE NULLIF(member.avatar, '') END,
    'display_name', member.display_name, 'role', workspace_member.role,
    'is_active', workspace_member.is_active, 'is_bot', member.is_bot
  ) AS member, workspace_member.created_at
  FROM workspace_members workspace_member
  JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
  JOIN users member ON member.id = workspace_member.member_id
  WHERE workspace.slug = $1 AND workspace_member.deleted_at IS NULL
  ORDER BY workspace_member.created_at DESC
  LIMIT $2 OFFSET $3
)
SELECT COALESCE((SELECT jsonb_agg(page.member ORDER BY page.created_at DESC) FROM page), '[]'::jsonb),
       totals.total_count
FROM totals
`, slug, limit, offset).Scan(&data, &count)
	if err != nil {
		return nil, 0, fmt.Errorf("list lite workspace members: %w", err)
	}
	if len(data) == 0 {
		data = []byte("[]")
	}
	return json.RawMessage(data), count, nil
}
