package mcpactions

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

// The generated closed union validates action-specific fields before decoding
// this shared application projection. No separate wire router is introduced.
type propertyInput struct {
	Workitem        string                                  `json:"workitem_id"`
	Value           json.RawMessage                         `json:"value"`
	DescriptionHTML *string                                 `json:"description_html"`
	Action          string                                  `json:"action"`
	Workspace       *string                                 `json:"workspace_slug"`
	Project         *string                                 `json:"project_id"`
	Type            string                                  `json:"workitem_type_id"`
	Property        string                                  `json:"workitem_property_id"`
	OptionProperty  string                                  `json:"property_id"`
	OptionID        string                                  `json:"option_id"`
	DisplayName     *string                                 `json:"display_name"`
	Kind            *string                                 `json:"property_type"`
	Relation        *string                                 `json:"relation_type"`
	Description     *string                                 `json:"description"`
	DisplayFormat   *string                                 `json:"display_format"`
	ExternalSource  *string                                 `json:"external_source"`
	ExternalID      *string                                 `json:"external_id"`
	Required        *bool                                   `json:"is_required"`
	Multiple        *bool                                   `json:"is_multi"`
	Active          *bool                                   `json:"is_active"`
	DefaultValue    *string                                 `json:"default_value"`
	Options         *string                                 `json:"options"`
	Configuration   *workitemtypes.PropertyAPIConfiguration `json:"configuration"`
	Name            *string                                 `json:"name"`
	Color           *string                                 `json:"color"`
	Default         *bool                                   `json:"is_default"`
	Attach          *string                                 `json:"attach_ids"`
	Detach          *string                                 `json:"detach_ids"`
	Cursor          *string                                 `json:"cursor"`
	PerPage         *int                                    `json:"per_page"`
	IncludeRevision bool                                    `json:"include_revision"`
	Request         *string                                 `json:"request_id"`
	Revision        *string                                 `json:"expected_revision"`
}

func selectProperty(input *api.WorkitemPropertyRequest) (propertyInput, error) {
	var out propertyInput
	if input == nil {
		return out, propertyInputError()
	}
	if input.Validate() != nil {
		return out, propertyInputError()
	}
	raw, err := json.Marshal(input)
	if err != nil || json.Unmarshal(raw, &out) != nil {
		return out, propertyInputError()
	}
	for _, value := range []**string{&out.DisplayName, &out.Kind, &out.Relation, &out.Description, &out.DisplayFormat, &out.ExternalSource, &out.ExternalID, &out.Name, &out.Color} {
		if *value != nil && **value == "" {
			*value = nil
		}
	}
	return out, nil
}
func propertyInputError() error { return api.MakeInvalidParams(errors.New("invalid property input")) }
func (in propertyInput) query(scope workitemtypes.APIScope) (workitemtypes.PropertyAPIQuery, error) {
	q := workitemtypes.PropertyAPIQuery{APIScope: scope, Action: in.Action, IncludeRevision: in.IncludeRevision}
	propertyID := in.Property
	if propertyID == "" {
		propertyID = in.OptionProperty
	}
	var err error
	if propertyID != "" {
		q.PropertyID, err = typeUUID(propertyID)
		if err != nil {
			return q, err
		}
	}
	if in.Type != "" {
		q.TypeID, err = typeUUID(in.Type)
		if err != nil {
			return q, err
		}
	}
	if in.OptionID != "" {
		q.OptionID, err = typeUUID(in.OptionID)
		if err != nil {
			return q, err
		}
	}
	if in.Project != nil && *in.Project != "" && (in.Action == "list" || in.Action == "list_options") {
		paging := djangoapi.Query{Params: map[string]string{}, Present: map[string]bool{}}
		if in.Cursor != nil && *in.Cursor != "" {
			paging.Params["cursor"], paging.Present["cursor"] = *in.Cursor, true
		}
		if in.PerPage != nil && *in.PerPage != 0 {
			paging.Params["per_page"], paging.Present["per_page"] = strconv.Itoa(*in.PerPage), true
		}
		q.Limit, _, q.Offset, err = djangoapi.PageWindow(paging)
		if err != nil {
			return q, propertyInputError()
		}
	}
	return q, nil
}
func (in propertyInput) command(q workitemtypes.PropertyAPIQuery) (workitemtypes.PropertyAPIMutation, error) {
	cmd := workitemtypes.PropertyAPIMutation{PropertyAPIQuery: q, ExpectedRevision: in.Revision, Changes: workitemtypes.PropertyAPIChanges{DisplayName: in.DisplayName, Kind: in.Kind, RelationType: in.Relation, Description: in.Description, DisplayFormat: in.DisplayFormat, ExternalSource: in.ExternalSource, ExternalID: in.ExternalID, Required: in.Required, Multiple: in.Multiple, Active: in.Active}, Option: workitemtypes.PropertyAPIOption{Description: in.Description, Color: in.Color, ExternalSource: in.ExternalSource, ExternalID: in.ExternalID, Default: in.Default}}
	cmd.Changes.Configuration = in.Configuration
	if in.Name != nil {
		cmd.Option.Name = *in.Name
	}
	var err error
	if in.Request != nil {
		id, typeUUIDErr := typeUUID(*in.Request)
		if typeUUIDErr != nil {
			return cmd, typeUUIDErr
		}
		cmd.RequestID = &id
	}
	cmd.Attach, err = milestoneIDs(in.Attach)
	if err != nil {
		return cmd, propertyInputError()
	}
	cmd.Detach, err = milestoneIDs(in.Detach)
	if err != nil {
		return cmd, propertyInputError()
	}
	if in.DefaultValue != nil && strings.TrimSpace(*in.DefaultValue) != "" {
		text := strings.TrimSpace(*in.DefaultValue)
		if !strings.HasPrefix(text, "[") || json.Unmarshal([]byte(text), &cmd.Changes.Defaults) != nil {
			cmd.Changes.Defaults = []string{text}
		}
	}
	if in.Options != nil && *in.Options != "" {
		if json.Unmarshal([]byte(*in.Options), &cmd.Changes.Options) != nil || cmd.Changes.Options == nil {
			return cmd, api.MakeInvalidParams(errors.New("options must be a JSON array of option objects"))
		}
		for _, option := range cmd.Changes.Options {
			if strings.TrimSpace(option.Name) == "" {
				return cmd, propertyInputError()
			}
		}
	}
	return cmd, nil
}
