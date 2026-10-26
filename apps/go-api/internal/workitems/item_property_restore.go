package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func normalizePropertyRestore(input *ItemEditInput) error {
	if input.Restore == nil {
		return nil
	}
	if input.Mode != "edit" || len(input.Changes) != 0 || input.ExpectedSourceVersion == "" {
		return failure(400, "error", "Restore one property revision on a saved work item without other changes.")
	}
	field, fieldErr := uuid.Parse(input.Restore.PropertyID)
	version, versionErr := uuid.Parse(input.Restore.VersionID)
	if fieldErr != nil || versionErr != nil {
		return failure(400, "error", "Choose a valid property revision.")
	}
	input.Restore = &ItemPropertyRestore{PropertyID: field.String(), VersionID: version.String()}
	return nil
}

// Called only after the owning item and workspace schema are locked and the
// caller's source token is checked. The receipt binds IDs, never a client copy
// of historical content. The ordinary reviewed writer validates the result and
// all current asset bindings before committing the new revision.
func (r *Repository) resolvePropertyRestore(ctx context.Context, q Request, command itemEditCommand, source map[string]any) (any, error) {
	tx, ok := r.db.(pgx.Tx)
	if !ok {
		return nil, errors.New("property restore requires a transaction")
	}
	state, err := loadModelHistory(ctx, tx, source)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, requiredObject()
	}
	var definition *property.Definition
	for _, candidate := range state.Definitions {
		if candidate.ID == command.Restore.PropertyID && candidate.Kind == property.RichText {
			definition = &candidate
			break
		}
	}
	if definition == nil {
		return nil, failure(409, "error", "This field is no longer available on the work item. Its history is still readable.")
	}
	config, _, err := property.Effective(*definition, state.ProjectID, state.TypeID)
	if err != nil {
		return nil, err
	}
	if !config.Active {
		return nil, failure(409, "error", "Activate this field before restoring a revision.")
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT v.value FROM work_item_property_versions v JOIN issues i ON i.id=v.issue_id AND i.workspace_id=v.workspace_id
 WHERE v.workspace_id=$1 AND i.project_id=$2 AND v.issue_id=$3 AND v.property_id=$4 AND v.id=$5 AND `+djangoapi.IssueVisibleSQL("i", "$6::uuid", "$1::uuid")+` FOR SHARE OF v`, state.WorkspaceID, state.ProjectID, state.ItemID, command.Restore.PropertyID, command.Restore.VersionID, q.ActorID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, requiredObject()
	}
	if err != nil {
		return nil, err
	}
	var historical any
	if err = json.Unmarshal(raw, &historical); err != nil {
		return nil, err
	}
	value, err := property.ValidateValue(property.RichText, config, historical, false, nil)
	if err != nil {
		return nil, invalid(map[string]any{"custom_properties": map[string]any{definition.ID: err.Error()}})
	}
	comparisonConfig := config
	comparisonConfig.Required = false
	current, currentErr := property.ValidateValue(property.RichText, comparisonConfig, state.Values[definition.ID], false, nil)
	if currentErr == nil && reflect.DeepEqual(value, current) {
		return nil, failure(409, "error", "This revision already matches the current field value.")
	}
	return value, nil
}
