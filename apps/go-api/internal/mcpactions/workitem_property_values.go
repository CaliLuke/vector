package mcpactions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type propertyValueWriter struct{ reader collaborationReader }

func (w propertyValueWriter) MutateResource(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	return w.reader.MutatePropertyValue(ctx, cmd)
}
func (s *Service) propertyValue(ctx context.Context, in propertyInput) (result loom.Nullable[api.WorkitemPropertyResult], err error) {
	read := in.Action == "get_value"
	ctx, actor, err := actionActor(ctx, in.Workspace, !read)
	if err != nil {
		return result, err
	}
	if in.Project == nil || *in.Project == "" {
		return result, propertyInputError()
	}
	scope, err := typeScope(actor, in.Project)
	if err != nil {
		return result, err
	}
	item, err := typeUUID(in.Workitem)
	if err != nil {
		return result, err
	}
	propertyID, err := typeUUID(in.OptionProperty)
	if err != nil {
		return result, err
	}
	var raw json.RawMessage
	if read {
		raw, err = s.properties.ReadPropertyValueAPI(ctx, workitemtypes.PropertyValueAPIQuery{APIScope: scope, IssueID: item, PropertyID: propertyID, IncludeRevision: in.IncludeRevision})
	} else {
		selected := resourceMutationInput{action: in.Action, project: *in.Project, target: in.Workitem, parent: in.OptionProperty, workspace: in.Workspace, request: in.Request, revision: in.Revision, fields: map[string]any{}}
		if in.Action == "set_value" {
			selected.fields, err = propertyValueFields(in)
			if err != nil {
				return result, err
			}
		}
		var value projectmutations.Result
		value, err = s.mutateResource(ctx, "workitem_property", selected, propertyValueWriter{s.collaborationReads})
		if err == nil {
			if in.Request != nil || in.Revision != nil {
				return propertyMutationReceipt(value, in.Request != nil)
			}
			raw = value.Record
		}
	}
	if err != nil {
		return result, workitemTypeError(err)
	}
	if string(raw) == "null" {
		return loom.NullValue[api.WorkitemPropertyResult](), nil
	}
	var data any
	if json.Unmarshal(raw, &data) != nil {
		return result, workitemResultError()
	}
	data = projectNestedTimestamps(data, actor.Timezone)
	if _, multiple := data.([]any); multiple {
		var values api.WorkitemPropertyArray
		if err = decodeWorkitemProjection(data, &values); err != nil {
			return result, err
		}
		return loom.NullableValue(api.NewWorkitemPropertyResultWorkitemPropertyArray(values)), nil
	}
	var value api.WorkitemPropertyValueRecord
	if err = decodeWorkitemProjection(data, &value); err != nil {
		return result, err
	}
	return loom.NullableValue(api.NewWorkitemPropertyResultWorkitemPropertyValueRecord(&value)), nil
}
func propertyValueFields(in propertyInput) (map[string]any, error) {
	out := map[string]any{}
	var value any
	if len(in.Value) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(in.Value))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil {
			return nil, propertyInputError()
		}
	}
	present := value != nil && value != ""
	html := in.DescriptionHTML != nil && *in.DescriptionHTML != ""
	if present == html {
		return nil, api.MakeInvalidParams(errors.New("provide value or description_html"))
	}
	if html {
		out["description_html"] = *in.DescriptionHTML
	} else {
		switch typed := value.(type) {
		case string, bool, json.Number:
		case []any:
			for _, item := range typed {
				if _, ok := item.(string); !ok {
					return nil, propertyInputError()
				}
			}
		default:
			return nil, propertyInputError()
		}
		out["value"] = value
	}
	if in.ExternalSource != nil {
		out["external_source"] = *in.ExternalSource
	}
	if in.ExternalID != nil {
		out["external_id"] = *in.ExternalID
	}
	return out, nil
}
