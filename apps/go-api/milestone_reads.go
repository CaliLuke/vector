package systemapi

import (
	"context"

	api "github.com/CaliLuke/go-plane/milestones/gen/milestones"
)

func (s *milestoneService) List(ctx context.Context, p *api.ListPayload) (*api.MilestonePage, error) {
	scope, err := s.scope(ctx, p.Slug, p.ProjectID)
	if err != nil {
		return nil, err
	}
	page, err := s.application.List(ctx, scope, milestoneQuery(ctx, p.Cursor, p.PerPage))
	if err != nil {
		return nil, milestoneError(err)
	}
	return milestoneProjection[api.MilestonePage](page.Envelope())
}
func (s *milestoneService) Retrieve(ctx context.Context, p *api.RetrievePayload) (*api.MilestoneRecord, error) {
	scope, err := s.scope(ctx, p.Slug, p.ProjectID)
	if err != nil {
		return nil, err
	}
	id, err := milestoneUUID(p.MilestoneID)
	if err != nil {
		return nil, err
	}
	record, err := s.application.Retrieve(ctx, scope, id, p.IncludeRevision != nil && *p.IncludeRevision)
	if err != nil {
		return nil, milestoneError(err)
	}
	return milestoneProjection[api.MilestoneRecord](record)
}
func (s *milestoneService) ListWorkitems(ctx context.Context, p *api.ListWorkitemsPayload) (*api.MilestoneLinkPage, error) {
	scope, err := s.scope(ctx, p.Slug, p.ProjectID)
	if err != nil {
		return nil, err
	}
	id, err := milestoneUUID(p.MilestoneID)
	if err != nil {
		return nil, err
	}
	page, err := s.application.ListWorkItems(ctx, scope, id, milestoneQuery(ctx, p.Cursor, p.PerPage))
	if err != nil {
		return nil, milestoneError(err)
	}
	return milestoneProjection[api.MilestoneLinkPage](page.Envelope())
}
