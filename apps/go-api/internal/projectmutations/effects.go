package projectmutations

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Effects records the existing native webhook/deletion payloads for the existing
// work-item effect worker. It never dispatches externally before commit.
type Effects struct{ Tx pgx.Tx }

func (e Effects) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if name == "plane.bgtasks.page_transaction_task.page_transaction" {
		return e.insert(ctx, "page_transaction", kwargs)
	}
	if name == "plane.bgtasks.page_version_task.track_page_version" {
		return e.insert(ctx, "page_version", kwargs)
	}
	if name == "plane.bgtasks.work_item_link_task.crawl_work_item_link_title" {
		if len(args) != 2 {
			return errors.New("link preview requires link and URL")
		}
		return e.insert(ctx, "link_preview", map[string]any{"id": args[0], "url": args[1]})
	}
	if name == "plane.bgtasks.issue_activities_task.issue_activity" {
		return e.insert(ctx, "activity", kwargs)
	}
	if !strings.HasPrefix(name, "plane.bgtasks.webhook_task.") {
		return errors.New("unsupported transactional task")
	}
	return e.insert(ctx, "webhook", map[string]any{"name": name, "args": args, "kwargs": kwargs})
}
func (e Effects) SoftDelete(ctx context.Context, app, model, id string) error {
	return e.insert(ctx, "soft_delete", map[string]any{"app": app, "model": model, "id": id})
}
func (e Effects) insert(ctx context.Context, kind string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = e.Tx.Exec(ctx, `INSERT INTO work_item_effects(id,kind,payload) VALUES($1,$2,$3)`, uuid.New(), kind, raw)
	return err
}

// RelationActivity captures immutable endpoint identity in the transaction outbox.
type RelationActivity struct {
	WorkspaceID, ProjectID, ItemID, ActorID, RelatedID uuid.UUID
	Label, Identifier, Verb                            string
}

func (e Effects) RelationActivity(ctx context.Context, relation RelationActivity) error {
	return e.Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", nil, map[string]any{
		"type":         "workitem_relation.activity." + relation.Verb,
		"workspace_id": relation.WorkspaceID.String(), "project_id": relation.ProjectID.String(),
		"issue_id": relation.ItemID.String(), "actor_id": relation.ActorID.String(),
		"epoch": time.Now().Unix(), "notification": false,
		"requested_data": map[string]any{"related_id": relation.RelatedID.String(), "label": relation.Label, "identifier": relation.Identifier},
	})
}
