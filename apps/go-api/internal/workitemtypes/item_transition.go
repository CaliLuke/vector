package workitemtypes

import (
	"context"
	"maps"
	"strings"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

type PreservedItemProperty struct {
	Definition property.Definition
	Value      any
	HTML       string
}

type ItemTransition struct {
	ItemValueResult
	Description string
	Preserved   []PreservedItemProperty
}

// PlanItemTransition is the value-preservation boundary for a reviewed edit.
// The caller must authorize, acquire the workspace schema lock, then lock and
// reload the canonical source item. Bind this plan and the complete proposed
// edit to its review receipt, then commit all item
// fields and effects atomically. This function does not save an item or receipt.
// Description is the proposed document, so preservation cannot discard edits.
func PlanItemTransition(ctx context.Context, tx pgx.Tx, in ItemValueInput, description string) (ItemTransition, error) {
	sourceVisible, err := transitionSourceVisible(ctx, tx, in)
	if err != nil {
		return ItemTransition{}, err
	}
	return planItemTransition(ctx, tx, in, description, sourceVisible)
}

// Schema migrations may preserve hidden values internally after administrator
// authorization. Only aggregate impact counts leave that boundary.
func planItemTransition(ctx context.Context, tx pgx.Tx, in ItemValueInput, description string, sourceVisible bool) (ItemTransition, error) {
	out := ItemTransition{Description: description, Preserved: []PreservedItemProperty{}}
	if !in.Existing {
		return out, reject(422, "Select an existing work item or draft to review.")
	}
	in.ProjectID, in.PreviousProjectID = strings.ToLower(in.ProjectID), strings.ToLower(in.PreviousProjectID)
	in.TypeID, in.PreviousTypeID = strings.ToLower(in.TypeID), strings.ToLower(in.PreviousTypeID)
	if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); err != nil {
		return out, err
	}
	destination, err := resolveItemType(ctx, tx, in)
	if err != nil {
		return out, err
	}
	previous := []property.Definition{}
	if in.PreviousTypeID != "" {
		previous, err = itemPropertyDefinitions(ctx, tx, in.WorkspaceID, in.PreviousTypeID)
		if err != nil {
			return out, err
		}
	}
	next := []property.Definition{}
	if destination != "" {
		next, err = itemPropertyDefinitions(ctx, tx, in.WorkspaceID, destination)
		if err != nil {
			return out, err
		}
	}
	in.Stored, in.Submitted, err = submittedFormerValues(ctx, tx, in, previous, next, sourceVisible)
	if err != nil {
		return out, err
	}
	remaining, err := transitionValues(in, destination, previous, next)
	if err != nil {
		return out, err
	}
	for _, def := range previous {
		value, present := in.Stored[def.ID]
		if !present || !nonemptyPropertyValue(value) {
			continue
		}
		if _, retained := remaining[def.ID]; retained {
			continue
		}
		config, _, configErr := property.Effective(def, in.PreviousProjectID, in.PreviousTypeID)
		if configErr != nil {
			return out, configErr
		}
		labels := map[string]string{}
		var labelErr error
		if sourceVisible {
			labels, labelErr = preservationLabels(ctx, tx, in.WorkspaceID, in.PreviousProjectID, def.Kind, value)
		}
		if labelErr != nil {
			return out, labelErr
		}
		rendered, renderErr := property.PreservationHTML(def, config, value, labels)
		if renderErr != nil {
			return out, renderErr
		}
		out.Preserved = append(out.Preserved, PreservedItemProperty{Definition: def, Value: value, HTML: rendered})
	}
	if len(out.Preserved) > 0 {
		var block strings.Builder
		block.WriteString("<h3>Values preserved from the previous work item configuration</h3>")
		for _, entry := range out.Preserved {
			block.WriteString(entry.HTML)
		}
		out.Description += block.String()
	}
	if len(out.Description) > content.MaxContentBytes {
		return out, reject(422, "Preserving these values would exceed the work item's description limit.")
	}
	in.Stored = remaining
	out.ItemValueResult, err = ValidateItemValues(ctx, tx, in)
	return out, err
}

func transitionValues(in ItemValueInput, destination string, previous, next []property.Definition) (map[string]any, error) {
	remaining := maps.Clone(in.Stored)
	oldDefinitions := map[string]property.Definition{}
	newDefinitions := map[string]property.Definition{}
	for _, def := range previous {
		oldDefinitions[def.ID] = def
	}
	for _, def := range next {
		newDefinitions[def.ID] = def
	}
	for id, value := range in.Stored {
		after, attached := newDefinitions[id]
		before, known := oldDefinitions[id]
		if !known {
			if nonemptyPropertyValue(value) {
				return nil, reject(409, "A stored property's definition is unavailable. Repair it before changing the work item configuration.")
			}
			delete(remaining, id)
			continue
		}
		if !attached {
			delete(remaining, id)
			continue
		}
		if in.PreviousProjectID == in.ProjectID && in.PreviousTypeID == destination {
			continue
		}
		removed, err := property.CascadingSelectionRequiresPreservation(before, after, property.Scope{ProjectID: in.PreviousProjectID, TypeID: in.PreviousTypeID}, property.Scope{ProjectID: in.ProjectID, TypeID: destination}, value)
		if err != nil {
			return nil, reject(422, err.Error())
		}
		if removed {
			delete(remaining, id)
		}
	}
	return remaining, nil
}
