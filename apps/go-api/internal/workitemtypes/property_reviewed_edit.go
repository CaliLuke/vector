package workitemtypes

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func propertyEditCommand(input PropertyInput) (propertyRemoval, error) {
	if input.Permanent || input.TypeID != "" || input.ID != "" && input.ID != input.Definition.ID {
		return propertyRemoval{}, reject(422, "Property updates preserve values and apply to the shared definition.")
	}
	if _, err := uuid.Parse(input.Definition.ID); err != nil {
		return propertyRemoval{}, reject(422, "Choose a property to update.")
	}
	definition := canonicalProperty(input.Definition)
	return propertyRemoval{Revision: input.Revision, Action: input.Action, ID: definition.ID, Definition: &definition}, nil
}

// Preview and commit use the same writer and validation. Only the existing-value
// preservation guard is bypassed: the review planner owns that migration.
func writeReviewedProperty(ctx context.Context, tx pgx.Tx, workspace, actor string, command propertyRemoval, catalog PropertyCatalog) error {
	input := PropertyInput{Revision: command.Revision, Action: command.Action, ID: command.ID, TypeID: command.TypeID, Permanent: command.Permanent}
	if command.Definition != nil {
		input.Definition = *command.Definition
		return saveProperty(ctx, tx, workspace, actor, input, catalog, true)
	}
	return removeProperty(ctx, tx, workspace, input, catalog, true)
}

func preservationReason(command propertyRemoval, definition property.Definition) string {
	if command.Action == "update" {
		return definition.Title + " configuration changed"
	}
	return definition.Title + " removed"
}
