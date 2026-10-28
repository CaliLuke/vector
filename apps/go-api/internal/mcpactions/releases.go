package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/releases"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type releaseApplication interface {
	Read(context.Context, releases.Query) (json.RawMessage, error)
	Mutate(context.Context, releases.Command) (projectmutations.Result, error)
}
type releaseInput struct {
	Action          string  `json:"action"`
	Release         string  `json:"release_id"`
	Label           string  `json:"label_id"`
	Tag             string  `json:"tag_id"`
	Workspace       *string `json:"workspace_slug"`
	Request         *string `json:"request_id"`
	Revision        *string `json:"expected_revision"`
	IncludeRevision bool    `json:"include_revision"`
	Name            *string `json:"name"`
	HTML            *string `json:"description_html"`
	Plain           *string `json:"description_stripped"`
	Description     *string `json:"description"`
	Status          *string `json:"status"`
	ReleaseDate     *string `json:"release_date"`
	TargetDate      *string `json:"target_date"`
	Lead            *string `json:"lead_id"`
	Prerelease      *bool   `json:"is_prerelease"`
	ExternalSource  *string `json:"external_source"`
	ExternalID      *string `json:"external_id"`
	Version         *string `json:"version"`
	Commit          *string `json:"commit_hash"`
	GitTag          *string `json:"git_tag"`
	Color           *string `json:"color"`
	Order           *int64  `json:"sort_order"`
	Add             *string `json:"add_ids"`
	Remove          *string `json:"remove_ids"`
	Labels          *string `json:"label_ids"`
	Cursor          *string `json:"cursor"`
	PerPage         *int    `json:"per_page"`
}

