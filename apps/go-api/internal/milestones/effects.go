package milestones

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func admitRecordEffect(ctx context.Context, tx pgx.Tx, opts WriteOptions, id uuid.UUID, changed, current map[string]any) error {
	return enqueueEffect(ctx, tx, "webhook", map[string]any{
		"name": "plane.bgtasks.webhook_task.model_activity", "args": []any{},
		"kwargs": map[string]any{"model_name": "milestone", "model_id": id.String(), "requested_data": changed, "current_instance": current, "actor_id": opts.ActorID.String(), "slug": opts.WorkspaceSlug, "workspace_id": opts.WorkspaceID.String(), "origin": opts.Origin},
	})
}

func admitDeletionEffect(ctx context.Context, tx pgx.Tx, opts WriteOptions, id uuid.UUID) error {
	return admitDirectWebhook(ctx, tx, opts, "milestone", "deleted", id, nil, nil, nil, nil, nil)
}

func admitLinkEffects(ctx context.Context, tx pgx.Tx, opts WriteOptions, milestone, item uuid.UUID, title string, add bool) error {
	var oldID, newID, oldTitle, newTitle any
	if add {
		newID, newTitle = milestone.String(), title
	} else {
		oldID, oldTitle = milestone.String(), title
	}
	if err := enqueueEffect(ctx, tx, "activity", map[string]any{
		"type": "issue.activity.updated", "issue_id": item.String(), "project_id": opts.ProjectID.String(), "actor_id": opts.ActorID.String(),
		"requested_data":   map[string]any{"milestone": newID, "milestone_title": newTitle},
		"current_instance": map[string]any{"milestone": oldID, "milestone_title": oldTitle},
		"notification":     true, "origin": opts.Origin, "epoch": time.Now().Unix(),
	}); err != nil {
		return err
	}
	return admitDirectWebhook(ctx, tx, opts, "issue", "updated", item, "milestone", oldTitle, newTitle, oldID, newID)
}

func admitDirectWebhook(ctx context.Context, tx pgx.Tx, opts WriteOptions, event, verb string, id uuid.UUID, field, oldValue, newValue, oldID, newID any) error {
	return enqueueEffect(ctx, tx, "webhook", map[string]any{
		"name": "plane.bgtasks.webhook_task.webhook_activity", "args": []any{},
		"kwargs": map[string]any{"event": event, "verb": verb, "event_id": id.String(), "field": field, "old_value": oldValue, "new_value": newValue, "old_identifier": oldID, "new_identifier": newID,
			"actor_id": opts.ActorID.String(), "slug": opts.WorkspaceSlug, "workspace_id": opts.WorkspaceID.String(), "current_site": opts.Origin},
	})
}

// The existing durable worker admits the committed row using this immutable ID.
// Any serialization or insertion error aborts the domain mutation and receipt.
func enqueueEffect(ctx context.Context, tx pgx.Tx, kind string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO work_item_effects(id,kind,payload) VALUES($1,$2,$3)`, uuid.NewString(), kind, raw)
	return err
}
