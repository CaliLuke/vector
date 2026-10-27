// Package mcpactions projects authenticated application operations onto MCP.
package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type identityReader interface {
	CurrentIdentity(context.Context, uuid.UUID) (json.RawMessage, error)
}

type projectMemberReader interface {
	ListProjectMembersInWorkspace(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
}

type memberDirectory interface {
	ReadCapabilities(context.Context, uuid.UUID, uuid.UUID, string) (membership.ReadCapabilities, error)
	ListWorkspaceMembers(context.Context, uuid.UUID, string, membership.WorkspaceMembersQuery) (membership.WorkspaceMembersPage, error)
	ListRolesInWorkspace(context.Context, uuid.UUID, string, string, int, int, uuid.UUID) ([]membership.RoleDefinition, int, error)
	RetrieveRoleInWorkspace(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (membership.RoleDefinition, error)
}

type Service struct {
	logContext    context.Context
	identity      identityReader
	projects      projectMemberReader
	directory     memberDirectory
	projectReads  projectReader
	projectWrites projectWriter
}

func New(identity identityReader, projects projectMemberReader, directory memberDirectory, projectReads projectReader, projectWrites projectWriter, options ...Option) (*Service, error) {
	if identity == nil || projects == nil || directory == nil || projectReads == nil || projectWrites == nil {
		return nil, errors.New("MCP identity, member directory and project applications are required")
	}
	service := &Service{identity: identity, projects: projects, directory: directory, projectReads: projectReads, projectWrites: projectWrites}
	for _, option := range options {
		option(service)
	}
	return service, nil
}

func (s *Service) Member(ctx context.Context, input *api.MemberRequest) (*api.MemberResult, error) {
	request, err := selectMemberAction(input)
	if err != nil {
		return nil, err
	}
	ctx, actor, err := readActor(ctx, request.workspace)
	if err != nil {
		return nil, err
	}

	switch request.action {
	case "list_workspace":
		return s.workspaceMembers(ctx, actor, request.workspaceList)
	case "list_roles":
		return s.memberRoles(ctx, actor, request.rolesList)
	case "retrieve_role":
		return s.memberRole(ctx, actor, request.role)
	}
	if request.action == "list_project" {
		members, readErr := s.projectMembers(ctx, actor, request.project)
		if readErr != nil {
			return nil, readErr
		}
		result := api.NewMemberResultProjectMembers(members)
		return &result, nil
	}
	identity, err := s.currentIdentity(ctx, actor)
	if err != nil {
		return nil, err
	}
	result := api.NewMemberResultMemberIdentity(identity)
	return &result, nil
}

type memberRequest struct {
	action, project string
	workspace       *string
	workspaceList   *api.MemberListWorkspace
	rolesList       *api.MemberListRoles
	role            *api.MemberRetrieveRole
}

func selectMemberAction(input *api.MemberRequest) (memberRequest, error) {
	if input != nil {
		if me, ok := input.AsMemberMe(); ok && me != nil && me.Action == "me" {
			return memberRequest{action: me.Action, workspace: me.WorkspaceSlug}, nil
		}
		if list, ok := input.AsMemberListProject(); ok && list != nil && list.Action == "list_project" {
			return memberRequest{action: list.Action, project: list.ProjectID, workspace: list.WorkspaceSlug}, nil
		}
		if list, ok := input.AsMemberListWorkspace(); ok && list != nil && list.Action == "list_workspace" {
			return memberRequest{action: list.Action, workspace: list.WorkspaceSlug, workspaceList: list}, nil
		}
		if list, ok := input.AsMemberListRoles(); ok && list != nil && list.Action == "list_roles" {
			return memberRequest{action: list.Action, workspace: list.WorkspaceSlug, rolesList: list}, nil
		}
		if role, ok := input.AsMemberRetrieveRole(); ok && role != nil && role.Action == "retrieve_role" {
			return memberRequest{action: role.Action, workspace: role.WorkspaceSlug, role: role}, nil
		}
	}
	return memberRequest{}, api.MakeInvalidParams(errors.New("invalid member action"))
}

func (s *Service) currentIdentity(ctx context.Context, actor mcpauth.Principal) (*api.MemberIdentity, error) {
	body, err := s.identity.CurrentIdentity(ctx, uuid.UUID(actor.UserID))
	if errors.Is(err, useridentity.ErrNotFound) {
		return nil, api.MakeNotFound(errors.New("identity unavailable"))
	}
	if err != nil {
		return nil, memberReadFailure()
	}
	// The pinned SDK fills absent UserLite properties with null. Keep that
	// normalization in this projection; HTTP retains its existing wire shape.
	result := &api.MemberIdentity{
		ID: loom.NullValue[string](), FirstName: loom.NullValue[string](),
		LastName: loom.NullValue[string](), Email: loom.NullValue[string](),
		Avatar: loom.NullValue[string](), AvatarURL: loom.NullValue[string](),
		DisplayName: loom.NullValue[string](),
	}
	if decodeErr := json.Unmarshal(body, &result); decodeErr != nil || result == nil {
		return nil, memberReadFailure()
	}
	return result, nil
}

func authenticationError(err error) error {
	switch {
	case errors.Is(err, mcptransport.ErrInvalidCredential):
		return api.MakeUnauthorized(errors.New("authentication expired"))
	case errors.Is(err, mcptransport.ErrAccessDenied):
		return api.MakeForbidden(errors.New("access revoked"))
	default:
		return memberReadFailure()
	}
}

func memberReadFailure() error {
	return api.MakeInternalError(errors.New("member read failed"))
}

var _ api.Service = (*Service)(nil)

func readActor(ctx context.Context, workspace *string) (context.Context, mcpauth.Principal, error) {
	return actionActor(ctx, workspace, false)
}

func actionActor(ctx context.Context, workspace *string, mutation bool) (context.Context, mcpauth.Principal, error) {
	ctx, err := mcpauth.Revalidate(ctx)
	if err != nil {
		return nil, mcpauth.Principal{}, authenticationError(err)
	}
	if err = mcpauth.RequireScopes(ctx, mutation); err != nil {
		return nil, mcpauth.Principal{}, authenticationError(err)
	}
	slug := ""
	if workspace != nil {
		slug = *workspace
	}
	actor, err := mcpauth.SelectWorkspace(ctx, slug)
	if err != nil {
		if errors.Is(err, mcpauth.ErrWorkspaceMismatch) {
			return nil, mcpauth.Principal{}, api.MakeInvalidParams(errors.New("invalid workspace selector"))
		}
		return nil, mcpauth.Principal{}, authenticationError(err)
	}
	if timezoneErr := djangosession.ValidateUserTimezone(actor.Timezone); timezoneErr != nil {
		return nil, mcpauth.Principal{}, memberReadFailure()
	}

	return ctx, actor, nil
}
