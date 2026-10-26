package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

// ItemHistoryState is captured from a canonical published item while its schema
// and item locks are held. Definitions retain former contexts across migrations.
// Private draft saves never enter this published history boundary.
type ItemHistoryState struct {
	WorkspaceID, ProjectID, ItemID, TypeID string
	RestoredPropertyID                     string
	Values                                 map[string]any
	Definitions                            []property.Definition
}

func LoadItemHistoryState(ctx context.Context, tx pgx.Tx, state ItemHistoryState) (ItemHistoryState, error) {

	raw, err := json.Marshal(state.Values)
	if err != nil {
		return state, err
	}
	state.Values = nil
	if err = json.Unmarshal(raw, &state.Values); err != nil {
		return state, err
	}
	if state.TypeID == "" {
		return state, nil
	}
	definitions, err := itemPropertyDefinitions(ctx, tx, state.WorkspaceID, state.TypeID)
	state.Definitions = definitions
	return state, err
}

// RecordItemPropertyChanges belongs to the same transaction as the item write.
// It records canonical changes, not requested patches or later worker snapshots.
// Permanent removal must instead retain history unchanged and emit only its
// existing value-free removal notice; it must not call this content recorder.
func RecordItemPropertyChanges(ctx context.Context, tx pgx.Tx, actor string, before *ItemHistoryState, after ItemHistoryState, preserved ...PreservedItemProperty) error {
	if tx == nil {
		return errors.New("item history requires a transaction")
	}
	definitions := map[string]property.Definition{}
	if before != nil {
		if before.ItemID != after.ItemID || before.WorkspaceID != after.WorkspaceID {
			return errors.New("item history scope changed")
		}
		for _, def := range before.Definitions {
			definitions[def.ID] = def
		}
	}
	for _, def := range after.Definitions {
		definitions[def.ID] = def
	}
	preservation := map[string]PreservedItemProperty{}
	for _, entry := range preserved {
		definitions[entry.Definition.ID] = entry.Definition
		preservation[entry.Definition.ID] = entry
	}
	ids := make([]string, 0, len(definitions))
	for id := range definitions {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		def := definitions[id]
		if def.Kind == property.Formula || def.Kind == property.Text && def.Configuration.Format == "Read-only" {
			continue
		}
		var old any
		if before != nil {
			old = before.Values[id]
		}
		next := after.Values[id]
		entry, keeping := preservation[id]
		equal, err := equalHistoryValues(def, old, next)
		if err != nil {
			return err
		}
		if equal && !keeping {
			continue
		}
		previousText, err := itemHistoryValueText(ctx, tx, before, def, old)
		if err != nil {
			return err
		}
		nextText, err := itemHistoryValueText(ctx, tx, &after, def, next)
		if err != nil {
			return err
		}
		if keeping {
			// The transition planner already authorized (or redacted) reference
			// labels. Do not resolve hidden source-project labels again here.
			nextText, err = property.ValueText(property.Definition{Kind: property.RichText}, property.Configuration{}, entry.HTML, nil)
			if err != nil {
				return err
			}
			nextText = "Preserved in description: " + nextText
		}
		verb := "updated"
		if id == after.RestoredPropertyID {
			verb = "restored"
		}
		if _, err = tx.Exec(ctx, `INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,actor_id,verb,field,old_identifier,new_identifier,old_value,new_value,comment,attachments)
 VALUES(gen_random_uuid(),clock_timestamp(),clock_timestamp(),$1,$2,$3,$4,$9,'custom_property',$5,$5,$6,$7,$8,ARRAY[]::text[])`, after.WorkspaceID, after.ProjectID, after.ItemID, actor, id, previousText, nextText, def.Title, verb); err != nil {
			return err
		}
		if def.Kind == property.RichText {
			source := "change"
			if id == after.RestoredPropertyID {
				source = "restore"
			}
			if keeping {
				next, source = entry.Value, "preservation"
			}
			if err = recordRichTextChange(ctx, tx, actor, before, after, def, old, next, source); err != nil {
				return err
			}
		}
	}
	return recordItemTypeChange(ctx, tx, actor, before, after)
}

func itemHistoryValueText(ctx context.Context, tx pgx.Tx, state *ItemHistoryState, def property.Definition, value any) (string, error) {
	if state == nil || value == nil {
		return "", nil
	}
	def = historyDefinition(*state, def)
	config, _, err := property.Effective(def, state.ProjectID, state.TypeID)
	if err != nil {
		return "", err
	}
	labels, err := preservationLabels(ctx, tx, state.WorkspaceID, state.ProjectID, def.Kind, value)
	if err != nil {
		return "", err
	}
	return property.ValueText(def, config, value, labels)
}

func equalHistoryValues(def property.Definition, left, right any) (bool, error) {
	canonical := func(value any) ([]byte, error) {
		if values, ok := value.([]any); ok && len(values) == 0 {
			value = nil
		}
		if values, ok := value.([]string); ok && len(values) == 0 {
			value = nil
		}
		if text, ok := value.(string); ok && text == "" {
			value = nil
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		// Persisted and validated values may have different Go numeric representations.
		var normalized any
		if err = json.Unmarshal(raw, &normalized); err != nil {
			return nil, err
		}
		if def.Configuration.Multiple && (def.Kind == property.Dropdown || def.Kind == property.Member || def.Kind == property.Release) {
			if values, ok := normalized.([]any); ok {
				slices.SortFunc(values, func(a, b any) int {
					x, _ := a.(string)
					y, _ := b.(string)
					if x < y {
						return -1
					}
					if x > y {
						return 1
					}
					return 0
				})
			}
		}
		return json.Marshal(normalized)
	}
	a, err := canonical(left)
	if err != nil {
		return false, err
	}
	b, err := canonical(right)
	return string(a) == string(b), err
}

func historyDefinition(state ItemHistoryState, fallback property.Definition) property.Definition {
	for _, candidate := range state.Definitions {
		if candidate.ID == fallback.ID {
			return candidate
		}
	}
	return fallback
}
