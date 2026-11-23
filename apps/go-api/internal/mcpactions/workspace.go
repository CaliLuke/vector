package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) workspaceIdentity(ctx context.Context, input *api.WorkspaceRetrieve) (*api.WorkspaceRecord, error) {
	if input == nil || input.Action != "retrieve" {
		return nil, api.MakeInvalidParams(errors.New("invalid workspace action"))
	}
	ctx, actor, err := s.scopedActor(ctx, input.WorkspaceSlug, false)
	if err != nil {
		return nil, err
	}
	result := &api.WorkspaceRecord{Slug: actor.WorkspaceSlug, ID: loom.NullValue[string](), Name: loom.NullValue[string](), ConnectedVia: "api_key_header"}
	if actor.CredentialClass() == "oauth" {
		result.ConnectedVia = "oauth"
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
	if _, ok := input.AsWorkspaceList(); ok {
		return s.listWorkspaces(ctx)
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

func (s *Service) listWorkspaces(ctx context.Context) (*api.WorkspaceResult, error) {
	ctx, actor, err := readActor(ctx, nil)
	if err != nil {
		return nil, err
	}
	workspaces, err := s.directory.ListWorkspaces(ctx, uuid.UUID(actor.UserID))
	if err != nil {
		return nil, api.MakeInternalError(errors.New("workspace list failed"))
	}
	records := make([]*api.AccessibleWorkspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		records = append(records, &api.AccessibleWorkspace{ID: workspace.ID, Name: workspace.Name, Slug: workspace.Slug})
	}
	result := api.NewWorkspaceResultWorkspaceListResult(&api.WorkspaceListResult{Results: records})
	return &result, nil
}

func (s *Service) scopedReadActor(ctx context.Context, workspace *string) (context.Context, mcpauth.Principal, error) {
	return s.scopedActor(ctx, workspace, false)
}

func (s *Service) scopedActor(ctx context.Context, workspace *string, mutation bool) (context.Context, mcpauth.Principal, error) {
	ctx, actor, err := actionActor(ctx, workspace, mutation)
	if err != nil || actor.WorkspaceSlug != "" {
		return ctx, actor, err
	}
	workspaces, err := s.directory.ListWorkspaces(ctx, uuid.UUID(actor.UserID))
	if err != nil {
		return nil, mcpauth.Principal{}, api.MakeInternalError(errors.New("workspace list failed"))
	}
	failure := api.MakeInvalidParams(errors.New("workspace selection required"))
	remedy := &loom.ErrorRemedy{SafeMessage: "No accessible workspaces."}
	if len(workspaces) > 0 {
		var choices strings.Builder
		choices.WriteString("Choose a workspace and retry with workspace_slug.")
		for _, workspace := range workspaces {
			choices.WriteString("\n")
			choices.WriteString(workspace.Slug)
			if workspace.Name != workspace.Slug {
				choices.WriteString("  ")
				choices.WriteString(strconv.Quote(workspace.Name))
			}
		}
		remedy.SafeMessage = choices.String()
	}
	return nil, mcpauth.Principal{}, loom.WithErrorRemedy(failure, remedy)
}
