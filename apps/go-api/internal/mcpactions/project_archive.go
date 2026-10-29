package mcpactions

import (
	"context"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

type projectWriter interface {
	resourceWriter
	ArchiveProject(context.Context, projects.ArchiveCommand) (projects.ArchiveResult, error)
	VisibleProjectWrites(context.Context, uuid.UUID, uuid.UUID, string) (projects.ProjectWriteVisibility, error)
}

func (s *Service) archiveProject(ctx context.Context, actor mcpauth.Principal, input *api.ProjectArchive) (*api.ProjectResult, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil || id == uuid.Nil {
		return nil, api.MakeInvalidParams(errors.New("invalid project identifier"))
	}
	var request *uuid.UUID
	if input.RequestID != nil {
		parsed, parseErr := uuid.Parse(*input.RequestID)
		if parseErr != nil || parsed == uuid.Nil {
			return nil, api.MakeInvalidParams(errors.New("invalid request identifier"))
		}
		request = &parsed
	}
	result, err := s.projectWrites.ArchiveProject(ctx, projects.ArchiveCommand{
		ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug, ProjectID: id,
		Archive: input.Action == "archive", RequestID: request, ExpectedRevision: input.ExpectedRevision,
		Revalidate: mutationRevalidator(actor, input.WorkspaceSlug, false),
	})
	if err != nil {
		return nil, projectWriteError(err)
	}
	if input.RequestID == nil && input.ExpectedRevision == nil {
		return nil, nil
	}
	receipt := &api.ProjectMutationReceipt{ProjectID: id.String(), Revision: result.Revision}
	if input.RequestID != nil {
		canonicalRequestID := request.String()
		receipt.RequestID = &canonicalRequestID
		if result.Receipt.ReplayExpiresAt == nil {
			return nil, api.MakeInternalError(errors.New("missing committed receipt expiry"))
		}
		expires := result.Receipt.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.ReplayExpiresAt = &expires
		receipt.Replayed = &result.Receipt.Replayed
	}
	projected := api.NewProjectResultProjectMutationReceipt(receipt)
	return &projected, nil
}

func projectWriteError(err error) error {
	switch {
	case errors.Is(err, mutationreceipts.ErrConflict):
		return api.MakeRequestConflict(errors.New("request input changed"))
	case errors.Is(err, projects.ErrRevisionConflict):
		return api.MakeRevisionConflict(errors.New("project revision changed"))
	case errors.Is(err, mutationreceipts.ErrIndeterminate):
		return api.MakeIndeterminate(errors.New("unacknowledged commit"))
	}
	// Preserve safe errors from live credential revalidation without turning them
	// into database errors or exposing their underlying credential failures.
	var serviceError *loom.ServiceError
	if errors.As(err, &serviceError) {
		return err
	}

	return projectReadError(err)
}
