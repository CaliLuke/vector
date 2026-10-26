package workitems

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// ItemEditInput identifies the saved source independently of proposed fields.
// Private drafts may change destination; published items use the move workflow.
type ItemEditInput struct {
	Restore               *ItemPropertyRestore `json:"restore,omitempty"`
	ExpectedSourceVersion string               `json:"expected_source_version,omitempty"`
	Mode                  string               `json:"mode"`
	ProjectID             string               `json:"project_id"`
	ItemID                string               `json:"item_id"`
	Revision              int64                `json:"revision"`
	Changes               map[string]any       `json:"changes"`
	ReviewID              string               `json:"-"`
}

type ItemPropertyRestore struct {
	PropertyID string `json:"property_id"`
	VersionID  string `json:"version_id"`
}

type ItemEditReview struct {
	ID                  string `json:"id"`
	PreservedProperties int    `json:"preserved_properties"`
	Description         string `json:"description_html"`
}

type itemEditCommand struct {
	Kind string `json:"kind"`
	ItemEditInput
}

func normalizedItemEdit(input ItemEditInput) (itemEditCommand, []byte, error) {
	empty := itemEditCommand{}
	if err := normalizePropertyRestore(&input); err != nil {
		return empty, nil, err
	}
	if input.ExpectedSourceVersion == "" && input.ReviewID == "" {
		return empty, nil, failure(400, "error", "Load the saved work item before reviewing changes.")
	}
	if input.ExpectedSourceVersion != "" {
		value, decodeErr := hex.DecodeString(input.ExpectedSourceVersion)
		if decodeErr != nil || len(value) != 32 || strings.ToLower(input.ExpectedSourceVersion) != input.ExpectedSourceVersion {
			return empty, nil, failure(400, "error", "Choose a valid saved work item version.")
		}
	}
	if !slices.Contains([]string{"edit", "draft", "publish"}, input.Mode) || input.Revision < 0 || input.Changes == nil {
		return empty, nil, failure(400, "error", "Choose a saved work item or private draft and its proposed changes.")
	}
	id, err := uuid.Parse(input.ItemID)
	if err != nil {
		return empty, nil, failure(400, "error", "Choose a valid work item or draft.")
	}
	input.ItemID = id.String()
	if input.ProjectID != "" {
		project, parseErr := uuid.Parse(input.ProjectID)
		if parseErr != nil {
			return empty, nil, failure(400, "error", "Choose a valid project.")
		}
		input.ProjectID = project.String()
	} else if input.Mode != "draft" {
		return empty, nil, failure(400, "error", "A destination project is required.")
	}
	allowed := []string{"project_id", "name", "description_html", "state_id", "priority", "assignee_ids", "label_ids", "cycle_id", "module_ids", "start_date", "target_date", "estimate_point", "parent_id", "type_id", "custom_properties"}
	for field := range input.Changes {
		if !slices.Contains(allowed, field) {
			return empty, nil, failure(400, "error", "This edit includes an unsupported field: "+field+".")
		}
	}
	if input.Mode == "edit" {
		if project, present := input.Changes["project_id"]; present && !strings.EqualFold(stringValue(project), input.ProjectID) {
			return empty, nil, failure(400, "error", "Use the move action to change a published work item's project.")
		}
	}
	if input.Mode != "edit" {
		if project, present := input.Changes["project_id"]; present && !strings.EqualFold(stringValue(project), input.ProjectID) {
			return empty, nil, failure(400, "error", "The proposed project does not match this edit's destination.")
		}
	}
	// Normalize through JSON once, preserving numbers and field presence. The
	// receipt binds the exact submitted patch, including explicit nulls and clears.
	raw, err := json.Marshal(input.Changes)
	if err != nil {
		return empty, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var changes map[string]any
	if err = decoder.Decode(&changes); err != nil {
		return empty, nil, err
	}
	input.Changes = changes
	if input.Mode != "edit" {
		input.Changes["project_id"] = nil
		if input.ProjectID != "" {
			input.Changes["project_id"] = input.ProjectID
		}
	}
	if err = normalizeItemEditReferences(input.Changes); err != nil {
		return empty, nil, err
	}
	command := itemEditCommand{Kind: "work_item_edit", ItemEditInput: input}
	body, err := json.Marshal(command)
	return command, body, err
}

// Reject malformed IDs before any SQL cast or legacy serializer filtering.
func normalizeItemEditReferences(fields map[string]any) error {
	for _, key := range []string{"project_id", "state_id", "estimate_point", "parent_id", "type_id", "cycle_id"} {
		value, present := fields[key]
		if !present || value == nil {
			continue
		}
		text, ok := value.(string)
		id, err := uuid.Parse(text)
		if !ok || err != nil {
			return invalid(map[string]any{key: []string{"Choose a valid selection or clear the field."}})
		}
		fields[key] = id.String()
	}
	for _, key := range []string{"assignee_ids", "label_ids", "module_ids"} {
		value, present := fields[key]
		if !present {
			continue
		}
		values, ok := value.([]any)
		if !ok {
			return invalid(map[string]any{key: []string{"Expected a list of selections; use an empty list to clear it."}})
		}
		if len(values) > 1000 {
			return invalid(map[string]any{key: []string{"Choose no more than 1,000 selections."}})
		}
		seen := map[string]bool{}
		for index, value := range values {
			text, ok := value.(string)
			id, err := uuid.Parse(text)
			if !ok || err != nil {
				return invalid(map[string]any{key: []string{fmt.Sprintf("Selection %d is not a valid ID.", index+1)}})
			}
			if seen[id.String()] {
				return invalid(map[string]any{key: []string{"Remove duplicate selections."}})
			}
			seen[id.String()] = true
			values[index] = id.String()
		}
	}
	return nil
}
