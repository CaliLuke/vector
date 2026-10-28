package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) workspaceIdentity(ctx context.Context, input *api.WorkspaceRetrieve) (*api.WorkspaceRecord, error) {
	if input == nil || input.Action != "retrieve" {
		return nil, api.MakeInvalidParams(errors.New("invalid workspace action"))
	}
	ctx, actor, err := actionActor(ctx, input.WorkspaceSlug, false)
	if err != nil {
		return nil, err
	}
	result := &api.WorkspaceRecord{Slug: actor.WorkspaceSlug, ID: loom.NullValue[string](), Name: loom.NullValue[string](), ConnectedVia: "api_key_header"}
	if actor.CredentialClass() == "oauth" {
		result.ConnectedVia = "oauth"
	}
	if actor.WorkspaceSlug == "" {
		return result, nil
	}
	identity, err := s.directory.WorkspaceIdentity(ctx, uuid.UUID(actor.UserID), uuid.UUID(actor.WorkspaceID), actor.WorkspaceSlug)
	if errors.Is(err, membership.ErrForbidden) {
		return nil, api.MakeForbidden(errors.New("workspace access denied"))
	}
	if err != nil {
		return nil, api.MakeInternalError(errors.New("workspace lookup failed"))
	}
	result.Slug = identity.Slug
	if result.ConnectedVia == "oauth" {
		result.ID, result.Name = loom.NullableValue(identity.ID), loom.NullableValue(identity.Name)
	}
	return result, nil
}

func (s *Service) Workspace(ctx context.Context, input *api.WorkspaceRequest) (*api.WorkspaceResult, error) {
	if input == nil || input.Validate() != nil {
		return nil, api.MakeInvalidParams(errors.New("invalid workspace input"))
	}
	if selected, ok := input.AsWorkspaceRetrieve(); ok {
		record, err := s.workspaceIdentity(ctx, selected)
		if err != nil {
			return nil, err
		}
		result := api.NewWorkspaceResultWorkspaceRecord(record)
		return &result, nil
	}
	raw, err := s.featureOperation(ctx, input)
	if err != nil {
		return nil, err
	}
	var result api.WorkspaceResult
	if json.Unmarshal(raw, &result) != nil {
		return nil, workitemResultError()
	}
	return &result, nil
}
