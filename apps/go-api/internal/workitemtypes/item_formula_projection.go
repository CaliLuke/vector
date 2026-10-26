package workitemtypes

import (
	"context"
	"maps"
	"time"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

// Called only after item/draft authorization, in the schema's coherent snapshot.
func loadItemFormulaInputs(ctx context.Context, tx pgx.Tx, workspace string, input ItemSchemaInput, projectReferencesVisible bool) (ItemFormulaInputs, error) {
	out := ItemFormulaInputs{Priority: "none"}
	table, id := "issues", input.IssueID
	if input.DraftID != "" {
		table, id = "draft_issues", input.DraftID
	}
	if id == "" {
		return out, nil
	}
	err := tx.QueryRow(ctx, `SELECT COALESCE(i.name,''),i.priority,COALESCE(i.start_date::text,''),COALESCE(i.target_date::text,''),
 (SELECT ep.value FROM estimate_points ep JOIN estimates e ON e.id=ep.estimate_id AND e.workspace_id=ep.workspace_id AND e.project_id=ep.project_id AND e.deleted_at IS NULL WHERE ep.id=i.estimate_point_id AND ep.workspace_id=i.workspace_id AND ep.project_id=i.project_id AND ep.deleted_at IS NULL AND $3)
 FROM `+table+` i WHERE i.id=$1 AND i.workspace_id=$2 AND i.deleted_at IS NULL`, id, workspace, projectReferencesVisible).Scan(&out.Name, &out.Priority, &out.StartDate, &out.DueDate, &out.Estimate)
	return out, err
}

func schemaFormulaValues(definitions []property.Definition, input ItemSchemaInput, typeID string, saved map[string]any) map[string]any {
	values := maps.Clone(saved)
	if values == nil {
		values = map[string]any{}
	}
	creating := input.Mode == "create" || input.Mode == "publish" || input.Mode == "draft" && input.DraftID == ""
	if !creating {
		return values
	}
	for _, def := range definitions {
		config, _, err := property.Effective(def, input.ProjectID, typeID)
		if err != nil || !config.Active {
			continue
		}
		if _, present := values[def.ID]; !present && config.Default != nil {
			values[def.ID] = config.Default
		}
	}
	return values
}

func schemaFormulas(ctx context.Context, tx pgx.Tx, workspace string, input ItemSchemaInput, identity schemaIdentity, projectVisible bool, definitions []property.Definition, projection *ItemProjectionPlan, out *ItemSchema) error {
	builtins, err := loadItemFormulaInputs(ctx, tx, workspace, input, projectVisible && identity.ProjectID == input.ProjectID)
	if err != nil {
		return err
	}
	// The same timestamp is returned with every result and shared by all formulas.
	if err = tx.QueryRow(ctx, `SELECT transaction_timestamp()`).Scan(&out.EvaluatedAt); err != nil {
		return err
	}
	out.EvaluatedAt = out.EvaluatedAt.UTC().Truncate(time.Microsecond)
	values := schemaFormulaValues(definitions, input, out.TypeID, identity.Values)
	builtins, values, err = previewFormulaInputs(ctx, tx, workspace, input, projectVisible, builtins, values, definitions, out.TypeID)
	if err != nil {
		return err
	}
	out.Computed = projection.Project(builtins, values, out.EvaluatedAt, input.Mode == "draft").Computed
	return nil
}
