package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"

	loom "github.com/CaliLuke/loom/pkg"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
	"github.com/jackc/pgx/v5/pgconn"
)

type workItemEditRepository interface {
	ReadItemEditSource(context.Context, workitems.Request, string, string, string) (workitems.Result, error)
	PreviewItemChange(context.Context, workitems.Request, workitems.ItemEditInput) (workitems.Result, error)
	CommitItemChange(context.Context, workitems.Request, workitems.ItemEditInput) (workitems.Result, error)
}
type workItemTypesTimezone struct{}

func (s *workItemTypesService) PreviewItemChange(ctx context.Context, p *api.PreviewItemChangePayload) (loom.JSONValue, error) {
	return s.runItemEdit(ctx, p.Slug, p.Data, false)
}
func (s *workItemTypesService) CommitItemChange(ctx context.Context, p *api.CommitItemChangePayload) (loom.JSONValue, error) {
	return s.runItemEdit(ctx, p.Slug, p.Data, true)
}
func (s *workItemTypesService) runItemEdit(ctx context.Context, slug string, data *api.ItemEditInput, commit bool) (loom.JSONValue, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	if data == nil {
		return nil, &api.ItemEditError{Kind: "status_400", Message: "A proposed edit is required."}
	}
	if s.itemEdits == nil {
		return nil, &api.ItemEditError{Kind: "status_500", Message: "Work item edits are unavailable. Please try again."}
	}
	timezone, _ := ctx.Value(workItemTypesTimezone{}).(string)
	if timezone == "" {
		timezone = "UTC"
	}
	changes := map[string]any{}
	raw, err := json.Marshal(data.Changes)
	if err != nil {
		return nil, itemEditServiceError(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&changes); err != nil {
		return nil, itemEditServiceError(err)
	}
	input := workitems.ItemEditInput{Mode: data.Mode, ItemID: data.ItemID, ProjectID: data.ProjectID, Revision: data.Revision, Changes: changes}
	if data.Restore != nil {
		input.Restore = &workitems.ItemPropertyRestore{PropertyID: data.Restore.PropertyID, VersionID: data.Restore.VersionID}
	}
	if data.ExpectedSourceVersion != nil {
		input.ExpectedSourceVersion = *data.ExpectedSourceVersion
	}
	if data.ReviewID != nil {
		input.ReviewID = *data.ReviewID
	}
	request := workitems.Request{ActorID: actor, Slug: slug, Timezone: timezone}
	operation := s.itemEdits.PreviewItemChange
	if commit {
		operation = s.itemEdits.CommitItemChange
	}
	result, err := operation(ctx, request, input)
	if err != nil {
		return nil, itemEditServiceError(err)
	}
	encoded, err := loom.JSONValueFrom(result.Body)
	if err != nil {
		return nil, itemEditServiceError(err)
	}
	return encoded, nil
}

func itemEditServiceError(err error) error {
	var request *djangoapi.HTTPError
	if errors.As(err, &request) && request.Status >= 400 && request.Status < 500 {
		out := &api.ItemEditError{Kind: "status_" + strconv.Itoa(request.Status), Message: "The work item could not be saved. Check the highlighted fields."}
		if fields := request.Body; fields != nil {
			out.Fields = map[string]loom.JSONValue{}
			for name, value := range fields {
				encoded, encodeErr := loom.JSONValueFrom(value)
				if encodeErr != nil {
					return itemEditServiceError(encodeErr)
				}
				out.Fields[name] = encoded
			}
			if message, ok := fields["error"].(string); ok {
				out.Message = message
			}
		}
		return out
	}
	var configuration *store.Error
	if errors.As(err, &configuration) {
		return &api.ItemEditError{Kind: "status_" + strconv.Itoa(configuration.Status), Message: configuration.Message}
	}
	var constraint *pgconn.PgError
	if errors.As(err, &constraint) && (constraint.Code == "23514" || constraint.Code == "23503" || constraint.Code == "23505") {
		return &api.ItemEditError{Kind: "status_409", Message: "The work item or its relationships changed. Reload and review again."}
	}
	return &api.ItemEditError{Kind: "status_500", Message: "The work item could not be saved. Please try again."}
}

func (s *workItemTypesService) ReadItemEditSource(ctx context.Context, p *api.ReadItemEditSourcePayload) (loom.JSONValue, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	if s.itemEdits == nil {
		return nil, &api.ItemEditError{Kind: "status_500", Message: "Work item edits are unavailable. Please try again."}
	}
	timezone, _ := ctx.Value(workItemTypesTimezone{}).(string)
	if timezone == "" {
		timezone = "UTC"
	}
	project := ""
	if p.ProjectID != nil {
		project = *p.ProjectID
	}
	result, err := s.itemEdits.ReadItemEditSource(ctx, workitems.Request{ActorID: actor, Slug: p.Slug, Timezone: timezone}, p.Mode, p.ItemID, project)
	if err != nil {
		return nil, itemEditServiceError(err)
	}
	encoded, err := loom.JSONValueFrom(result.Body)
	if err != nil {
		return nil, itemEditServiceError(err)
	}
	return encoded, nil
}
