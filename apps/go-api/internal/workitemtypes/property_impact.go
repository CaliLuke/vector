package workitemtypes

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

func rejectDestructivePropertyEdit(ctx context.Context, tx pgx.Tx, workspace string, before, after property.Definition) error {
	if reflect.DeepEqual(before.Configuration, after.Configuration) && reflect.DeepEqual(before.Contexts, after.Contexts) {
		return nil
	}
	if before.Kind != property.Cascading {
		return nil
	}
	rows, err := tx.Query(ctx, `SELECT COALESCE(project_id::text,''),COALESCE(type_id::text,''),custom_properties->$2 FROM issues WHERE workspace_id=$1 AND custom_properties ? $2 UNION ALL SELECT COALESCE(project_id::text,''),COALESCE(type_id::text,''),custom_properties->$2 FROM draft_issues WHERE workspace_id=$1 AND custom_properties ? $2`, workspace, before.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var project, typeID string
		var raw []byte
		if err = rows.Scan(&project, &typeID, &raw); err != nil {
			return err
		}
		var value any
		if err = json.Unmarshal(raw, &value); err != nil {
			return err
		}
		if value == nil {
			continue
		}
		scope := property.Scope{ProjectID: project, TypeID: typeID}
		removed, changeErr := property.CascadingSelectionRequiresPreservation(before, after, scope, scope, value)
		if changeErr != nil {
			return reject(422, changeErr.Error())
		}
		if removed {
			return reject(409, "This change removes selected cascading values. Review preservation before saving.")
		}
	}
	return rows.Err()
}
