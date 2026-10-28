package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

func (s *Service) milestoneMutation(ctx context.Context, input milestoneInput) (result *api.MilestoneResult, err error) {
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp.milestone."+input.action)
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, "milestone", input.action, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		if err == nil {
			id, _ := uuid.Parse(input.project)
			fields["committed_project_id"] = id.String()
			if result != nil {
				if receipt, ok := result.AsMilestoneMutationReceipt(); ok && receipt != nil {
					fields["committed_milestone_id"] = receipt.MilestoneID
					if receipt.Replayed != nil {
						fields["replayed"] = *receipt.Replayed
					}
				} else if record, ok := result.AsMilestoneRecord(); ok && record != nil {
					fields["committed_milestone_id"] = record.ID
				}
			} else {
				id, _ := uuid.Parse(input.target)
				fields["committed_milestone_id"] = id.String()
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	verified, current, authErr := actionActor(ctx, input.workspace, true)
	if authErr != nil {
		return nil, authErr
	}
	actor = current
	scope, id, err := milestoneScope(actor, input)
	if err != nil {
		return nil, err
	}
	options := milestones.WriteOptions{Scope: scope, ExpectedRevision: input.revision, Origin: s.origin, Revalidate: func(ctx context.Context) error {
		_, fresh, authErr := actionActor(ctx, input.workspace, true)
		if authErr != nil {
			return authErr
		}
		if fresh.UserID != actor.UserID || fresh.WorkspaceID != actor.WorkspaceID {
			return api.MakeForbidden(errors.New("authorized scope changed"))
		}
		return nil
	}}
	if input.request != nil {
		request, parseErr := milestoneUUID(*input.request)
		if parseErr != nil {
			return nil, parseErr
		}
		options.RequestID = &request
	}
	var value milestones.MutationResult
	switch input.action {
	case "create":
		value, err = s.milestones.Create(verified, options, input.fields)
	case "update":
		value, err = s.milestones.Update(verified, options, id, input.fields)
	case "delete":
		value, err = s.milestones.Delete(verified, options, id)
	case "manage_workitems":
		add, parseErr := milestoneIDs(input.add)
		if parseErr != nil {
			return nil, parseErr
		}
		remove, parseErr := milestoneIDs(input.remove)
		if parseErr != nil {
			return nil, parseErr
		}
		if len(add)+len(remove) == 0 {
			return nil, milestoneInvalid()
		}
		value, err = s.milestones.ManageWorkItems(verified, options, id, add, remove)
	}
	if err != nil {
		return nil, milestoneError(err)
	}
	return milestoneMutationResult(input, value)
}

func milestoneMutationResult(input milestoneInput, value milestones.MutationResult) (*api.MilestoneResult, error) {
	recordExpected := input.action == "create" || input.action == "update"
	if recordExpected != (value.Result != nil) {
		return nil, milestoneError(nil)
	}
	if input.request == nil && input.revision == nil {
		if !recordExpected {
			return nil, nil
		}
		result := api.NewMilestoneResultMilestoneRecord(milestoneRecord(*value.Result))
		return &result, nil
	}
	if _, err := milestoneUUID(value.MilestoneID); err != nil || value.Revision == "" {
		return nil, milestoneError(nil)
	}
	receipt := &api.MilestoneMutationReceipt{MilestoneID: value.MilestoneID, Revision: value.Revision, Result: loom.NullValue[api.MilestoneRecord]()}
	if recordExpected {
		receipt.Result = loom.NullableValue(*milestoneRecord(*value.Result))
	}
	if input.request != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return nil, milestoneError(nil)
		}
		request := value.RequestID.String()
		expires := value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &request, &expires, value.Replayed
	}
	result := api.NewMilestoneResultMilestoneMutationReceipt(receipt)
	return &result, nil
}

func milestoneIDs(input *string) ([]uuid.UUID, error) {
	if input == nil || strings.TrimSpace(*input) == "" {
		return nil, nil
	}
	value := strings.TrimSpace(*input)
	var parts []string
	if strings.HasPrefix(value, "[") {
		if json.Unmarshal([]byte(value), &parts) != nil {
			return nil, milestoneInvalid()
		}
	} else {
		for part := range strings.SplitSeq(value, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
	}
	ids := make([]uuid.UUID, len(parts))
	for i, part := range parts {
		id, err := milestoneUUID(part)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

func milestoneError(err error) error {
	switch {
	case errors.Is(err, milestones.ErrForbidden):
		return api.MakeForbidden(errors.New("milestone access denied"))
	case errors.Is(err, milestones.ErrNotFound):
		return api.MakeNotFound(errors.New("milestone unavailable"))
	case errors.Is(err, milestones.ErrInvalidInput), errors.Is(err, mutationreceipts.ErrInvalidRequest):
		return milestoneInvalid()
	case errors.Is(err, milestones.ErrConflict):
		return api.MakeConflict(errors.New("milestone conflict"))
	case errors.Is(err, milestones.ErrRevisionConflict):
		return api.MakeRevisionConflict(errors.New("milestone revision changed"))

	}
	if receiptErr := receiptOperationError(err); receiptErr != nil {
		return receiptErr
	}
	var failure *loom.ServiceError
	if errors.As(err, &failure) {
		return err
	}
	return api.MakeInternalError(errors.New("milestone operation failed"))
}
