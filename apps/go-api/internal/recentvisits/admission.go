// Package recentvisits owns read-side visit admission and its publisher fallback.
package recentvisits

import (
	"context"

	"github.com/google/uuid"
)

type Admitter interface {
	AdmitRecentVisit(context.Context, string, string, string, string, string, string) error
}

type Publisher interface {
	Task(context.Context, string, []any, map[string]any) error
}

type Visit struct {
	Slug, Entity, EntityID, UserID, ProjectID string
}

func Admit(ctx context.Context, durable Admitter, fallback Publisher, visit Visit) error {
	if durable != nil {
		return durable.AdmitRecentVisit(ctx, uuid.NewString(), visit.Slug, visit.Entity, visit.EntityID, visit.UserID, visit.ProjectID)
	}
	if fallback == nil {
		return nil
	}
	return fallback.Task(ctx, "plane.bgtasks.recent_visited_task.recent_visited_task", []any{}, map[string]any{
		"slug": visit.Slug, "entity_name": visit.Entity, "entity_identifier": visit.EntityID, "user_id": visit.UserID, "project_id": visit.ProjectID,
	})
}
