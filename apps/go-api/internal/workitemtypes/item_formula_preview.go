package workitemtypes

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

// ItemFormulaPreview is transient editor input. It never updates item values,
// subscriptions, history, or notification state.
type ItemFormulaPreview struct {
	Name, Priority, StartDate, DueDate, EstimatePointID *string
	Values                                              map[string]any
}

type invalidFormulaInput struct{}

func previewFormulaInputs(ctx context.Context, tx pgx.Tx, workspace string, input ItemSchemaInput, projectVisible bool, builtins ItemFormulaInputs, values map[string]any, definitions []property.Definition, typeID string) (ItemFormulaInputs, map[string]any, error) {
	preview := input.Preview
	if preview == nil {
		return builtins, values, nil
	}
	for _, change := range []struct {
		value  *string
		target *string
	}{{preview.Name, &builtins.Name}, {preview.Priority, &builtins.Priority}, {preview.StartDate, &builtins.StartDate}, {preview.DueDate, &builtins.DueDate}} {
		if change.value != nil {
			*change.target = *change.value
		}
	}
	if preview.Name != nil {
		name, failures := djangoinput.TextField(*preview.Name, true, 255, nil)
		if len(failures) > 0 {
			return builtins, nil, reject(422, strings.Join(failures, " "))
		}
		builtins.Name = name
	}
	if preview.EstimatePointID != nil {
		builtins.Estimate = nil
		if *preview.EstimatePointID != "" {
			if !projectVisible || input.ProjectID == "" {
				return builtins, nil, reject(403, "Project access is required to select an estimate.")
			}
			err := tx.QueryRow(ctx, `SELECT ep.value FROM estimate_points ep JOIN estimates e ON e.id=ep.estimate_id AND e.workspace_id=ep.workspace_id AND e.project_id=ep.project_id AND e.deleted_at IS NULL WHERE ep.id=$1 AND ep.workspace_id=$2 AND ep.project_id=$3 AND ep.deleted_at IS NULL`, *preview.EstimatePointID, workspace, input.ProjectID).Scan(&builtins.Estimate)
			if errors.Is(err, pgx.ErrNoRows) {
				return builtins, nil, reject(422, "Select an available estimate from this project.")
			}
			if err != nil {
				return builtins, nil, err
			}
		}
	}
	definitionsByID := map[string]property.Definition{}
	for _, def := range definitions {
		definitionsByID[def.ID] = def
	}
	values = maps.Clone(values)
	for id, value := range preview.Values {
		def, exists := definitionsByID[id]
		if !exists {
			return builtins, nil, reject(422, "A preview value is not attached to this type.")
		}
		config, _, err := property.Effective(def, input.ProjectID, typeID)
		if err != nil {
			return builtins, nil, err
		}
		if input.Mode == "draft" && def.Kind == property.Cascading {
			return builtins, nil, reject(422, "Cascading properties are unavailable on drafts.")
		}
		if !config.Active || def.Kind == property.Formula || def.Kind == property.Text && config.Format == "Read-only" {
			return builtins, nil, reject(422, "Inactive and read-only properties cannot be edited.")
		}
		// Input errors are evaluated only if a formula references that field. Keeping
		// invalid typed input here preserves lazy IF and does not turn it into zero.
		if slices.Contains([]property.Kind{property.Text, property.Number, property.Boolean, property.Date}, def.Kind) {
			canonical, validationErr := property.ValidateValue(def.Kind, config, value, true, nil)
			if validationErr != nil {
				values[id] = invalidFormulaInput{}
			} else {
				values[id] = canonical
			}
		} else {
			values[id] = value
		}
	}
	return builtins, values, nil
}
