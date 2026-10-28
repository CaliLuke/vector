package projectstatereads

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func checkStateExternalIdentity(ctx context.Context, tx pgx.Tx, input StateWriteInput, fields map[string]any, currentID, currentSource *string) error {
	if !hasStateField(fields, "external_id") && !hasStateField(fields, "external_source") {
		return nil
	}
	id := effectiveStateIdentity(fields, "external_id", currentID)
	source := effectiveStateIdentity(fields, "external_source", currentSource)
	if id == "" || source == "" {
		return nil
	}
	var existing bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM states s JOIN workspaces w ON w.id=s.workspace_id
 WHERE s.project_id=$1 AND w.slug=$2 AND s.external_source=$3 AND s.external_id=$4
 AND s.deleted_at IS NULL AND s.id<>$5)`, input.ProjectID, input.Slug, source, id, input.StateID).Scan(&existing); err != nil {
		return err
	}
	if existing {
		return &WriteFailure{Status: 409, Body: map[string]any{"error": "State with the same external id and external source already exists", "id": input.StateID}}
	}
	return nil
}

func effectiveStateIdentity(fields map[string]any, key string, current *string) string {
	if value, present := fields[key]; present {
		text, _ := value.(string)
		return text
	}
	if current != nil {
		return *current
	}
	return ""
}
