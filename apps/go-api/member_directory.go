package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	stduuid "uuid"

	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
)

func memberFilterValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func memberFilterBoolean(value *string) (*bool, error) {
	if value == nil {
		return nil, nil
	}
	if *value != "true" && *value != "false" {
		return nil, store.ErrInvalidMemberQuery
	}
	result := *value == "true"
	return &result, nil
}
func rejectRepeatedMemberParameters(ctx context.Context, names ...string) error {
	values := ReadServiceQueryValues(ctx)
	for _, name := range names {
		if len(values[name]) > 1 {
			return memberDirectoryError(store.ErrInvalidMemberQuery)
		}
	}
	return nil
}
func workspaceMemberFilters(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugMembersLitePayload) (store.WorkspaceMembersQuery, error) {
	if err := rejectRepeatedMemberParameters(ctx, "first_name", "last_name", "email", "display_name", "role_slug", "is_active", "is_bot", "order_by"); err != nil {
		return store.WorkspaceMembersQuery{}, err
	}
	active, err := memberFilterBoolean(payload.IsActive)
	if err != nil {
		return store.WorkspaceMembersQuery{}, memberDirectoryError(err)
	}
	bot, err := memberFilterBoolean(payload.IsBot)
	if err != nil {
		return store.WorkspaceMembersQuery{}, memberDirectoryError(err)
	}
	return store.WorkspaceMembersQuery{FirstName: memberFilterValue(payload.FirstName), LastName: memberFilterValue(payload.LastName), Email: memberFilterValue(payload.Email), DisplayName: memberFilterValue(payload.DisplayName), RoleSlug: memberFilterValue(payload.RoleSlug), OrderBy: memberFilterValue(payload.OrderBy), IsActive: active, IsBot: bot}, nil
}

func memberDirectoryError(err error) error {
	if errors.Is(err, store.ErrInvalidMemberQuery) {
		return invitationsMembershipError(400, map[string]any{"detail": "Invalid member query."})
	}
	return invitationsMembershipRepositoryError(err)
}

func (service *invitationsMembershipService) ListMemberRoles(ctx context.Context, payload *api.ListMemberRolesPayload) (*api.MemberRolesPage, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authErr := service.authorizeWorkspace(ctx, principal, payload.Slug, true, 5, 15, 20); authErr != nil {
		return nil, authErr
	}
	if queryErr := rejectRepeatedMemberParameters(ctx, "namespace", "per_page", "cursor"); queryErr != nil {
		return nil, queryErr
	}
	size := 20
	if payload.PerPage != nil {
		size, err = strconv.Atoi(*payload.PerPage)
		if err != nil {
			return nil, memberDirectoryError(store.ErrInvalidMemberQuery)
		}
	}
	cursor, err := store.ParseMemberCursor(payload.Cursor, size)
	if err != nil || (cursor.Previous && cursor.Value != float64(size)) {
		return nil, memberDirectoryError(store.ErrInvalidMemberQuery)
	}
	roles, total, err := service.repository.ListRoles(ctx, stduuid.UUID(principal.UserID), payload.Slug, memberFilterValue(payload.Namespace), size, cursor.Page)
	if err != nil {
		return nil, memberDirectoryError(err)
	}
	page, err := store.NewMemberPage(roles, total, size, cursor.Page)
	if err != nil {
		return nil, memberDirectoryError(err)
	}
	body, err := json.Marshal(page)
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	return decodeInvitationsMembershipResponse[*api.MemberRolesPage](ctx, body)
}

func (service *invitationsMembershipService) RetrieveMemberRole(ctx context.Context, payload *api.RetrieveMemberRolePayload) (*api.MemberRoleDefinition, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authErr := service.authorizeWorkspace(ctx, principal, payload.Slug, true, 5, 15, 20); authErr != nil {
		return nil, authErr
	}
	id, err := stduuid.Parse(payload.RoleID)
	if err != nil {
		return nil, memberDirectoryError(store.ErrInvalidMemberQuery)
	}
	role, err := service.repository.RetrieveRole(ctx, stduuid.UUID(principal.UserID), payload.Slug, id)
	if err != nil {
		return nil, memberDirectoryError(err)
	}
	return &api.MemberRoleDefinition{ID: role.ID, Name: role.Name, Slug: role.Slug, Namespace: role.Namespace}, nil
}
