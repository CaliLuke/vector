package mcpactions

import (
	"context"
	"strconv"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/milestones"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) milestoneRead(ctx context.Context, input milestoneInput) (*api.MilestoneResult, error) {
	ctx, actor, err := actionActor(ctx, input.workspace, false)
	if err != nil {
		return nil, err
	}
	scope, id, err := milestoneScope(actor, input)
	if err != nil {
		return nil, err
	}
	if input.action == "retrieve" {
		record, readErr := s.milestones.Retrieve(ctx, scope, id, input.includeRevision)
		if readErr != nil {
			return nil, milestoneError(readErr)
		}
		if input.includeRevision && record.Revision == nil {
			return nil, milestoneError(nil)
		}
		result := api.NewMilestoneResultMilestoneRecord(milestoneRecord(record))
		return &result, nil
	}
	query, err := milestoneQuery(input)
	if err != nil {
		return nil, err
	}
	if input.action == "list" {
		page, readErr := s.milestones.List(ctx, scope, query)
		if readErr != nil {
			return nil, milestoneError(readErr)
		}
		result := milestonePage(page, func(v milestones.Record) *api.MilestonePageEntry {
			return &api.MilestonePageEntry{ID: loom.NullableValue(v.ID), Title: &v.Title, CreatedAt: &v.CreatedAt, UpdatedAt: &v.UpdatedAt, TargetDate: milestoneNullable(v.TargetDate), ExternalSource: milestoneNullable(v.ExternalSource), ExternalID: milestoneNullable(v.ExternalID)}
		})
		return &result, nil
	}
	page, readErr := s.milestones.ListWorkItems(ctx, scope, id, query)
	if readErr != nil {
		return nil, milestoneError(readErr)
	}
	result := milestonePage(page, func(v milestones.Link) *api.MilestonePageEntry {
		return &api.MilestonePageEntry{ID: loom.NullableValue(v.ID), Issue: &v.Issue, Milestone: &v.Milestone}
	})
	return &result, nil
}

func milestoneQuery(input milestoneInput) (djangoapi.Query, error) {
	size := 1000
	if input.perPage != nil && *input.perPage != 0 {
		size = *input.perPage
	}
	if size < 1 || size > 1000 {
		return djangoapi.Query{}, milestoneInvalid()
	}
	query := djangoapi.Query{Params: map[string]string{"per_page": strconv.Itoa(size)}, Present: map[string]bool{"per_page": true}}
	if input.cursor != nil && *input.cursor != "" {
		query.Params["cursor"], query.Present["cursor"] = *input.cursor, true
	}
	return query, nil
}

func milestonePage[T any](page milestones.Page[T], project func(T) *api.MilestonePageEntry) api.MilestoneResult {
	envelope := page.Envelope()
	result := &api.MilestonePage{Results: make([]*api.MilestonePageEntry, len(page.Results)), Count: envelope["count"].(int), TotalCount: envelope["total_count"].(int), NextCursor: envelope["next_cursor"].(string), PrevCursor: envelope["prev_cursor"].(string), NextPageResults: envelope["next_page_results"].(bool), PrevPageResults: envelope["prev_page_results"].(bool)}
	for i, row := range page.Results {
		result.Results[i] = project(row)
	}
	return api.NewMilestoneResultMilestonePage(result)
}
func milestoneNullable(value *string) loom.Nullable[string] {
	if value == nil {
		return loom.NullValue[string]()
	}
	return loom.NullableValue(*value)
}
func milestoneRecord(v milestones.Record) *api.MilestoneRecord {
	return &api.MilestoneRecord{ID: v.ID, Title: v.Title, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, TargetDate: milestoneNullable(v.TargetDate), ExternalSource: milestoneNullable(v.ExternalSource), ExternalID: milestoneNullable(v.ExternalID), Revision: v.Revision}
}
