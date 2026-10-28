package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

type pageApplication interface {
	ReadPageAPI(context.Context, pagereads.PageAPIQuery) (json.RawMessage, error)
	MutatePageAPI(context.Context, pagereads.PageAPICommand) (projectmutations.Result, error)
}
type pageInput struct {
	Action          string          `json:"action"`
	Workspace       *string         `json:"workspace_slug"`
	Project         string          `json:"project_id"`
	Page            string          `json:"page_id"`
	Collection      string          `json:"collection_id"`
	Parent          string          `json:"parent_id"`
	Item            string          `json:"workitem_id"`
	ItemLink        string          `json:"workitem_page_id"`
	PageLink        string          `json:"page_collection_id"`
	Member          string          `json:"collection_member_id"`
	User            string          `json:"user_id"`
	Pages           *string         `json:"page_ids"`
	Name            *string         `json:"name"`
	HTML            *string         `json:"description_html"`
	Color           *string         `json:"color"`
	Source          *string         `json:"external_source"`
	External        *string         `json:"external_id"`
	Access          json.RawMessage `json:"access"`
	MemberAccess    string          `json:"member_access"`
	Locked          *bool           `json:"is_locked"`
	Archive         *bool           `json:"archive"`
	ArchivePages    *bool           `json:"archive_pages"`
	Order           *float64        `json:"sort_order"`
	Search          string          `json:"search"`
	Cursor          *string         `json:"cursor"`
	PerPage         *int            `json:"per_page"`
	Request         *string         `json:"request_id"`
	Revision        *string         `json:"expected_revision"`
	IncludeRevision bool            `json:"include_revision"`
}

