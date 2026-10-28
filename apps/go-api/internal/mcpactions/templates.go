package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/templates"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type templateApplication interface {
	List(context.Context, templates.Query) (json.RawMessage, error)
	Mutate(context.Context, templates.Command) (projectmutations.Result, error)
}
type templateInput struct {
	Action          string  `json:"action"`
	Kind            string  `json:"kind"`
	Project         string  `json:"project_id"`
	ID              string  `json:"template_id"`
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	Data            *string `json:"template_data"`
	Workspace       *string `json:"workspace_slug"`
	Request         *string `json:"request_id"`
	Revision        *string `json:"expected_revision"`
	IncludeRevision bool    `json:"include_revision"`
}

func (s *Service) Template(ctx context.Context, input *api.TemplateRequest) (loom.Nullable[api.TemplateResult], error) {
	raw, err := s.templateOperation(ctx, input)
	return decodeReleaseResult[api.TemplateResult](raw, err)
}
func selectTemplate(input *api.TemplateRequest) (templateInput, templates.Command, error) {
	var in templateInput
	var cmd templates.Command
	if input == nil || input.Validate() != nil {
		return in, cmd, api.MakeInvalidParams(errors.New("invalid template input"))
	}
	raw, err := json.Marshal(input)
	if err != nil || json.Unmarshal(raw, &in) != nil {
		return in, cmd, workitemResultError()
	}
	cmd.Kind, cmd.Action, cmd.IncludeRevision, cmd.ExpectedRevision = in.Kind, in.Action, in.IncludeRevision, in.Revision
	if in.Project != "" {
		cmd.ProjectID, err = typeUUID(in.Project)
		if err != nil {
			return in, cmd, err
		}
	}
	if in.ID != "" {
		cmd.ID, err = typeUUID(in.ID)
		if err != nil {
			return in, cmd, err
		}
	}
	if in.Request != nil {
		id, err := typeUUID(*in.Request)
		if err != nil {
			return in, cmd, err
		}
		cmd.RequestID = &id
	}
	if in.Name != nil && *in.Name != "" {
		cmd.Name = in.Name
	}
	if in.Description != nil && *in.Description != "" {
		cmd.Description = in.Description
	}
	if in.Data != nil && *in.Data != "" {
		if djangoapi.DecodeJSON([]byte(*in.Data), &cmd.Data) != nil || cmd.Data == nil {
			return in, cmd, api.MakeInvalidParams(errors.New("template_data must be a JSON object"))
		}
	}
	return in, cmd, nil
}
func (s *Service) templateOperation(ctx context.Context, input *api.TemplateRequest) (json.RawMessage, error) {
	in, cmd, err := selectTemplate(input)
	if err != nil {
		return nil, err
	}
	ctx, actor, err := actionActor(ctx, in.Workspace, in.Action != "list")
	if err != nil {
		return nil, err
	}
	cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	var raw json.RawMessage
	if in.Action == "list" {
		raw, err = s.templates.List(ctx, cmd.Query)
	} else {
		var result projectmutations.Result
		result, err = s.mutateTemplate(ctx, actor, in.Workspace, cmd)
		if err == nil {
			raw, err = resourceReceiptJSON(result, in.Request != nil || in.Revision != nil, in.Request != nil)
		}
	}
	if errors.Is(err, templates.ErrWorkspaceOwned) {
		return nil, api.MakeWorkspaceOwned(errors.New("workspace owns templates"))
	}
	if err != nil {
		return nil, archiveResourceError(err)
	}
	var value any
	if djangoapi.DecodeJSON(raw, &value) != nil {
		return nil, workitemResultError()
	}
	return json.Marshal(djangoapi.Timestamps(value, actor.Timezone))
}

// resourceReceiptJSON is shared by native catalogue adapters with varying result shapes.
func resourceReceiptJSON(result projectmutations.Result, extended, keyed bool) (json.RawMessage, error) {
	if !extended {
		return result.Record, nil
	}
	var value any
	if djangoapi.DecodeJSON(result.Record, &value) != nil {
		return nil, workitemResultError()
	}
	body := map[string]any{"resource_id": result.ID, "revision": result.Revision, "result": value}
	if keyed {
		if result.RequestID == nil || result.ReplayExpiresAt == nil || result.Replayed == nil {
			return nil, workitemResultError()
		}
		body["request_id"], body["replay_expires_at"], body["replayed"] = result.RequestID.String(), result.ReplayExpiresAt.UTC().Format(time.RFC3339Nano), *result.Replayed
	}
	return json.Marshal(body)
}
func (s *Service) mutateTemplate(ctx context.Context, actor mcpauth.Principal, selector *string, cmd templates.Command) (projectmutations.Result, error) {
	cmd.Revalidate = mutationRevalidator(actor, selector, false)
	return s.observeMutation(ctx, actor, "template", cmd.Action, func(ctx context.Context) (projectmutations.Result, error) {
		return s.templates.Mutate(ctx, cmd)
	})
}
