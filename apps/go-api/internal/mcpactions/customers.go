package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"go.opentelemetry.io/otel"
)

type customerApplication interface {
	Read(context.Context, customers.Query) (json.RawMessage, error)
	Mutate(context.Context, customers.Command) (projectmutations.Result, error)
}
type customerInput struct {
	Action          string  `json:"action"`
	Customer        string  `json:"customer_id"`
	Property        string  `json:"property_id"`
	Request         *string `json:"request_id"`
	CustomerRequest string  `json:"customer_request_id"`
	Mutation        *string `json:"mutation_id"`
	Workspace       *string `json:"workspace_slug"`
	Revision        *string `json:"expected_revision"`
	IncludeRevision bool    `json:"include_revision"`
	Name            *string `json:"name"`
	HTML            *string `json:"description_html"`
	Email           *string `json:"email"`
	URL             *string `json:"website_url"`
	Domain          *string `json:"domain"`
	Employees       *int64  `json:"employees"`
	Stage           *string `json:"stage"`
	Contract        *string `json:"contract_status"`
	Revenue         *string `json:"revenue"`
	Source          *string `json:"external_source"`
	External        *string `json:"external_id"`
	Link            *string `json:"link"`
	DisplayName     *string `json:"display_name"`
	Description     *string `json:"description"`
	PropertyType    *string `json:"property_type"`
	RelationType    *string `json:"relation_type"`
	Format          *string `json:"display_format"`
	Default         *string `json:"default_value"`
	Options         *string `json:"options"`
	Values          *string `json:"values"`
	Required        *bool   `json:"is_required"`
	Multi           *bool   `json:"is_multi"`
	Active          *bool   `json:"is_active"`
	Add             *string `json:"link_ids"`
	Remove          *string `json:"unlink_ids"`
	Items           *string `json:"workitem_ids"`
	Search          string  `json:"search"`
	Query           string  `json:"query"`
	Cursor          *string `json:"cursor"`
	PerPage         *int    `json:"per_page"`
}