func (s *Service) Page(ctx context.Context, input *api.PageRequest) (loom.Nullable[api.PageResult], error) {
	if input == nil {
		return loom.Nullable[api.PageResult]{}, pageInputError()
	}
	raw, err := s.pageOperation(ctx, "page", input)
	return decodeReleaseResult[api.PageResult](raw, err)
}
func (s *Service) Collection(ctx context.Context, input *api.CollectionRequest) (loom.Nullable[api.CollectionResult], error) {
	if input == nil {
		return loom.Nullable[api.CollectionResult]{}, pageInputError()
	}
	raw, err := s.pageOperation(ctx, "collection", input)
	return decodeReleaseResult[api.CollectionResult](raw, err)
}
func pageInputError() error {
	return api.MakeInvalidParams(errors.New("invalid page or collection input"))
}
func selectPage(tool string, input interface{ Validate() error }) (pageInput, pagereads.PageAPICommand, error) {
	var in pageInput
	cmd := pagereads.PageAPICommand{PageAPIQuery: pagereads.PageAPIQuery{Tool: tool, Params: map[string]string{}, Present: map[string]bool{}}, Fields: map[string]any{}}
	if input.Validate() != nil {
		return in, cmd, pageInputError()
	}
	raw, err := json.Marshal(input)
	if err != nil || json.Unmarshal(raw, &in) != nil {
		return in, cmd, pageInputError()
	}
	cmd.Action, cmd.IncludeRevision, cmd.ExpectedRevision, cmd.Search = in.Action, in.IncludeRevision, in.Revision, in.Search
	// Keep selectors separate: identical UUID strings may identify distinct fields.
	selectors := []struct {
		text   string
		target *uuid.UUID
	}{{in.Project, &cmd.ProjectID}, {in.Page, &cmd.PageID}, {in.Collection, &cmd.CollectionID}, {in.Parent, &cmd.ParentID}, {in.Item, &cmd.ItemID}, {in.Member, &cmd.MemberID}, {in.User, &cmd.UserID}, {in.ItemLink, &cmd.LinkID}, {in.PageLink, &cmd.LinkID}}
	for _, selector := range selectors {
		if selector.text != "" {
			*selector.target, err = typeUUID(selector.text)
			if err != nil {
				return in, cmd, err
			}
		}
	}
	if in.Request != nil {
		id, typeUUIDErr := typeUUID(*in.Request)
		if typeUUIDErr != nil {
			return in, cmd, typeUUIDErr
		}
		cmd.RequestID = &id
	}
	if in.Cursor != nil && *in.Cursor != "" {
		cmd.Params["cursor"], cmd.Present["cursor"] = *in.Cursor, true
	}
	if in.PerPage != nil && *in.PerPage != 0 {
		cmd.Params["per_page"], cmd.Present["per_page"] = strconv.Itoa(*in.PerPage), true
	}
	for key, value := range map[string]*string{"name": in.Name, "description_html": in.HTML, "color": in.Color, "external_source": in.Source, "external_id": in.External} {
		if value != nil && *value != "" {
			cmd.Fields[key] = *value
		}
	}
	for key, value := range map[string]*bool{"is_locked": in.Locked, "archive": in.Archive, "archive_pages": in.ArchivePages} {
		if value != nil {
			cmd.Fields[key] = *value
		}
	}
	if in.Order != nil {
		cmd.Fields["sort_order"] = *in.Order
	}
	if len(in.Access) > 0 && string(in.Access) != "null" {
		if tool == "page" {
			var access int
			if json.Unmarshal(in.Access, &access) != nil {
				return in, cmd, pageInputError()
			}
			cmd.Fields["access"] = access
		} else {
			var access string
			if json.Unmarshal(in.Access, &access) != nil {
				return in, cmd, pageInputError()
			}
			if access != "" {
				level, ok := map[string]int{"public": 0, "private": 1}[access]
				if !ok {
					return in, cmd, pageInputError()
				}
				cmd.Fields["access"] = level
			}
		}
	}
	if in.MemberAccess != "" {
		level, ok := map[string]int{"view": 0, "comment": 1, "edit": 2}[in.MemberAccess]
		if !ok {
			return in, cmd, pageInputError()
		}
		cmd.Fields["member_access"] = level
	}
	cmd.PageIDs, err = milestoneIDs(in.Pages)
	if err != nil {
		return in, cmd, pageInputError()
	}
	return in, cmd, nil
}
func (s *Service) pageOperation(ctx context.Context, tool string, input interface{ Validate() error }) (json.RawMessage, error) {
	in, cmd, err := selectPage(tool, input)
	if err != nil {
		return nil, err
	}
	write := cmd.Action != "list" && cmd.Action != "retrieve" && cmd.Action != "list_workitem_pages" && cmd.Action != "list_pages" && cmd.Action != "search_pages" && cmd.Action != "list_members"
	ctx, actor, err := actionActor(ctx, in.Workspace, write)
	if err != nil {
		return nil, err
	}
	cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	var raw json.RawMessage
	if !write {
		raw, err = s.pages.ReadPageAPI(ctx, cmd.PageAPIQuery)
	} else {
		var value projectmutations.Result
		value, err = s.mutatePage(ctx, actor, in.Workspace, cmd)
		if err == nil {
			raw = value.Record
			if in.Request != nil || in.Revision != nil {
				var body any
				if json.Unmarshal(raw, &body) != nil {
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
		return nil, pageOperationError(err)
	}
	var body any
	if json.Unmarshal(raw, &body) != nil {
		return nil, workitemResultError()
	}
	if !write && (cmd.Action == "list_pages" || tool == "page" && cmd.Action == "list") {
		row, ok := body.(map[string]any)
		if !ok {
			return nil, workitemResultError()
		}
		for key := range row {
			switch key {
			case "results", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results", "count", "total_count", "revision":
			default:
				delete(row, key)
			}
		}
	}
	return json.Marshal(projectNestedTimestamps(body, actor.Timezone))
}
func (s *Service) mutatePage(ctx context.Context, actor mcpauth.Principal, selector *string, cmd pagereads.PageAPICommand) (projectmutations.Result, error) {
	cmd.Revalidate = mutationRevalidator(actor, selector, true)
	return s.observeMutation(ctx, actor, cmd.Tool, cmd.Action, func(ctx context.Context) (projectmutations.Result, error) {
		return s.pages.MutatePageAPI(ctx, cmd)
	})
}

func pageOperationError(err error) error {
	var failure *pagereads.LifecycleError
	if errors.As(err, &failure) {
		switch failure.Status {
		case 400, 409:
			return api.MakeInvalidParams(errors.New("page operation rejected"))
		case 403:
			return api.MakeForbidden(errors.New("page access denied"))
		case 404:
			return api.MakeNotFound(errors.New("page unavailable"))
		}
	}
	return archiveResourceError(err)
}
