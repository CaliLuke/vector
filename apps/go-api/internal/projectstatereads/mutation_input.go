package projectstatereads

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"github.com/google/uuid"
)

var (
	ErrInvalidMutation     = errors.New("invalid state mutation")
	ErrRevisionConflict    = errors.New("state revision has changed")
	ErrWorkspaceNotManaged = errors.New("workspace_not_managed")
)

// StateChanges preserves omission, false and zero. Explicit null clearing is
// available through native HTTP's existing parser, not the pinned MCP inputs.
type StateChanges struct {
	Name, Color, Description, Group *string
	ExternalSource, ExternalID      *string
	Sequence                        *float64
	Default                         *bool
}

type MutationCommand struct {
	ActorID, WorkspaceID, ProjectID, StateID uuid.UUID
	WorkspaceSlug, Action                    string
	Changes                                  StateChanges
	RequestID                                *uuid.UUID
	ExpectedRevision                         *string
	Revalidate                               func(context.Context) error
}

func (changes StateChanges) fields() map[string]any {
	fields := map[string]any{}
	for key, value := range map[string]*string{"name": changes.Name, "color": changes.Color, "description": changes.Description,
		"group": changes.Group, "external_source": changes.ExternalSource, "external_id": changes.ExternalID} {
		if value != nil {
			fields[key] = *value
		}
	}
	if changes.Sequence != nil {
		value := *changes.Sequence
		if value == 0 {
			value = 0
		} // Canonicalize negative zero for receipt hashes.
		fields["sequence"] = value
	}
	if changes.Default != nil {
		fields["default"] = *changes.Default
	}
	return fields
}

func normalizeStateMutation(cmd MutationCommand) (map[string]any, json.RawMessage, error) {
	if cmd.ActorID == uuid.Nil || cmd.WorkspaceID == uuid.Nil || cmd.WorkspaceSlug == "" ||
		!oneOf(cmd.Action, "create", "update", "delete") ||
		cmd.ExpectedRevision != nil && *cmd.ExpectedRevision == "" {
		return nil, nil, ErrInvalidMutation
	}
	create := cmd.Action == "create"
	if create && (cmd.StateID != uuid.Nil || cmd.ExpectedRevision != nil) || !create && cmd.StateID == uuid.Nil {
		return nil, nil, ErrInvalidMutation
	}
	if cmd.Changes.Sequence != nil && (math.IsNaN(*cmd.Changes.Sequence) || math.IsInf(*cmd.Changes.Sequence, 0)) {
		return nil, nil, ErrInvalidMutation
	}
	if cmd.ProjectID == uuid.Nil && (cmd.Changes.Sequence != nil || cmd.Changes.Default != nil || create && cmd.Changes.Group == nil) {
		return nil, nil, ErrInvalidMutation
	}
	fields := cmd.Changes.fields()
	if cmd.Action == "delete" && len(fields) != 0 {
		return nil, nil, ErrInvalidMutation
	}
	if err := validateStateFields(fields, create, false, nil); err != nil {
		return nil, nil, err
	}
	if create {
		for key, value := range map[string]any{"description": "", "group": "backlog", "default": false} {
			if !hasStateField(fields, key) {
				fields[key] = value
			}
		}
	}
	input := map[string]any{"fields": fields}
	if !create {
		input["state_id"] = cmd.StateID.String()
	}
	if cmd.ExpectedRevision != nil {
		input["expected_revision"] = *cmd.ExpectedRevision
	}
	raw, err := json.Marshal(input)
	return fields, raw, err
}
