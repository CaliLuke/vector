package mcpactions

import (
	"context"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
)

func (s *Service) DiscoverActions(ctx context.Context) (mcptransport.ActionVisibility, error) {
	ctx, err := mcpauth.Revalidate(ctx)
	if err != nil {
		return nil, err
	}
	visible := mcptransport.ActionVisibility{}
	actor, _ := mcpauth.PrincipalFromContext(ctx) // Revalidate returned a verified context.
	if !actor.AllowsAction(false) {
		return visible, nil
	}
	reads, err := s.directory.ReadCapabilities(ctx, uuid.UUID(actor.UserID), uuid.UUID(actor.WorkspaceID), actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	visible["member"] = map[string]bool{"me": true, "list_workspace": reads.WorkspaceMembers, "list_project": reads.ProjectMembers, "list_roles": reads.Workspace, "retrieve_role": reads.Workspace}
	visible["cycle"] = map[string]bool{"list": reads.ProjectMembers, "retrieve": reads.ProjectMembers}
	visible["project"] = map[string]bool{"list": reads.Workspace, "retrieve": reads.Workspace}
	if !actor.AllowsAction(true) || !reads.Workspace {
		return visible, nil
	}
	writes, err := s.projectWrites.VisibleProjectWrites(ctx, actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug)
	if err != nil {
		return nil, err
	}
	visible["project"]["archive"], visible["project"]["unarchive"] = writes.Archive, writes.Unarchive
	return visible, nil
}