func (s *Service) Customer(ctx context.Context, input *api.CustomerRequest) (loom.Nullable[api.CustomerResult], error) {
	if input == nil {
		return loom.Nullable[api.CustomerResult]{}, customerInputError()
	}
	raw, err := s.customerOperation(ctx, "customer", input)
	return decodeReleaseResult[api.CustomerResult](raw, err)
}
func (s *Service) CustomerProperty(ctx context.Context, input *api.CustomerPropertyRequest) (loom.Nullable[api.CustomerPropertyResult], error) {
	if input == nil {
		return loom.Nullable[api.CustomerPropertyResult]{}, customerInputError()
	}
	raw, err := s.customerOperation(ctx, "customer_property", input)
	return decodeReleaseResult[api.CustomerPropertyResult](raw, err)
}
func (s *Service) CustomerRequest(ctx context.Context, input *api.CustomerRequestRequest) (loom.Nullable[api.CustomerRequestResult], error) {
	if input == nil {
		return loom.Nullable[api.CustomerRequestResult]{}, customerInputError()
	}
	raw, err := s.customerOperation(ctx, "customer_request", input)
	return decodeReleaseResult[api.CustomerRequestResult](raw, err)
}
func customerInputError() error { return api.MakeInvalidParams(errors.New("invalid customer input")) }
func selectCustomer(tool string, input interface{ Validate() error }) (customerInput, customers.Command, error) {
	var in customerInput
	cmd := customers.Command{Query: customers.Query{Tool: tool, Params: map[string]string{}, Present: map[string]bool{}}, Fields: map[string]any{}}
	if input.Validate() != nil {
		return in, cmd, customerInputError()
	}
	raw, err := json.Marshal(input)
	if err != nil || json.Unmarshal(raw, &in) != nil {
		return in, cmd, customerInputError()
	}
	cmd.Action, cmd.ExpectedRevision, cmd.IncludeRevision = in.Action, in.Revision, in.IncludeRevision
	if in.Customer != "" {
		id, typeUUIDErr := typeUUID(in.Customer)
		if typeUUIDErr != nil {
			return in, cmd, typeUUIDErr
		}
		if tool == "customer" && in.Action != "list_workitems" && in.Action != "manage_workitems" {
			cmd.ID = id
		} else {
			cmd.CustomerID = id
		}
	}
	if in.Property != "" {
		cmd.ID, err = typeUUID(in.Property)
		if err != nil {
			return in, cmd, err
		}
	}
	retry := in.Request
	if tool == "customer_request" {
		retry = in.Mutation
		if in.Request != nil {
			cmd.ID, err = typeUUID(*in.Request)
			if err != nil {
				return in, cmd, err
			}
			cmd.RequestID = cmd.ID
		}
	}
	if in.CustomerRequest != "" {
		cmd.RequestID, err = typeUUID(in.CustomerRequest)
		if err != nil {
			return in, cmd, err
		}
	}
	if retry != nil {
		id, typeUUIDErr2 := typeUUID(*retry)
		if typeUUIDErr2 != nil {
			return in, cmd, typeUUIDErr2
		}
		cmd.RetryID = &id
	}
	for key, value := range map[string]*string{"name": in.Name, "description_html": in.HTML, "email": in.Email, "website_url": in.URL, "domain": in.Domain, "stage": in.Stage, "contract_status": in.Contract, "revenue": in.Revenue, "external_source": in.Source, "external_id": in.External, "link": in.Link, "display_name": in.DisplayName, "description": in.Description, "property_type": in.PropertyType, "relation_type": in.RelationType, "display_format": in.Format} {
		if value != nil && *value != "" {
			cmd.Fields[key] = *value
		}
	}
	for key, value := range map[string]*bool{"is_required": in.Required, "is_multi": in.Multi, "is_active": in.Active} {
		if value != nil {
			cmd.Fields[key] = *value
		}
	}
	if in.Employees != nil && *in.Employees != 0 {
		cmd.Fields["employees"] = *in.Employees
	}
	if in.Default != nil && strings.TrimSpace(*in.Default) != "" {
		value := strings.TrimSpace(*in.Default)
		defaults := []string{value}
		if strings.HasPrefix(value, "[") {
			if json.Unmarshal([]byte(value), &defaults) != nil {
				return in, cmd, customerInputError()
			}
		}
		cmd.Fields["default_value"] = defaults
	}
	if in.Options != nil && *in.Options != "" {
		if json.Unmarshal([]byte(*in.Options), &cmd.Options) != nil || cmd.Options == nil {
			return in, cmd, customerInputError()
		}
	}
	if in.Values != nil {
		if json.Unmarshal([]byte(*in.Values), &cmd.Values) != nil || cmd.Values == nil {
			return in, cmd, customerInputError()
		}
	}
	add := in.Add
	if in.Items != nil {
		add = in.Items
	}
	cmd.Add, err = milestoneIDs(add)
	if err != nil {
		return in, cmd, err
	}
	cmd.Remove, err = milestoneIDs(in.Remove)
	if err != nil {
		return in, cmd, err
	}
	if in.Cursor != nil && *in.Cursor != "" {
		cmd.Params["cursor"], cmd.Present["cursor"] = *in.Cursor, true
	}
	if in.PerPage != nil && *in.PerPage != 0 {
		cmd.Params["per_page"], cmd.Present["per_page"] = strconv.Itoa(*in.PerPage), true
	}
	cmd.Search = in.Query
	if in.Action == "list_workitems" {
		cmd.Search = in.Search
	}
	return in, cmd, nil
}
func (s *Service) customerOperation(ctx context.Context, tool string, input interface{ Validate() error }) (json.RawMessage, error) {
	in, cmd, err := selectCustomer(tool, input)
	if err != nil {
		return nil, err
	}
	write := in.Action != "list" && in.Action != "retrieve" && in.Action != "list_workitems" && in.Action != "get_values"
	ctx, actor, err := actionActor(ctx, in.Workspace, write)
	if err != nil {
		return nil, err
	}
	cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	var raw json.RawMessage
	if !write {
		raw, err = s.customers.Read(ctx, cmd.Query)
	} else {
		var result projectmutations.Result
		result, err = s.mutateCustomer(ctx, actor, in.Workspace, cmd)
		if err == nil {
			raw, err = resourceReceiptJSON(result, cmd.RetryID != nil || in.Revision != nil, cmd.RetryID != nil)
			if err == nil && tool == "customer_request" && cmd.RetryID != nil {
				var receipt map[string]any
				if json.Unmarshal(raw, &receipt) != nil {
					return nil, workitemResultError()
				}
				receipt["mutation_id"] = receipt["request_id"]
				delete(receipt, "request_id")
				raw, err = json.Marshal(receipt)
			}
		}
	}
	if errors.Is(err, customers.ErrDisabled) {
		return nil, api.MakeFeatureDisabled(errors.New("customers disabled"))
	}
	if err != nil {
		return nil, archiveResourceError(err)
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil, workitemResultError()
	}
	return json.Marshal(projectNestedTimestamps(value, actor.Timezone))
}
func (s *Service) mutateCustomer(ctx context.Context, actor mcpauth.Principal, selector *string, cmd customers.Command) (value projectmutations.Result, err error) {
	start := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp."+cmd.Tool+"."+cmd.Action)
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, cmd.Tool, cmd.Action, err)
		fields["latency_ms"] = time.Since(start).Milliseconds()
		if err == nil {
			fields["committed_resource_id"] = value.ID
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	cmd.Revalidate = func(ctx context.Context) error {
		_, current, err := actionActor(ctx, selector, true)
		if err != nil {
			return err
		}
		if current.UserID != actor.UserID || current.WorkspaceID != actor.WorkspaceID {
			return api.MakeForbidden(errors.New("authorized scope changed"))
		}
		return nil
	}
	return s.customers.Mutate(ctx, cmd)
}
