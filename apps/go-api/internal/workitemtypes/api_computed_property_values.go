package workitemtypes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Computed values have a stable, derived identity, not a persisted value row.
// This function is called only after LoadPropertyValueScope authorizes the item
// and its active property, inside the same database snapshot.
func projectComputedPropertyValueAPI(ctx context.Context, tx pgx.Tx, q PropertyValueAPIQuery, scope PropertyValueScope) (json.RawMessage, error) {
	var value any = scope.Configuration.Content
	if scope.Definition.Kind == property.Formula {
		definitions, err := itemPropertyDefinitions(ctx, tx, q.WorkspaceID.String(), scope.typeID)
		if err != nil {
			return nil, err
		}
		inputs, err := loadItemFormulaInputs(ctx, tx, q.WorkspaceID.String(), ItemSchemaInput{ProjectID: q.ProjectID.String(), IssueID: q.IssueID.String(), Mode: "edit"}, true)
		if err != nil {
			return nil, err
		}
		var instant time.Time
		if err = tx.QueryRow(ctx, `SELECT transaction_timestamp()`).Scan(&instant); err != nil {
			return nil, err
		}
		result, exists := EvaluateItemFormulas(definitions, q.ProjectID.String(), scope.typeID, inputs, scope.stored, instant)[q.PropertyID.String()]
		if !exists || result.Status == "error" {
			return nil, reject(422, "The property formula could not be evaluated.")
		}
		value = result.Value
	}
	kind, _ := apiKind(scope.Definition.Kind)
	id := uuid.NewSHA1(q.IssueID, []byte("vector:computed-property:"+q.PropertyID.String()))
	record := map[string]any{"id": id.String(), "property_id": q.PropertyID.String(), "issue_id": q.IssueID.String(), "value": value, "value_type": kind, "value_detail": nil, "external_source": nil, "external_id": nil, "created_at": nil, "updated_at": nil}
	if q.IncludeRevision {
		// A computed value can change with the catalogue or clock, without an
		// item write. Its read-only token therefore fingerprints the projection.
		raw, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		record["revision"] = fmt.Sprintf("workitem_property_computed:%x", sha256.Sum256(raw))
	}
	return json.Marshal(record)
}
