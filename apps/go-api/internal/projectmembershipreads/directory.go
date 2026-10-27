package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"uuid"

	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DirectoryReader owns authorized workspace member and role catalogue reads.
// Construct it with the primary connection, never a read replica.
type DirectoryReader struct{ primary directoryDatabase }
type directoryDatabase interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewDirectoryReader(primary directoryDatabase) (*DirectoryReader, error) {
	if primary == nil {
		return nil, errors.New("primary membership database is required")
	}
	return &DirectoryReader{primary: primary}, nil
}

var ErrInvalidMemberQuery = errors.New("invalid member query")

type WorkspaceMembersQuery struct {
	// ExpectedWorkspaceID binds a previously verified workspace through slug changes.
	ExpectedWorkspaceID                               *uuid.UUID
	FirstName, LastName, Email, DisplayName, RoleSlug string
	IsActive, IsBot                                   *bool
	OrderBy                                           string
	Limit, Page                                       int
}

type WorkspaceMembersPage struct {
	Results json.RawMessage
	Total   int
}

// RoleDefinition describes an existing role; it never grants permissions.
type RoleDefinition struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Namespace string `json:"namespace"`
}

type roleMapping struct {
	Namespace string `json:"namespace"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Owner     bool   `json:"owner"`
}

// Namespace, then semantic rank. Project Member is Contributor in the pinned
// API catalogue. Commenter and custom roles have no existing permission model.
var membershipRoleMappings = [...]roleMapping{
	{"project", "admin", "Admin", 20, false},
	{"project", "contributor", "Contributor", 15, false},
	{"project", "guest", "Guest", 5, false},
	{"workspace", "owner", "Owner", 20, true},
	{"workspace", "admin", "Admin", 20, false},
	{"workspace", "member", "Member", 15, false},
	{"workspace", "guest", "Guest", 5, false},
}

func (r *DirectoryReader) roleCatalogue(ctx context.Context, actor uuid.UUID, slug string, expectedWorkspace *uuid.UUID) ([]RoleDefinition, error) {
	if actor == (uuid.UUID{}) || slug == "" {
		return nil, ErrForbidden
	}
	var workspace googleuuid.UUID
	err := r.primary.QueryRow(ctx, `SELECT w.id FROM workspaces w
 JOIN workspace_members actor ON actor.workspace_id=w.id
 JOIN users actor_user ON actor_user.id=actor.member_id
 WHERE w.slug=$2 AND ($3::uuid IS NULL OR w.id=$3) AND w.deleted_at IS NULL AND actor.member_id=$1
 AND actor.is_active AND actor.deleted_at IS NULL AND actor.role IN (5,15,20)
 AND actor_user.is_active`, actor, slug, expectedWorkspace).Scan(&workspace)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, fmt.Errorf("read role catalogue scope: %w", err)
	}
	result := make([]RoleDefinition, 0, len(membershipRoleMappings))
	for _, mapping := range membershipRoleMappings {
		id := googleuuid.NewSHA1(workspace, []byte("vector.role/v1/"+mapping.Namespace+"/"+mapping.Slug))
		result = append(result, RoleDefinition{ID: id.String(), Name: mapping.Name, Slug: mapping.Slug, Namespace: mapping.Namespace})
	}
	return result, nil
}

func (r *DirectoryReader) ListRoles(ctx context.Context, actor uuid.UUID, slug, namespace string, limit, page int) ([]RoleDefinition, int, error) {
	return r.listRoles(ctx, actor, slug, namespace, limit, page, nil)
}

// ListRolesInWorkspace retains the caller's immutable workspace selection.
func (r *DirectoryReader) ListRolesInWorkspace(ctx context.Context, actor uuid.UUID, slug, namespace string, limit, page int, workspace uuid.UUID) ([]RoleDefinition, int, error) {
	return r.listRoles(ctx, actor, slug, namespace, limit, page, &workspace)
}

func (r *DirectoryReader) listRoles(ctx context.Context, actor uuid.UUID, slug, namespace string, limit, page int, expectedWorkspace *uuid.UUID) ([]RoleDefinition, int, error) {
	offset, err := memberPageOffset(limit, page)
	if err != nil || (namespace != "" && namespace != "workspace" && namespace != "project") {
		return nil, 0, ErrInvalidMemberQuery
	}
	roles, err := r.roleCatalogue(ctx, actor, slug, expectedWorkspace)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]RoleDefinition, 0, len(roles))
	for _, role := range roles {
		if namespace == "" || role.Namespace == namespace {
			filtered = append(filtered, role)
		}
	}
	total := len(filtered)
	start := min(offset, total)
	end := start + min(limit, total-start)
	return filtered[start:end], total, nil
}

func (r *DirectoryReader) RetrieveRole(ctx context.Context, actor uuid.UUID, slug string, id uuid.UUID) (RoleDefinition, error) {
	return r.retrieveRole(ctx, actor, slug, id, nil)
}

// RetrieveRoleInWorkspace retains the caller's immutable workspace selection.
func (r *DirectoryReader) RetrieveRoleInWorkspace(ctx context.Context, actor uuid.UUID, slug string, id, workspace uuid.UUID) (RoleDefinition, error) {
	return r.retrieveRole(ctx, actor, slug, id, &workspace)
}

func (r *DirectoryReader) retrieveRole(ctx context.Context, actor uuid.UUID, slug string, id uuid.UUID, expectedWorkspace *uuid.UUID) (RoleDefinition, error) {
	roles, err := r.roleCatalogue(ctx, actor, slug, expectedWorkspace)
	if err != nil {
		return RoleDefinition{}, err
	}
	for _, role := range roles {
		if role.ID == id.String() {
			return role, nil
		}
	}
	return RoleDefinition{}, ErrNotFound
}

func memberPageOffset(limit, page int) (int, error) {
	maxInt := int(^uint(0) >> 1)
	if limit < 1 || limit > 1000 || page < 0 || page > (maxInt-1)/limit {
		return 0, ErrInvalidMemberQuery
	}
	return page * limit, nil
}

func memberDirectoryOrder(raw string) (string, error) {
	if raw == "" {
		raw = "-created_at"
	}
	columns := map[string]string{
		"created_at": "membership_created_at", "updated_at": "membership_updated_at",
		"first_name": "first_name", "last_name": "last_name", "email": "email", "display_name": "display_name",
		"role": "role", "is_active": "is_active", "is_bot": "is_bot",
	}
	seen := make(map[string]bool)
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		name := strings.TrimPrefix(part, "-")
		column, ok := columns[name]
		if !ok || seen[name] {
			return "", ErrInvalidMemberQuery
		}
		seen[name] = true
		direction := "ASC"
		if strings.HasPrefix(part, "-") {
			direction = "DESC"
		}
		result = append(result, column+" "+direction)
	}
	return strings.Join(append(result, "membership_id ASC"), ", "), nil
}

func (r *DirectoryReader) ListWorkspaceMembers(ctx context.Context, actor uuid.UUID, slug string, query WorkspaceMembersQuery) (WorkspaceMembersPage, error) {
	if actor == (uuid.UUID{}) || slug == "" {
		return WorkspaceMembersPage{}, ErrForbidden
	}
	offset, err := memberPageOffset(query.Limit, query.Page)
	if err != nil {
		return WorkspaceMembersPage{}, err
	}
	ordering, err := memberDirectoryOrder(query.OrderBy)
	if err != nil {
		return WorkspaceMembersPage{}, err
	}
	mappingJSON, err := json.Marshal(membershipRoleMappings)
	if err != nil {
		return WorkspaceMembersPage{}, fmt.Errorf("encode member role mapping: %w", err)
	}
	var body []byte
	var total int
	err = r.primary.QueryRow(ctx, `WITH scope AS (
 SELECT w.id,w.owner_id FROM workspaces w
 WHERE w.slug=@slug AND (@workspace::uuid IS NULL OR w.id=@workspace) AND w.deleted_at IS NULL AND EXISTS (
  SELECT 1 FROM workspace_members actor JOIN users actor_user ON actor_user.id=actor.member_id
  WHERE actor.workspace_id=w.id AND actor.member_id=@actor AND actor.is_active
  AND actor.deleted_at IS NULL AND actor.role IN (15,20) AND actor_user.is_active
 )
), members AS MATERIALIZED (
 SELECT wm.id AS membership_id,wm.created_at AS membership_created_at,wm.updated_at AS membership_updated_at,
 u.id,u.first_name,u.last_name,u.email,u.avatar,
 CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/'
 ELSE NULLIF(u.avatar,'') END AS avatar_url,
 u.display_name,wm.role,wm.is_active,u.is_bot,
 (SELECT m->>'slug' FROM jsonb_array_elements(@mappings::jsonb) m
  WHERE m->>'namespace'='workspace' AND (m->>'level')::int=wm.role
  AND (m->>'owner')::boolean=(wm.member_id=scope.owner_id AND wm.role=20)) AS role_slug
 FROM scope JOIN workspace_members wm ON wm.workspace_id=scope.id JOIN users u ON u.id=wm.member_id
 WHERE wm.deleted_at IS NULL
), filtered AS MATERIALIZED (
 SELECT * FROM members WHERE
 (@first_name='' OR strpos(lower(COALESCE(first_name,'')),lower(@first_name))>0) AND
 (@last_name='' OR strpos(lower(COALESCE(last_name,'')),lower(@last_name))>0) AND
 (@email='' OR strpos(lower(COALESCE(email,'')),lower(@email))>0) AND
 (@display_name='' OR strpos(lower(COALESCE(display_name,'')),lower(@display_name))>0) AND
 (@role_slug='' OR role_slug=@role_slug) AND
 (@is_active::boolean IS NULL OR is_active=@is_active) AND
 (@is_bot::boolean IS NULL OR is_bot=@is_bot)
), page AS (
 SELECT *,row_number() OVER (ORDER BY `+ordering+`) AS position
 FROM filtered ORDER BY `+ordering+` LIMIT @limit OFFSET @offset
)
SELECT COALESCE((SELECT jsonb_agg(jsonb_build_object(
 'id',id,'first_name',COALESCE(first_name,''),'last_name',COALESCE(last_name,''),'email',COALESCE(email,''),
 'avatar',avatar,'avatar_url',avatar_url,'display_name',COALESCE(display_name,''),
 'role',role,'role_slug',role_slug,'is_active',is_active,'is_bot',is_bot
 ) ORDER BY position) FROM page),'[]'::jsonb),
 (SELECT count(*)::int FROM filtered) FROM scope`, pgx.StrictNamedArgs{
		"slug": slug, "workspace": query.ExpectedWorkspaceID, "actor": actor, "mappings": string(mappingJSON),
		"first_name": query.FirstName, "last_name": query.LastName, "email": query.Email, "display_name": query.DisplayName,
		"role_slug": query.RoleSlug, "is_active": query.IsActive, "is_bot": query.IsBot,
		"limit": query.Limit + 1, "offset": offset,
	}).Scan(&body, &total)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceMembersPage{}, ErrForbidden
	}
	if err != nil {
		return WorkspaceMembersPage{}, fmt.Errorf("read workspace member directory: %w", err)
	}
	return WorkspaceMembersPage{Results: json.RawMessage(body), Total: total}, nil
}
