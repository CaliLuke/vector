package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

// An absent property object is different from a malformed object or an explicit
// null. Individual null values are valid clears; the enclosing object is required.
func submittedCustomValues(input djangoinput.Input) (map[string]any, error) {
	value, present := input.Fields["custom_properties"]
	if !present {
		return nil, nil
	}
	if input.Form {
		text, ok := value.(string)
		if !ok || !json.Valid([]byte(text)) {
			return nil, invalid(map[string]any{"custom_properties": []string{"Enter a JSON object."}})
		}
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
	}
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return nil, invalid(map[string]any{"custom_properties": []string{"Expected an object of property values."}})
	}
	return object, nil
}

// The caller owns authorization and a transaction, and has locked old before
// merging submitted standard fields into row. Only submitted custom keys change.
func (r *Repository) validateCustomValues(ctx context.Context, actor string, row, old map[string]any, submitted map[string]any, draft, creating, typeSpecified bool) ([]string, error) {
	tx, ok := r.db.(pgx.Tx)
	if !ok {
		return nil, errors.New("custom property writes require a transaction")
	}
	stored, _ := old["custom_properties"].(map[string]any)
	creator := stringValue(row["created_by_id"])
	if creating {
		creator = actor
	}
	valueInput := workitemtypes.ItemValueInput{
		WorkspaceID: stringValue(row["workspace_id"]), ProjectID: stringValue(row["project_id"]),
		PreviousProjectID: stringValue(old["project_id"]), TypeID: stringValue(row["type_id"]),
		PreviousTypeID: stringValue(old["type_id"]), CreatorID: creator, ActorID: actor, ParentID: stringValue(row["parent_id"]), Existing: old != nil, TypeSpecified: typeSpecified,
		Stored: stored, Submitted: submitted, Draft: draft, Creating: creating,
		ProjectChanged: old != nil && old["project_id"] != row["project_id"],
	}
	var result workitemtypes.ItemValueResult
	var err error
	if r.itemEdit != nil {
		plan, planErr := workitemtypes.PlanItemTransition(ctx, tx, valueInput, stringValue(row["description_html"]))
		result, err = plan.ItemValueResult, planErr
		if err == nil {
			row["description_html"] = plan.Description
			r.itemEdit.plan = plan
		}
	} else {
		result, err = workitemtypes.ValidateItemValues(ctx, tx, valueInput)
	}

	var fieldError *workitemtypes.ItemValueError
	var configurationError *workitemtypes.Error
	if errors.As(err, &fieldError) {
		return nil, invalid(map[string]any{"custom_properties": fieldError.Fields})
	}
	if errors.As(err, &configurationError) {
		status := configurationError.Status
		// Ordinary work-item transports use 400 for field validation. The
		// settings API's 422 is not part of their existing HTTP contract.
		if status == 422 {
			status = 400
		}
		return nil, failure(status, "error", configurationError.Message)
	}
	if err != nil {
		return nil, err
	}
	row["type_id"] = nil
	if result.TypeID != "" {
		row["type_id"] = result.TypeID
	}
	// A dedicated value-delete removes presence only after native validation has
	// accepted the clear (including required/read-only checks), before history,
	// model persistence, search indexing and webhook snapshots are captured.
	for _, id := range r.clearedPropertyIDs {
		if value, present := submitted[id]; !present || value != nil {
			return nil, errors.New("invalid property clear")
		}
		delete(result.Values, id)
	}
	row["custom_properties"] = result.Values
	if r.propertyValueMutation && r.itemEdit == nil {
		assetWriter := *r
		assetWriter.itemEdit = &itemEditState{command: itemEditCommand{ItemEditInput: ItemEditInput{Mode: "edit", ItemID: stringValue(row["id"]), ProjectID: stringValue(row["project_id"])}}}
		if err = assetWriter.associateItemEditAssets(ctx, actor, row, old); err != nil {
			return nil, err
		}
	}
	if r.itemEdit != nil {
		if err = r.associateItemEditAssets(ctx, actor, row, old); err != nil {
			return nil, err
		}
	}
	return result.Subscribers, nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func (r *Repository) subscribePropertyMembers(ctx context.Context, actor string, row map[string]any, members []string) error {
	for _, member := range members {
		// Preserve manual subscriptions; the unique active subscription index
		// also handles a concurrent explicit subscription.
		_, err := r.db.Exec(ctx, `INSERT INTO issue_subscribers(id,issue_id,subscriber_id,project_id,workspace_id,created_at,updated_at,created_by_id)
 SELECT gen_random_uuid(),$1,$2,$3,$4,now(),now(),$5 WHERE NOT EXISTS(SELECT 1 FROM issue_subscribers WHERE issue_id=$1 AND subscriber_id=$2 AND deleted_at IS NULL) ON CONFLICT DO NOTHING`, row["id"], member, row["project_id"], row["workspace_id"], actor)
		if err != nil {
			return err
		}
	}
	return nil
}

func hasSubmittedType(fields map[string]any) bool {
	_, alias := fields["type_id"]
	_, canonical := fields["type"]
	return alias || canonical
}