func (s *Service) Release(ctx context.Context, input *api.ReleaseRequest) (loom.Nullable[api.ReleaseResult], error) {
	if input == nil {
		return loom.Nullable[api.ReleaseResult]{}, releaseInputError()
	}
	raw, err := s.releaseOperation(ctx, "release", input)
	return decodeReleaseResult[api.ReleaseResult](raw, err)
}
func (s *Service) ReleaseLabel(ctx context.Context, input *api.ReleaseLabelRequest) (loom.Nullable[api.ReleaseLabelResult], error) {
	if input == nil {
		return loom.Nullable[api.ReleaseLabelResult]{}, releaseInputError()
	}
	raw, err := s.releaseOperation(ctx, "release_label", input)
	return decodeReleaseResult[api.ReleaseLabelResult](raw, err)
}
func (s *Service) ReleaseTag(ctx context.Context, input *api.ReleaseTagRequest) (loom.Nullable[api.ReleaseTagResult], error) {
	if input == nil {
		return loom.Nullable[api.ReleaseTagResult]{}, releaseInputError()
	}
	raw, err := s.releaseOperation(ctx, "release_tag", input)
	return decodeReleaseResult[api.ReleaseTagResult](raw, err)
}
func decodeReleaseResult[T any](raw json.RawMessage, err error) (loom.Nullable[T], error) {
	if err != nil {
		return loom.Nullable[T]{}, err
	}
	if string(raw) == "null" {
		return loom.NullValue[T](), nil
	}
	var value T
	if json.Unmarshal(raw, &value) != nil {
		return loom.Nullable[T]{}, workitemResultError()
	}
	return loom.NullableValue(value), nil
}
func releaseInputError() error { return api.MakeInvalidParams(errors.New("invalid release input")) }
func selectRelease(tool string, input interface{ Validate() error }) (releaseInput, releases.Command, error) {
	var in releaseInput
	cmd := releases.Command{Query: releases.Query{Tool: tool, Params: map[string]string{}, Present: map[string]bool{}}, Fields: map[string]any{}}
	if input.Validate() != nil {
		return in, cmd, releaseInputError()
	}
	raw, err := json.Marshal(input)
	if err != nil || json.Unmarshal(raw, &in) != nil {
		return in, cmd, releaseInputError()
	}
	cmd.Action, cmd.IncludeRevision, cmd.ExpectedRevision = in.Action, in.IncludeRevision, in.Revision
	id := in.Release
	if tool == "release_label" {
		id = in.Label
		if in.Release != "" {
			cmd.ReleaseID, err = typeUUID(in.Release)
			if err != nil {
				return in, cmd, err
			}
		}
	}
	if tool == "release_tag" {
		id = in.Tag
	}
	if id != "" {
		cmd.ID, err = typeUUID(id)
		if err != nil {
			return in, cmd, err
		}
	}
	if in.Request != nil {
		request, typeUUIDErr := typeUUID(*in.Request)
		if typeUUIDErr != nil {
			return in, cmd, typeUUIDErr
		}
		cmd.RequestID = &request
	}
	if in.Cursor != nil && *in.Cursor != "" {
		cmd.Params["cursor"], cmd.Present["cursor"] = *in.Cursor, true
	}
	if in.PerPage != nil && *in.PerPage != 0 {
		cmd.Params["per_page"], cmd.Present["per_page"] = strconv.Itoa(*in.PerPage), true
	}
	for key, value := range map[string]*string{"name": in.Name, "description_html": in.HTML, "description_stripped": in.Plain, "description": in.Description, "status": in.Status, "release_date": in.ReleaseDate, "target_date": in.TargetDate, "lead_id": in.Lead, "external_source": in.ExternalSource, "external_id": in.ExternalID, "version": in.Version, "commit_hash": in.Commit, "git_tag": in.GitTag, "color": in.Color} {
		if value != nil && *value != "" {
			cmd.Fields[key] = *value
		}
	}
	if tool == "release" && in.Tag != "" {
		cmd.Fields["tag_id"] = in.Tag
	}
	if in.Prerelease != nil {
		cmd.Fields["is_prerelease"] = *in.Prerelease
	}
	if in.Order != nil {
		cmd.Fields["sort_order"] = *in.Order
	}
	add := in.Add
	if in.Labels != nil {
		add = in.Labels
	}
	cmd.Add, err = milestoneIDs(add)
	if err != nil {
		return in, cmd, releaseInputError()
	}
	cmd.Remove, err = milestoneIDs(in.Remove)
	if err != nil {
		return in, cmd, releaseInputError()
	}
	return in, cmd, nil
}
func (s *Service) releaseOperation(ctx context.Context, tool string, input interface{ Validate() error }) (json.RawMessage, error) {
	in, cmd, err := selectRelease(tool, input)
	if err != nil {
		return nil, err
	}
	write := cmd.Action != "list" && cmd.Action != "retrieve" && cmd.Action != "get_changelog" && cmd.Action != "list_workitems"
	ctx, actor, err := actionActor(ctx, in.Workspace, write)
	if err != nil {
		return nil, err
	}
	cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	var raw json.RawMessage
	if !write {
		raw, err = s.releases.Read(ctx, cmd.Query)
	} else {
		var value projectmutations.Result
		value, err = s.mutateRelease(ctx, actor, in.Workspace, cmd)
		if err == nil {
			raw = value.Record
			if in.Request != nil || in.Revision != nil {
				var body any
				if len(raw) > 0 && json.Unmarshal(raw, &body) != nil {
					return nil, workitemResultError()
				}
				receipt := map[string]any{"resource_id": value.ID, "revision": value.Revision, "result": body}
				if in.Request != nil {
					if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
						return nil, workitemResultError()
					}
					receipt["request_id"], receipt["replay_expires_at"], receipt["replayed"] = value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano), *value.Replayed
				}
				raw, err = json.Marshal(receipt)
			}
		}
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
func (s *Service) mutateRelease(ctx context.Context, actor mcpauth.Principal, selector *string, cmd releases.Command) (projectmutations.Result, error) {
	cmd.Revalidate = mutationRevalidator(actor, selector, true)
	return s.observeMutation(ctx, actor, cmd.Tool, cmd.Action, func(ctx context.Context) (projectmutations.Result, error) {
		return s.releases.Mutate(ctx, cmd)
	})
}
