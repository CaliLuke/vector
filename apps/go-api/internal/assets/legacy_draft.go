package assets

import (
	"context"
	"maps"

	"github.com/google/uuid"
)

func (repository *PostgresRepository) createLegacyDraftAsset(ctx context.Context, id, actor, draft uuid.UUID, fields map[string]any, key string) (map[string]any, error) {
	workspace, ok := fields["workspace"].(uuid.UUID)
	if !ok {
		return nil, notFoundAssetError()
	}
	for _, field := range []string{"user", "issue", "comment", "page"} {
		if legacyUUIDField(fields, field) != nil {
			return nil, notFoundAssetError()
		}
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	project, err := lockOwnedAssetDraft(ctx, tx, workspace, actor, draft)
	if err != nil {
		return nil, err
	}
	fields = maps.Clone(fields)
	fields["project"] = nil
	if project != nil {
		fields["project"] = *project
	}
	fields["entity_type"] = "DRAFT_ISSUE_DESCRIPTION"
	fields["entity_identifier"] = draft.String()
	writer := &PostgresRepository{database: tx}
	result, err := writer.insertLegacyFileAsset(ctx, id, actor, fields, key)
	if err != nil {
		return nil, err
	}
	return result, tx.Commit(ctx)
}
