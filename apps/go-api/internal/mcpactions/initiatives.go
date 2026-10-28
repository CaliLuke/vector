package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/CaliLuke/go-plane/internal/initiatives"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type initiativeApplication interface {
	Read(context.Context, initiatives.Query) (json.RawMessage, error)
	Mutate(context.Context, initiatives.Command) (projectmutations.Result, error)
}
type initiativeInput struct {
	Action          string  `json:"action"`
	ID              string  `json:"initiative_id"`
	Workspace       *string `json:"workspace_slug"`
	Name            *string `json:"name"`
	HTML            *string `json:"description_html"`
	Start           *string `json:"start_date"`
	End             *string `json:"end_date"`
	State           *string `json:"state"`
	Lead            *string `json:"lead"`
	Projects        *string `json:"project_ids"`
	Add             *string `json:"add_ids"`
	Remove          *string `json:"remove_ids"`
	Cursor          *string `json:"cursor"`
	PerPage         *int    `json:"per_page"`
	IncludeRevision bool    `json:"include_revision"`
	Request         *string `json:"request_id"`
	Revision        *string `json:"expected_revision"`
}

func (s *Service) Initiative(ctx context.Context, input *api.InitiativeRequest) (loom.Nullable[api.InitiativeResult], error) {
	raw, err := s.initiativeOperation(ctx, input)
	return decodeReleaseResult[api.InitiativeResult](raw, err)
}
func selectInitiative(input *api.InitiativeRequest) (initiativeInput, initiatives.Command, error) {
	var in initiativeInput
	cmd := initiatives.Command{Query: initiatives.Query{Params: map[string]string{}, Present: map[string]bool{}}, Fields: map[string]any{}}
	if input == nil || input.Validate() != nil {
		return in, cmd, api.MakeInvalidParams(errors.New("invalid initiative input"))
	}
	raw, err := json.Marshal(input)
	if err != nil || json.Unmarshal(raw, &in) != nil {
		return in, cmd, workitemResultError()
	}
	cmd.Action, cmd.IncludeRevision, cmd.ExpectedRevision = in.Action, in.IncludeRevision, in.Revision
	if in.ID != "" {
		cmd.ID, err = typeUUID(in.ID)
		if err != nil {
			return in, cmd, err
		}
	}
	if in.Request != nil {
		id, typeUUIDErr := typeUUID(*in.Request)
		if typeUUIDErr != nil {
			return in, cmd, typeUUIDErr
		}
		cmd.RequestID = &id
	}
	for key, value := range map[string]*string{"name": in.Name, "description_html": in.HTML, "start_date": in.Start, "end_date": in.End, "state": in.State, "lead_id": in.Lead} {
		if value != nil && *value != "" {
			cmd.Fields[key] = *value
		}
	}
	if in.Cursor != nil && *in.Cursor != "" {
		cmd.Params["cursor"], cmd.Present["cursor"] = *in.Cursor, true
	}
	if in.PerPage != nil && *in.PerPage != 0 {
		cmd.Params["per_page"], cmd.Present["per_page"] = strconv.Itoa(*in.PerPage), true
	}
	add := in.Add
	if in.Projects != nil {
		add = in.Projects
	}
	cmd.Add, err = milestoneIDs(add)
	if err != nil {
		return in, cmd, err
	}
	cmd.Remove, err = milestoneIDs(in.Remove)
	return in, cmd, err
}
func (s *Service) initiativeOperation(ctx context.Context, input *api.InitiativeRequest) (json.RawMessage, error) {
	in, cmd, err := selectInitiative(input)
	if err != nil {
		return nil, err
	}
	write := in.Action != "list" && in.Action != "retrieve" && in.Action != "list_projects" && in.Action != "list_workitems"
	ctx, actor, err := actionActor(ctx, in.Workspace, write)
	if err != nil {
		return nil, err
	}
	cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	var raw json.RawMessage
	if !write {
		raw, err = s.initiatives.Read(ctx, cmd.Query)
	} else {
		var value projectmutations.Result
		value, err = s.mutateInitiative(ctx, actor, in.Workspace, cmd)
		if err == nil {
			raw, err = resourceReceiptJSON(value, in.Request != nil || in.Revision != nil, in.Request != nil)
		}
	}
	if errors.Is(err, initiatives.ErrDisabled) {
		return nil, api.MakeFeatureDisabled(errors.New("initiatives disabled"))
	}
	if err != nil {
		return nil, archiveResourceError(err)
	}
	var body any
	if json.Unmarshal(raw, &body) != nil {
		return nil, workitemResultError()
	}
	body = projectNestedTimestamps(body, actor.Timezone)
	if in.Action == "list_projects" || in.Action == "list_workitems" {
		page, ok := body.(map[string]any)
		if !ok {
			return nil, workitemResultError()
		}
		rows, ok := page["results"].([]any)
		if !ok {
			return nil, workitemResultError()
		}
		for i, row := range rows {
			if in.Action == "list_workitems" {
				rows[i], err = workitemReadProjection(row, false)
			} else {
				record := &api.ProjectDetail{}
				err = decodeWorkitemProjection(row, record)
				rows[i] = record
			}
			if err != nil {
				return nil, err
			}
		}
		for key := range page {
			switch key {
			case "results", "count", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results", "revision":
			default:
				delete(page, key)
			}
		}
	}
	return json.Marshal(body)
}
func (s *Service) mutateInitiative(ctx context.Context, actor mcpauth.Principal, selector *string, cmd initiatives.Command) (projectmutations.Result, error) {
	cmd.Revalidate = mutationRevalidator(actor, selector, false)
	return s.observeMutation(ctx, actor, "initiative", cmd.Action, func(ctx context.Context) (projectmutations.Result, error) {
		return s.initiatives.Mutate(ctx, cmd)
	})
}
