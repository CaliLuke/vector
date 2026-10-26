package workitemtypes

import (
	"context"
	"slices"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

// A deleted type's explicit scope must disappear, never become an all-types
// context. Shared definitions and their values/history belong to the workspace.
func removeTypeConfiguration(ctx context.Context, tx pgx.Tx, workspace, actor, typeID string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM work_item_type_properties WHERE type_id=$1`, typeID); err != nil {
		return err
	}
	catalog, err := loadProperties(ctx, tx, workspace, actor, true)
	if err != nil {
		return err
	}
	for i, record := range catalog.Properties {
		def, changed := withoutTypeContexts(record.Definition, typeID)
		if !changed {
			continue
		}
		if err = saveProperty(ctx, tx, workspace, actor, PropertyInput{Action: "update", Definition: def}, catalog, true); err != nil {
			return err
		}
		catalog.Properties[i].Definition = def
	}
	return validatePropertyAttachments(catalog)
}

func withoutTypeContexts(def property.Definition, typeID string) (property.Definition, bool) {
	contexts := make([]property.Context, 0, len(def.Contexts))
	changed := false
	for _, scope := range def.Contexts {
		if slices.Contains(scope.Types, typeID) {
			changed = true
			scope.Types = slices.DeleteFunc(slices.Clone(scope.Types), func(id string) bool { return id == typeID })
			if len(scope.Types) == 0 {
				continue
			}
		}
		contexts = append(contexts, scope)
	}
	def.Contexts = contexts
	return def, changed
}
