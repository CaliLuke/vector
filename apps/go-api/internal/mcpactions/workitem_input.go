package mcpactions

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func selectWorkitemMutation(input *api.WorkitemRequest) (out resourceMutationInput, err error) {
	invalid := func() (resourceMutationInput, error) {
		return out, api.MakeInvalidParams(errors.New("invalid work item mutation"))
	}
	if input == nil {
		return invalid()
	}
	var payload any
	if v, ok := input.AsWorkitemCreate(); ok && v != nil && v.Action == "create" {
		out = resourceMutationInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, request: v.RequestID}
		payload = v
	} else if v, ok := input.AsWorkitemUpdate(); ok && v != nil && v.Action == "update" {
		out = resourceMutationInput{action: v.Action, project: v.ProjectID, target: v.WorkitemID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision}
		payload = v
	} else if v, ok := input.AsWorkitemDelete(); ok && v != nil && v.Action == "delete" {
		out = resourceMutationInput{action: v.Action, project: v.ProjectID, target: v.WorkitemID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision, fields: map[string]any{}}
	} else if v, ok := input.AsWorkitemArchive(); ok && v != nil && v.Action == "archive" {
		out = resourceMutationInput{action: v.Action, project: v.ProjectID, target: v.WorkitemID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision, fields: map[string]any{"archive": true}}
		if flag, ok := v.Archive.Value(); ok {
			out.fields["archive"] = flag
		}
	} else if v, ok := input.AsWorkitemManageAssignee(); ok && v != nil && v.Action == "manage_assignee" {
		out = resourceMutationInput{action: v.Action, project: v.ProjectID, target: v.WorkitemID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision}
		out.fields, err = planningFields(v.AddUserID, v.RemoveUserID)
	} else if v, ok := input.AsWorkitemManageLabel(); ok && v != nil && v.Action == "manage_label" {
		out = resourceMutationInput{action: v.Action, project: v.ProjectID, target: v.WorkitemID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision}
		out.fields, err = planningFields(v.AddLabelID, v.RemoveLabelID)
	} else {
		return invalid()
	}
	if err != nil {
		return out, err
	}
	if payload != nil {
		out.fields, err = workitemWriteInput(payload)
		if err != nil {
			return invalid()
		}
	}
	return out, nil
}
func workitemWriteInput(payload any) (map[string]any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var values map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&values); err != nil {
		return nil, err
	}
	fields := map[string]any{}
	for _, key := range []string{"name", "type_id", "point", "description_html", "description_stripped", "priority", "start_date", "target_date", "sort_order", "is_draft", "parent", "state", "estimate_point", "external_source", "external_id", "assignees", "labels"} {
		value, present := values[key]
		if !present {
			continue
		}
		if value == nil {
			if key == "start_date" || key == "target_date" {
				fields[key] = nil
			}
			continue
		}
		if value == "" {
			continue
		}
		if key == "point" || key == "sort_order" {
			number, ok := value.(json.Number)
			if !ok {
				return nil, errors.New("invalid numeric field")
			}
			n, err := number.Float64()
			if err != nil {
				return nil, err
			}
			if n == 0 {
				continue
			}
		}
		if key == "assignees" || key == "labels" {
			list, ok := value.([]any)
			if !ok {
				return nil, errors.New("invalid identifier list")
			}
			ids := make([]string, len(list))
			for i, item := range list {
				text, ok := item.(string)
				if !ok {
					return nil, errors.New("invalid identifier")
				}
				ids[i] = text
			}
			value = ids
		}
		fields[key] = value
	}
	if fields["description_html"] == nil {
		if plain, ok := fields["description_stripped"].(string); ok {
			escape := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#x27;", "\n", "<br/>")
			fields["description_html"] = "<p>" + escape.Replace(plain) + "</p>"
		}
	}
	delete(fields, "description_stripped")
	return fields, nil
}
