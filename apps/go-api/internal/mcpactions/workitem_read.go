package mcpactions

import (
	"context"
	"errors"
	"io"
	"strconv"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitems"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	googleuuid "github.com/google/uuid"
)

type collaborationReader interface {
	ReadIntake(context.Context, workitems.Request) (workitems.Result, error)
	MutateIntake(context.Context, projectmutations.Command) (projectmutations.Result, error)
	MutatePropertyValue(context.Context, projectmutations.Command) (projectmutations.Result, error)
	ReadWorkitem(context.Context, workitems.Request) (workitems.Result, error)
	MutateWorkitem(context.Context, projectmutations.Command) (projectmutations.Result, error)
	MutateComment(context.Context, projectmutations.Command) (projectmutations.Result, error)
	MutateLink(context.Context, projectmutations.Command) (projectmutations.Result, error)
	ExecuteCollaboration(context.Context, workitems.Request, io.Reader, string) (workitems.Result, error)
}
type activityReader interface {
	Read(context.Context, activityreads.Principal, activityreads.Request) (activityreads.Result, error)
}
type collaborationRead struct {
	reader collaborationReader
	kind   string
}

func (r collaborationRead) ReadAPIV1(ctx context.Context, user string, q djangoapi.ProjectReadRequest) (any, error) {
	workspace := googleuuid.UUID(*q.ExpectedWorkspaceID)
	result, err := r.reader.ExecuteCollaboration(ctx, workitems.Request{
		ExpectedWorkspaceID: &workspace, API: true, Method: "GET", Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_" + r.kind,
		ActorID: user, Slug: q.Params["slug"], ProjectID: q.Params["project_id"], IssueID: q.Params["issue_id"], Timezone: "UTC", Params: q.Params, Present: q.Present,
	}, nil, "")
	return result.Body, err
}

type activityRead struct{ reader activityReader }

func (r activityRead) ReadAPIV1(ctx context.Context, user string, q djangoapi.ProjectReadRequest) (any, error) {
	workspace := googleuuid.UUID(*q.ExpectedWorkspaceID)
	operation := "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities"
	if q.Operation == "detail" {
		operation += "_by_pk"
	}
	result, err := r.reader.Read(ctx, activityreads.Principal{UserID: googleuuid.MustParse(user), Authenticated: true, Timezone: "UTC"}, activityreads.Request{
		ExpectedWorkspaceID: &workspace, Operation: operation, Params: q.Params, Present: q.Present,
	})
	return result.Body, err
}

type itemReadSelection struct {
	action, project, issue, target string
	workspace, cursor              *string
	perPage                        *int
	includeRevision                bool
}

func selectItemRead(q itemReadSelection) (djangoapi.ProjectReadRequest, *string, error) {
	request := djangoapi.ProjectReadRequest{Params: map[string]string{"project_id": q.project, "issue_id": q.issue}, Present: map[string]bool{}}
	valid := q.action == "list" || q.action == "retrieve"
	for _, id := range []string{q.project, q.issue} {
		if _, err := uuid.Parse(id); err != nil {
			valid = false
		}
	}
	request.Operation = "list"
	if q.action == "retrieve" {
		request.Operation = "detail"
		request.Params["pk"] = q.target
		if q.includeRevision {
			request.Params["include_revision"], request.Present["include_revision"] = "true", true
		}
		if _, err := uuid.Parse(q.target); err != nil {
			valid = false
		}
	}
	if q.cursor != nil && *q.cursor != "" {
		request.Params["cursor"], request.Present["cursor"] = *q.cursor, true
	}
	if q.perPage != nil && *q.perPage != 0 {
		request.Params["per_page"], request.Present["per_page"] = strconv.Itoa(*q.perPage), true
		valid = valid && *q.perPage > 0 && *q.perPage <= 1000
	}
	if !valid {
		return djangoapi.ProjectReadRequest{}, nil, api.MakeInvalidParams(errors.New("invalid work-item read action"))
	}
	return request, q.workspace, nil
}

// All three tools share the same list arguments and selection rules.
type itemReadRequest interface {
	AsWorkitemReadList() (*api.WorkitemReadList, bool)
}

func selectItemReadUnion[Input itemReadRequest](input *Input, detail func(Input) itemReadSelection) (djangoapi.ProjectReadRequest, *string, error) {
	selection := itemReadSelection{}
	if input != nil {
		if list, ok := (*input).AsWorkitemReadList(); ok && list != nil && list.Action == "list" {
			selection = itemReadSelection{action: list.Action, project: list.ProjectID, issue: list.WorkitemID, workspace: list.WorkspaceSlug, cursor: list.Cursor, perPage: list.PerPage}
		} else {
			selection = detail(*input)
		}
	}
	return selectItemRead(selection)
}

func retrieveItemSelection(action, project, issue, target string, workspace *string, revision *bool) itemReadSelection {
	if action != "retrieve" {
		return itemReadSelection{}
	}
	return itemReadSelection{action: action, project: project, issue: issue, target: target, workspace: workspace, includeRevision: revision != nil && *revision}
}
