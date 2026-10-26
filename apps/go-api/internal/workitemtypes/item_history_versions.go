package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

func recordRichTextChange(ctx context.Context, tx pgx.Tx, actor string, before *ItemHistoryState, after ItemHistoryState, def property.Definition, old, next any, source string) error {
	same, err := equalHistoryValues(def, old, next)
	if err != nil {
		return err
	}
	// A preservation snapshot already captures an unchanged former value. Do
	// not also reconstruct a second identical baseline in that operation.
	if before != nil && nonemptyPropertyValue(old) && (source != "preservation" || !same) {
		var latest []byte
		err := tx.QueryRow(ctx, `SELECT value FROM work_item_property_versions WHERE workspace_id=$1 AND issue_id=$2 AND property_id=$3 ORDER BY created_at DESC,id DESC LIMIT 1`, after.WorkspaceID, after.ItemID, def.ID).Scan(&latest)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var previous any
		if len(latest) > 0 {
			if err = json.Unmarshal(latest, &previous); err != nil {
				return err
			}
		}
		equal, err := equalHistoryValues(def, previous, old)
		if err != nil {
			return err
		}
		if !equal {
			// Legacy/preservation-only history may not contain the value being replaced.
			// Capture it honestly as a baseline with unknown author, not as this edit.
			if err = insertPropertyVersion(ctx, tx, "", *before, def, old, "baseline"); err != nil {
				return err
			}
		}
	}
	return insertPropertyVersion(ctx, tx, actor, after, def, next, source)
}

// RecordPreservedProperty is used by schema removals, which have already
// authorized and rendered their value-preservation plan under schema locks.
// Permanent removal must never call it.
func RecordPreservedProperty(ctx context.Context, tx pgx.Tx, actor string, state ItemHistoryState, entry PreservedItemProperty) error {
	if entry.Definition.Kind != property.RichText {
		return nil
	}
	return insertPropertyVersion(ctx, tx, actor, state, entry.Definition, entry.Value, "preservation")
}

func insertPropertyVersion(ctx context.Context, tx pgx.Tx, actor string, state ItemHistoryState, def property.Definition, value any, source string) error {
	def = historyDefinition(state, def)
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	text, err := itemHistoryValueText(ctx, tx, &state, def, value)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO work_item_property_versions(id,workspace_id,issue_id,property_id,value,search_text,actor_id,property_title,source)
 VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8)`, state.WorkspaceID, state.ItemID, def.ID, raw, text, actor, def.Title, source)
	return err
}
