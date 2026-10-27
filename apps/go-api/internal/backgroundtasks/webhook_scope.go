package backgroundtasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

// New milestone producers always bind durable work to the immutable workspace.
// Legacy producer envelopes without this field retain their existing behavior.
func webhookWorkspaceID(kwargs map[string]json.RawMessage, event string, field json.RawMessage) (string, error) {
	var id string
	if raw, present := kwargs["workspace_id"]; present {
		if err := json.Unmarshal(raw, &id); err != nil || id == "" {
			return "", fmt.Errorf("%w: invalid workspace_id", ErrPermanentWebhookTask)
		}
	}
	return id, validateWebhookWorkspace(id, event, field)
}

func validateWebhookWorkspace(id, event string, field json.RawMessage) error {
	if id == "" && event != "milestone" && (event != "issue" || stringValue(field) != "milestone") {
		return nil
	}
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil {
		return fmt.Errorf("%w: workspace_id must be a nonzero UUID", ErrPermanentWebhookTask)
	}
	if _, ok := webhookModelTables[event]; !ok || event == "user" {
		return fmt.Errorf("%w: unsupported workspace model", ErrPermanentWebhookTask)
	}
	return nil
}

func webhookSubscriptionSQL(event string) string {
	switch event {
	case "project", "issue", "issue_comment", "milestone":
		return " AND h." + event + "=true"
	case "module", "module_issue":
		return " AND h.module=true"
	case "cycle", "cycle_issue":
		return " AND h.cycle=true"
	default:
		return "" // Preserve the legacy event policy outside the new model.
	}
}

func (store *PostgresWebhookTaskStore) webhookModelOwned(ctx context.Context, event, id, workspaceID string) (bool, error) {
	table, ok := webhookModelTables[event]
	if !ok || event == "user" {
		return false, fmt.Errorf("%w: unsupported workspace model", ErrPermanentWebhookTask)
	}
	// Deleted events still have a soft-deleted row whose immutable ownership is
	// available. A physically removed row cannot establish tenant ownership.
	var owned bool
	err := store.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id=$1 AND workspace_id=$2)`, id, workspaceID).Scan(&owned)
	return owned, err
}

// Deliberately project the public record, never to_jsonb of the storage row.
var milestoneWebhookProjectionSQL = `SELECT json_build_object(
 'id',model.id,'title',model.title,'target_date',model.target_date::text,
 'external_source',model.external_source,'external_id',model.external_id,
 'created_at',` + djangoapi.DjangoTimestampSQL("model.created_at") + `,
 'updated_at',` + djangoapi.DjangoTimestampSQL("model.updated_at") + `,
 'project',model.project_id,'workspace',model.workspace_id)
 FROM milestones model WHERE model.id=$1 AND model.deleted_at IS NULL`
