package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) projectMembers(ctx context.Context, actor mcpauth.Principal, projectID string) (api.ProjectMembers, error) {
	project, err := uuid.Parse(projectID)
	if err != nil || actor.WorkspaceSlug == "" {
		return nil, api.MakeInvalidParams(errors.New("project and workspace are required"))
	}
	body, err := s.projects.ListProjectMembers(ctx, uuid.UUID(actor.UserID), actor.WorkspaceSlug, project)
	if err != nil {
		var failure *djangoapi.HTTPError
		if errors.As(err, &failure) && failure.Status == http.StatusForbidden {
			return nil, api.MakeForbidden(errors.New("project access denied"))
		}
		if errors.As(err, &failure) && (failure.Status == http.StatusNotFound || failure.Status == http.StatusBadRequest) {
			return nil, api.MakeNotFound(errors.New("project unavailable"))
		}
		return nil, memberReadFailure()
	}
	return decodeMemberRows(body)
}

func decodeMemberRows(body json.RawMessage) (api.ProjectMembers, error) {
	var rows []json.RawMessage
	if err := json.Unmarshal(body, &rows); err != nil || rows == nil {
		return nil, memberReadFailure()
	}
	members := make(api.ProjectMembers, 0, len(rows))
	for _, row := range rows {
		// ProjectMember extends UserLite in the pinned SDK. Properties absent
		// from the authoritative HTTP response remain null, including role data.
		member := &api.MemberIdentityWithRole{
			ID: loom.NullValue[string](), FirstName: loom.NullValue[string](),
			LastName: loom.NullValue[string](), Email: loom.NullValue[string](),
			Avatar: loom.NullValue[string](), AvatarURL: loom.NullValue[string](),
			DisplayName: loom.NullValue[string](), Role: loom.NullValue[int](),
			RoleSlug: loom.NullValue[string](), IsActive: loom.NullValue[bool](), IsBot: loom.NullValue[bool](),
		}
		if err := json.Unmarshal(row, &member); err != nil || member == nil {
			return nil, memberReadFailure()
		}
		members = append(members, member)
	}
	return members, nil
}
