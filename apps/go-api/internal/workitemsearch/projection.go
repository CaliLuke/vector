// Package workitemsearch owns the sanitized current-value search projection.
// History is deliberately separate: a saved revision is not a current value.
package workitemsearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

type fieldValue struct {
	id    string
	value any
}

// Refresh must share the canonical item's transaction and schema/item locks.
// Index every known saved rich-text value, even if its context or rollout is off;
// attachment and effective configuration are checked by the search query.
func Refresh(ctx context.Context, tx pgx.Tx, workspace, item string) error {
	values, err := readValues(ctx, tx, workspace, item)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM work_item_property_search WHERE issue_id=$1 AND workspace_id=$2`, item, workspace); err != nil {
		return err
	}
	for _, field := range values {
		text, hash, err := project(field.value)
		if err != nil {
			return fmt.Errorf("index work item %s property %s: %w", item, field.id, err)
		}
		if text == "" {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO work_item_property_search(issue_id,property_id,workspace_id,source_hash,search_text) VALUES($1,$2,$3,$4,$5)`, item, field.id, workspace, hash, text); err != nil {
			return err
		}
	}
	return nil
}

func readValues(ctx context.Context, tx pgx.Tx, workspace, item string) ([]fieldValue, error) {
	rows, err := tx.Query(ctx, `SELECT p.id::text,v.value FROM issues i CROSS JOIN LATERAL jsonb_each(i.custom_properties) v
 JOIN work_item_properties p ON p.id::text=v.key AND p.workspace_id=i.workspace_id
 WHERE i.id=$1 AND i.workspace_id=$2 AND NOT i.is_draft AND p.definition->>'kind'='Rich text' AND v.value<>'null'::jsonb ORDER BY p.id`, item, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []fieldValue
	for rows.Next() {
		var entry fieldValue
		var raw []byte
		if err = rows.Scan(&entry.id, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &entry.value); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func project(value any) (string, string, error) {
	saved, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("stored rich text is not a string")
	}
	text, err := property.ValueText(property.Definition{Kind: property.RichText}, property.Configuration{}, saved, nil)
	if err != nil {
		return "", "", fmt.Errorf("stored rich text cannot be sanitized")
	}
	digest := sha256.Sum256([]byte(saved))
	return text, hex.EncodeToString(digest[:]), nil
}
