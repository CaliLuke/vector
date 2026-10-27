package systemapi

import (
	"context"

	api "github.com/CaliLuke/go-plane/milestones/gen/milestones"
	"github.com/google/uuid"
)

func (s *milestoneService) Create(ctx context.Context, p *api.CreatePayload) (*api.MilestoneWriteResult, error) {
	if p.Body == nil {
		return nil, milestoneFailure("invalid_input", "Milestone body is required.")
	}
	body := p.Body
	opts, err := s.writeOptions(ctx, p.Slug, p.ProjectID, body.RequestID, body.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	result, err := s.application.Create(ctx, opts, milestoneFields(body.Title, body.TargetDate, body.ExternalSource, body.ExternalID))
	return milestoneWriteResult(result, body.RequestID != nil || body.ExpectedRevision != nil, err)
}
func (s *milestoneService) Update(ctx context.Context, p *api.UpdatePayload) (*api.MilestoneWriteResult, error) {
	if p.Body == nil {
		return nil, milestoneFailure("invalid_input", "Milestone body is required.")
	}
	body := p.Body
	opts, err := s.writeOptions(ctx, p.Slug, p.ProjectID, body.RequestID, body.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	id, err := milestoneUUID(p.MilestoneID)
	if err != nil {
		return nil, err
	}
	result, err := s.application.Update(ctx, opts, id, milestoneFields(body.Title, body.TargetDate, body.ExternalSource, body.ExternalID))
	return milestoneWriteResult(result, body.RequestID != nil || body.ExpectedRevision != nil, err)
}
func (s *milestoneService) Delete(ctx context.Context, p *api.DeletePayload) (*api.MilestoneEmptyResult, error) {
	opts, err := s.writeOptions(ctx, p.Slug, p.ProjectID, p.RequestID, p.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	id, err := milestoneUUID(p.MilestoneID)
	if err != nil {
		return nil, err
	}
	result, err := s.application.Delete(ctx, opts, id)
	return milestoneEmptyResult(result, p.RequestID != nil || p.ExpectedRevision != nil, err)
}
func (s *milestoneService) AddWorkitems(ctx context.Context, p *api.AddWorkitemsPayload) (*api.MilestoneEmptyResult, error) {
	return s.manageMilestoneItems(ctx, p.Slug, p.ProjectID, p.MilestoneID, p.Body, true)
}
func (s *milestoneService) RemoveWorkitems(ctx context.Context, p *api.RemoveWorkitemsPayload) (*api.MilestoneEmptyResult, error) {
	return s.manageMilestoneItems(ctx, p.Slug, p.ProjectID, p.MilestoneID, p.Body, false)
}
func (s *milestoneService) manageMilestoneItems(ctx context.Context, slug, project, milestone string, body *api.MilestoneIssues, add bool) (*api.MilestoneEmptyResult, error) {
	if body == nil {
		return nil, milestoneFailure("invalid_input", "Milestone body is required.")
	}
	opts, err := s.writeOptions(ctx, slug, project, body.RequestID, body.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	id, err := milestoneUUID(milestone)
	if err != nil {
		return nil, err
	}
	items := make([]uuid.UUID, len(body.Issues))
	for index, value := range body.Issues {
		items[index], err = milestoneUUID(value)
		if err != nil {
			return nil, err
		}
	}
	var additions, removals []uuid.UUID
	if add {
		additions = items
	} else {
		removals = items
	}
	result, err := s.application.ManageWorkItems(ctx, opts, id, additions, removals)
	return milestoneEmptyResult(result, body.RequestID != nil || body.ExpectedRevision != nil, err)
}
