// Package mcpstub supplies an explicit rejecting service for transport-only tests.
package mcpstub

import (
	"context"
	"errors"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

// Service rejects every tool call. Embed it and override only exercised actions.
type Service struct{}

var _ api.Service = Service{}

func (Service) Workspace(context.Context, *api.WorkspaceRequest) (result *api.WorkspaceResult, err error) {
	return result, errors.New("unexpected Workspace call in transport-only fixture")
}

func (Service) Member(context.Context, *api.MemberRequest) (result *api.MemberResult, err error) {
	return result, errors.New("unexpected Member call in transport-only fixture")
}

func (Service) Project(context.Context, *api.ProjectRequest) (result loom.Nullable[api.ProjectResult], err error) {
	return result, errors.New("unexpected Project call in transport-only fixture")
}

func (Service) Cycle(context.Context, *api.CycleRequest) (result loom.Nullable[api.CycleResult], err error) {
	return result, errors.New("unexpected Cycle call in transport-only fixture")
}

func (Service) Module(context.Context, *api.ModuleRequest) (result loom.Nullable[api.ModuleResult], err error) {
	return result, errors.New("unexpected Module call in transport-only fixture")
}

func (Service) Label(context.Context, *api.LabelRequest) (result loom.Nullable[api.LabelResult], err error) {
	return result, errors.New("unexpected Label call in transport-only fixture")
}

func (Service) WorkitemComment(context.Context, *api.WorkitemCommentRequest) (result loom.Nullable[api.WorkitemCommentResult], err error) {
	return result, errors.New("unexpected WorkitemComment call in transport-only fixture")
}

func (Service) WorkitemLink(context.Context, *api.WorkitemLinkRequest) (result loom.Nullable[api.WorkitemLinkResult], err error) {
	return result, errors.New("unexpected WorkitemLink call in transport-only fixture")
}

func (Service) WorkitemActivity(context.Context, *api.WorkitemActivityRequest) (result *api.WorkitemActivityResult, err error) {
	return result, errors.New("unexpected WorkitemActivity call in transport-only fixture")
}

func (Service) Milestone(context.Context, *api.MilestoneRequest) (result loom.Nullable[api.MilestoneResult], err error) {
	return result, errors.New("unexpected Milestone call in transport-only fixture")
}

func (Service) State(context.Context, *api.StateRequest) (result loom.Nullable[api.StateResult], err error) {
	return result, errors.New("unexpected State call in transport-only fixture")
}

func (Service) ProjectEstimate(context.Context, *api.ProjectEstimateRequest) (result loom.Nullable[api.ProjectEstimateResult], err error) {
	return result, errors.New("unexpected ProjectEstimate call in transport-only fixture")
}

func (Service) WorkitemType(context.Context, *api.WorkitemTypeRequest) (result loom.Nullable[api.WorkitemTypeResult], err error) {
	return result, errors.New("unexpected WorkitemType call in transport-only fixture")
}

func (Service) Workitem(context.Context, *api.WorkitemRequest) (result loom.Nullable[api.WorkitemResult], err error) {
	return result, errors.New("unexpected Workitem call in transport-only fixture")
}

func (Service) GetPqlReference(context.Context, *api.GetPqlReferencePayload) (result *api.GetPqlReferenceResult, err error) {
	return result, errors.New("unexpected GetPqlReference call in transport-only fixture")
}

func (Service) WorkitemAttachment(context.Context, *api.WorkitemAttachmentRequest) (result loom.Nullable[api.WorkitemAttachmentResult], err error) {
	return result, errors.New("unexpected WorkitemAttachment call in transport-only fixture")
}

func (Service) Customer(context.Context, *api.CustomerRequest) (result loom.Nullable[api.CustomerResult], err error) {
	return result, errors.New("unexpected Customer call in transport-only fixture")
}

func (Service) CustomerProperty(context.Context, *api.CustomerPropertyRequest) (result loom.Nullable[api.CustomerPropertyResult], err error) {
	return result, errors.New("unexpected CustomerProperty call in transport-only fixture")
}

func (Service) CustomerRequest(context.Context, *api.CustomerRequestRequest) (result loom.Nullable[api.CustomerRequestResult], err error) {
	return result, errors.New("unexpected CustomerRequest call in transport-only fixture")
}

func (Service) Template(context.Context, *api.TemplateRequest) (result loom.Nullable[api.TemplateResult], err error) {
	return result, errors.New("unexpected Template call in transport-only fixture")
}

func (Service) Initiative(context.Context, *api.InitiativeRequest) (result loom.Nullable[api.InitiativeResult], err error) {
	return result, errors.New("unexpected Initiative call in transport-only fixture")
}

func (Service) Page(context.Context, *api.PageRequest) (result loom.Nullable[api.PageResult], err error) {
	return result, errors.New("unexpected Page call in transport-only fixture")
}

func (Service) Collection(context.Context, *api.CollectionRequest) (result loom.Nullable[api.CollectionResult], err error) {
	return result, errors.New("unexpected Collection call in transport-only fixture")
}

func (Service) Release(context.Context, *api.ReleaseRequest) (result loom.Nullable[api.ReleaseResult], err error) {
	return result, errors.New("unexpected Release call in transport-only fixture")
}

func (Service) ReleaseLabel(context.Context, *api.ReleaseLabelRequest) (result loom.Nullable[api.ReleaseLabelResult], err error) {
	return result, errors.New("unexpected ReleaseLabel call in transport-only fixture")
}

func (Service) ReleaseTag(context.Context, *api.ReleaseTagRequest) (result loom.Nullable[api.ReleaseTagResult], err error) {
	return result, errors.New("unexpected ReleaseTag call in transport-only fixture")
}

func (Service) Intake(context.Context, *api.IntakeRequest) (result loom.Nullable[api.IntakeResult], err error) {
	return result, errors.New("unexpected Intake call in transport-only fixture")
}

func (Service) WorkLog(context.Context, *api.WorkLogRequest) (result loom.Nullable[api.WorkLogResult], err error) {
	return result, errors.New("unexpected WorkLog call in transport-only fixture")
}

func (Service) WorkitemRelation(context.Context, *api.WorkitemRelationRequest) (result loom.Nullable[api.WorkitemRelationResult], err error) {
	return result, errors.New("unexpected WorkitemRelation call in transport-only fixture")
}

func (Service) WorkitemProperty(context.Context, *api.WorkitemPropertyRequest) (result loom.Nullable[api.WorkitemPropertyResult], err error) {
	return result, errors.New("unexpected WorkitemProperty call in transport-only fixture")
}
