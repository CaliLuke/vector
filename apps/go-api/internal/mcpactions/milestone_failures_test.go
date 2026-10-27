package mcpactions_test

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const milestoneTestID = "50000000-0000-4000-8000-000000000001"

func TestMilestoneFailuresAreTypedAndSanitized(t *testing.T) {
	for _, tc := range []struct {
		failure error
		code    string
	}{
		{milestones.ErrForbidden, "forbidden"}, {milestones.ErrNotFound, "not_found"}, {milestones.ErrInvalidInput, "invalid_params"},
		{milestones.ErrConflict, "conflict"}, {milestones.ErrRevisionConflict, "revision_conflict"}, {mutationreceipts.ErrConflict, "request_conflict"},
		{mutationreceipts.ErrInvalidRequest, "invalid_params"}, {mutationreceipts.ErrIndeterminate, "indeterminate"}, {errors.New("private-database-marker"), "internal_error"},
	} {
		for _, action := range []string{"list", "retrieve", "list_workitems", "create"} {
			t.Run(tc.code+"/"+action, func(t *testing.T) {
				f := newFixture(t)
				failure := fmt.Errorf("private-database-marker: %w", tc.failure)
				f.milestones.list = func(milestones.Scope, djangoapi.Query) (milestones.Page[milestones.Record], error) {
					return milestones.Page[milestones.Record]{}, failure
				}
				f.milestones.read = func(milestones.Scope, uuid.UUID, bool) (milestones.Record, error) {
					return milestones.Record{}, failure
				}
				f.milestones.links = func(milestones.Scope, uuid.UUID, djangoapi.Query) (milestones.Page[milestones.Link], error) {
					return milestones.Page[milestones.Link]{}, failure
				}
				f.milestones.write = func(context.Context, string, milestones.WriteOptions, uuid.UUID, milestones.Fields, []uuid.UUID, []uuid.UUID) (milestones.MutationResult, error) {
					return milestones.MutationResult{}, failure
				}
				input := `{"action":` + fmt.Sprintf("%q", action) + `,"project_id":"` + readProject + `"`
				if action == "retrieve" || action == "list_workitems" {
					input += `,"milestone_id":"` + milestoneTestID + `"`
				}
				if action == "create" {
					input += `,"title":"private-title-marker"`
				}
				result := rpc(t, f.handler(t, nil), "test-one", "tools/call", milestoneCall(input+`}`))
				require.True(t, result.IsError)
				require.Empty(t, result.StructuredContent)
				require.Contains(t, result.Content[0].Text, "["+tc.code+"]")
				require.NotContains(t, result.Content[0].Text, "private-")
			})
		}
	}
}

func milestoneContext(t *testing.T, f fixture) context.Context {
	t.Helper()
	request := httptest.NewRequest("POST", "https://vector.example/mcp/api-key", nil)
	request.Header.Set("Authorization", "Bearer test-one")
	request.Header.Set("X-Workspace-Slug", "workspace-one")
	ctx, err := f.auth.Authenticate(t.Context(), request)
	require.NoError(t, err)
	return ctx
}
func TestDirectMilestoneCallsRejectMalformedInputs(t *testing.T) {
	f := newFixture(t)
	ctx := milestoneContext(t, f)
	for _, input := range []*api.MilestoneRequest{
		nil, {},
		new(api.NewMilestoneRequestMilestoneList(nil)), new(api.NewMilestoneRequestMilestoneRetrieve(nil)), new(api.NewMilestoneRequestMilestoneCreate(nil)),
		new(api.NewMilestoneRequestMilestoneUpdate(nil)), new(api.NewMilestoneRequestMilestoneDelete(nil)), new(api.NewMilestoneRequestMilestoneListWorkitems(nil)), new(api.NewMilestoneRequestMilestoneManageWorkitems(nil)),
		new(api.NewMilestoneRequestMilestoneList(&api.MilestoneList{Action: "list", ProjectID: "bad"})),
		new(api.NewMilestoneRequestMilestoneList(&api.MilestoneList{Action: "list", ProjectID: readProject, PerPage: new(-1)})),
		new(api.NewMilestoneRequestMilestoneRetrieve(&api.MilestoneRetrieve{Action: "retrieve", ProjectID: readProject, MilestoneID: uuid.Nil.String()})),
		new(api.NewMilestoneRequestMilestoneCreate(&api.MilestoneCreate{Action: "create", ProjectID: "bad"})),
		new(api.NewMilestoneRequestMilestoneCreate(&api.MilestoneCreate{Action: "create", ProjectID: readProject, RequestID: new("bad")})),
		new(api.NewMilestoneRequestMilestoneManageWorkitems(&api.MilestoneManageWorkitems{Action: "manage_workitems", ProjectID: readProject, MilestoneID: milestoneTestID, AddIds: new("bad")})),
		new(api.NewMilestoneRequestMilestoneManageWorkitems(&api.MilestoneManageWorkitems{Action: "manage_workitems", ProjectID: readProject, MilestoneID: milestoneTestID, RemoveIds: new("bad")})),
	} {
		_, err := f.service.Milestone(ctx, input)
		require.Error(t, err)
	}
	for _, input := range []api.MilestoneRequest{
		api.NewMilestoneRequestMilestoneList(&api.MilestoneList{Action: "list", ProjectID: readProject}),
		api.NewMilestoneRequestMilestoneCreate(&api.MilestoneCreate{Action: "create", ProjectID: readProject, Title: "test"}),
	} {
		_, err := f.service.Milestone(t.Context(), &input)
		require.Error(t, err)
	}
}

func TestMilestoneExtensionsFailClosedWithoutDomainMetadata(t *testing.T) {
	for _, value := range []milestones.MutationResult{
		{}, {MilestoneID: milestoneTestID, Revision: "revision"},
		{MilestoneID: "bad", Revision: "revision", Result: &milestones.Record{}},
		{MilestoneID: milestoneTestID, Result: &milestones.Record{}},
		{MilestoneID: milestoneTestID, Revision: "revision", Result: &milestones.Record{}},
	} {
		f := newFixture(t)
		f.milestones.write = func(context.Context, string, milestones.WriteOptions, uuid.UUID, milestones.Fields, []uuid.UUID, []uuid.UUID) (milestones.MutationResult, error) {
			return value, nil
		}
		result := rpc(t, f.handler(t, nil), "test-one", "tools/call", milestoneCall(`{"action":"create","project_id":"`+readProject+`","title":"test","request_id":"60000000-0000-4000-8000-000000000001"}`))
		require.True(t, result.IsError)
		require.Contains(t, result.Content[0].Text, "[internal_error]")
	}
	f := newFixture(t)
	f.milestones.read = func(milestones.Scope, uuid.UUID, bool) (milestones.Record, error) { return milestones.Record{}, nil }
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", milestoneCall(`{"action":"retrieve","project_id":"`+readProject+`","milestone_id":"`+milestoneTestID+`","include_revision":true}`))
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].Text, "[internal_error]")
}

func TestMilestoneRevalidationRejectsChangedActor(t *testing.T) {
	f := newFixture(t)
	f.milestones.write = func(ctx context.Context, _ string, options milestones.WriteOptions, _ uuid.UUID, _ milestones.Fields, _, _ []uuid.UUID) (milestones.MutationResult, error) {
		f.keys.owners["test-one"] = f.keys.owners["test-two"]
		return milestones.MutationResult{}, options.Revalidate(ctx)
	}
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", milestoneCall(`{"action":"create","project_id":"`+readProject+`","title":"test"}`))
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].Text, "[forbidden]")
}
