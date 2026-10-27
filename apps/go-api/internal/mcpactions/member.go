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
	"github.com/CaliLuke/go-plane/internal/useridentity"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type identityReader interface {
	CurrentIdentity(context.Context, uuid.UUID) (json.RawMessage, error)
}

type projectMemberReader interface {
	ListProjectMembers(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
}

type Service struct {
	identity identityReader
	projects projectMemberReader
}

func New(identity identityReader, projects projectMemberReader) (*Service, error) {
	if identity == nil || projects == nil {
		return nil, errors.New("MCP identity and project member applications are required")
	}
	return &Service{identity: identity, projects: projects}, nil
}

func (s *Service) Member(ctx context.Context, input *api.MemberListProjectOrMemberMe) (*api.MemberIdentityOrProjectMembers, error) {
	request, err := selectMemberAction(input)
	if err != nil {
		return nil, err
	}
	ctx, err = mcpauth.Revalidate(ctx)
	if err != nil {
		return nil, authenticationError(err)
	}
	slug := ""
	if request.workspace != nil {
		slug = *request.workspace
	}
	actor, err := mcpauth.SelectWorkspace(ctx, slug)
	if err != nil {
		return nil, api.MakeInvalidParams(errors.New("invalid workspace selector"))
	}
	if timezoneErr := djangosession.ValidateUserTimezone(actor.Timezone); timezoneErr != nil {
		return nil, memberReadFailure()
	}
	if request.action == "list_project" {
		members, readErr := s.projectMembers(ctx, actor, request.project)
		if readErr != nil {
			return nil, readErr
		}
		result := api.NewMemberIdentityOrProjectMembersProjectMembers(members)
		return &result, nil
	}
	identity, err := s.currentIdentity(ctx, actor)
	if err != nil {
		return nil, err
	}
	result := api.NewMemberIdentityOrProjectMembersMemberIdentity(identity)
	return &result, nil
}

type memberRequest struct {
	action, project string
	workspace       *string
}

func selectMemberAction(input *api.MemberListProjectOrMemberMe) (memberRequest, error) {
	if input != nil {
		if me, ok := input.AsMemberMe(); ok && me != nil && me.Action == "me" {
			return memberRequest{action: me.Action, workspace: me.WorkspaceSlug}, nil
		}
		if list, ok := input.AsMemberListProject(); ok && list != nil && list.Action == "list_project" {
			return memberRequest{action: list.Action, project: list.ProjectID, workspace: list.WorkspaceSlug}, nil
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
