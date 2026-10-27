package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func optionalMemberText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func optionalMemberBool(value loom.Nullable[bool]) *bool {
	v, ok := value.Value()
	if !ok {
		return nil
	}
	return &v
}

func memberPageInput(sizeArg *int, cursorArg *string, fallback int, legacy bool, workspace string) (int, int, error) {
	size := fallback
	if sizeArg != nil && *sizeArg != 0 {
		size = *sizeArg
	}
	if size < 1 || size > 1000 || workspace == "" {
		return 0, 0, directoryFailure(membership.ErrInvalidMemberQuery)
	}
	if optionalMemberText(cursorArg) == "" {
		cursorArg = nil
	}
	cursor, err := membership.ParseMemberCursor(cursorArg, size)
	if err != nil || cursor.Page < 0 {
		return 0, 0, directoryFailure(membership.ErrInvalidMemberQuery)
	}
	if cursor.Previous && cursor.Value != float64(size) {
		if legacy {
			return 0, 0, memberReadFailure()
		}
		return 0, 0, directoryFailure(membership.ErrInvalidMemberQuery)
	}
	return size, cursor.Page, nil
}

func (s *Service) workspaceMembers(ctx context.Context, actor mcpauth.Principal, input *api.MemberListWorkspace) (*api.MemberResult, error) {
	size, page, err := memberPageInput(input.PerPage, input.Cursor, 100, true, actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	query := membership.WorkspaceMembersQuery{
		ExpectedWorkspaceID: new(uuid.UUID(actor.WorkspaceID)),
		FirstName:           optionalMemberText(input.FirstName), LastName: optionalMemberText(input.LastName),
		Email: optionalMemberText(input.Email), DisplayName: optionalMemberText(input.DisplayName),
		RoleSlug: optionalMemberText(input.RoleSlug), OrderBy: optionalMemberText(input.OrderBy),
		IsActive: optionalMemberBool(input.IsActive), IsBot: optionalMemberBool(input.IsBot), Limit: size, Page: page,
	}
	result, err := s.directory.ListWorkspaceMembers(ctx, uuid.UUID(actor.UserID), actor.WorkspaceSlug, query)
	if err != nil {
		return nil, directoryFailure(err)
	}
	rows, err := decodeMemberRows(result.Results)
	if err != nil {
		return nil, err
	}
	body, err := encodeMemberPage[api.MemberDirectoryPage](rows, result.Total, size, page)
	if err != nil {
		return nil, err
	}
	reply := api.NewMemberResultMemberDirectoryPage(body)
	return &reply, nil
}

func (s *Service) memberRoles(ctx context.Context, actor mcpauth.Principal, input *api.MemberListRoles) (*api.MemberResult, error) {
	size, page, err := memberPageInput(input.PerPage, input.Cursor, 20, false, actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	roles, total, err := s.directory.ListRolesInWorkspace(ctx, uuid.UUID(actor.UserID), actor.WorkspaceSlug, optionalMemberText(input.Namespace), size, page, uuid.UUID(actor.WorkspaceID))
	if err != nil {
		return nil, directoryFailure(err)
	}
	body, err := encodeMemberPage[api.MemberDirectoryPage](roles, total, size, page)
	if err != nil {
		return nil, err
	}
	reply := api.NewMemberResultMemberDirectoryPage(body)
	return &reply, nil
}

func (s *Service) memberRole(ctx context.Context, actor mcpauth.Principal, input *api.MemberRetrieveRole) (*api.MemberResult, error) {
	id, err := uuid.Parse(input.RoleID)
	if err != nil || actor.WorkspaceSlug == "" {
		return nil, directoryFailure(membership.ErrInvalidMemberQuery)
	}
	role, err := s.directory.RetrieveRoleInWorkspace(ctx, uuid.UUID(actor.UserID), actor.WorkspaceSlug, id, uuid.UUID(actor.WorkspaceID))
	if err != nil {
		return nil, directoryFailure(err)
	}
	reply := api.NewMemberResultMemberRoleDefinition(&api.MemberRoleDefinition{ID: role.ID, Name: role.Name, Slug: role.Slug, Namespace: role.Namespace})
	return &reply, nil
}

func encodeMemberPage[R, T any](rows []T, total, size, page int) (*R, error) {
	envelope, err := membership.NewMemberPage(rows, total, size, page)
	if err != nil {
		return nil, memberReadFailure()
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, memberReadFailure()
	}
	result := new(R)
	if err := json.Unmarshal(body, result); err != nil {
		return nil, memberReadFailure()
	}
	return result, nil
}

func directoryFailure(err error) error {
	switch {
	case errors.Is(err, membership.ErrForbidden):
		return api.MakeForbidden(errors.New("workspace access denied"))
	case errors.Is(err, membership.ErrNotFound):
		return api.MakeNotFound(errors.New("member resource unavailable"))
	case errors.Is(err, membership.ErrInvalidMemberQuery):
		return api.MakeInvalidParams(errors.New("invalid member query"))
	default:
		return memberReadFailure()
	}
}
