package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

type collaborationReadStub struct {
	value any
	err   error
}

func (s *collaborationReadStub) ExecuteCollaboration(context.Context, workitems.Request, io.Reader, string) (workitems.Result, error) {
	return workitems.Result{Status: 200, Body: s.value}, s.err
}

type activityReadStub struct {
	value any
	err   error
}

func (s *activityReadStub) Read(context.Context, activityreads.Principal, activityreads.Request) (activityreads.Result, error) {
	return activityreads.Result{Body: s.value}, s.err
}

const readIssue = "80000000-0000-4000-8000-000000000001"
const readProject = "30000000-0000-4000-8000-000000000001"
const readChild = "90000000-0000-4000-8000-000000000001"

func itemReadInput(tool, action string) string {
	input := map[string]any{"action": action, "project_id": readProject, "workitem_id": readIssue}
	if action == "retrieve" {
		input[tool[len("workitem_"):]+"_id"] = readChild
	}
	body, _ := json.Marshal(input)
	return string(body)
}

func TestCommentMentionsMatchPinnedHandler(t *testing.T) {
	body, err := os.ReadFile("../../mcp/testdata/plane-mcp/comment_mentions.json")
	require.NoError(t, err)
	var golden struct {
		Cases []struct{ Name, Input, Expected string }
	}
	require.NoError(t, json.Unmarshal(body, &golden))
	require.Len(t, golden.Cases, 23)
	for _, tc := range golden.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			f := newFixture(t)
			f.collaborationReads.value = map[string]any{"comment_html": tc.Input}
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", resourceCall("workitem_comment", itemReadInput("workitem_comment", "retrieve")))
			require.False(t, result.IsError, result.Content)
			var row map[string]any
			require.NoError(t, json.Unmarshal(result.StructuredContent, &row))
			require.Equal(t, tc.Expected, row["comment_html"])
		})
	}
}

func TestWorkitemReadFailuresAreSanitized(t *testing.T) {
	for _, tool := range []string{"workitem_comment", "workitem_link", "workitem_activity"} {
		for _, tc := range []struct {
			err  error
			code string
		}{
			{&djangoapi.HTTPError{Status: 400}, "invalid_params"},
			{&djangoapi.HTTPError{Status: 403}, "forbidden"},
			{&djangoapi.HTTPError{Status: 404}, "not_found"},
			{errors.New("private database secret"), "internal_error"},
		} {
			t.Run(tool+"/"+tc.code, func(t *testing.T) {
				f := newFixture(t)
				f.collaborationReads.err = tc.err
				f.activityReads.err = tc.err
				result := rpc(t, f.handler(t, nil), "test-one", "tools/call", resourceCall(tool, itemReadInput(tool, "list")))
				require.True(t, result.IsError)
				require.Contains(t, result.Content[0].Text, "["+tc.code+"]")
				require.NotContains(t, result.Content[0].Text, "secret")
				require.Empty(t, result.StructuredContent)
			})
		}
	}
}

func TestWorkitemReadsRejectMalformedRecords(t *testing.T) {
	for _, tc := range []struct {
		tool  string
		value any
	}{
		{"workitem_comment", map[string]any{"access": "PUBLIC"}},
		{"workitem_comment", map[string]any{"comment_html": 12}},
		{"workitem_comment", map[string]any{"attachments": make(chan int)}},
		{"workitem_link", map[string]any{}},
		{"workitem_link", map[string]any{"url": "https://example.test", "title": 12}},
		{"workitem_link", map[string]any{"url": "https://example.test", "metadata": make(chan int)}},
		{"workitem_activity", map[string]any{}},
		{"workitem_activity", map[string]any{"project": readProject, "workspace": "workspace", "epoch": 1.5}},
		{"workitem_activity", map[string]any{"project": readProject, "workspace": "workspace", "verb": 12}},
		{"workitem_activity", map[string]any{"project": readProject, "workspace": "workspace", "attachments": make(chan int)}},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			f := newFixture(t)
			f.collaborationReads.value = tc.value
			f.activityReads.value = tc.value
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", resourceCall(tc.tool, itemReadInput(tc.tool, "retrieve")))
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].Text, "[internal_error]")
		})
	}
}

func TestDirectWorkitemReadsRejectInvalidRequests(t *testing.T) {
	f := newFixture(t)
	exerciseChildInputs(t, f.service.WorkitemComment, api.NewWorkitemCommentRequestWorkitemReadList, api.NewWorkitemCommentRequestWorkitemCommentRetrieve)
	exerciseChildInputs(t, f.service.WorkitemLink, api.NewWorkitemLinkRequestWorkitemReadList, api.NewWorkitemLinkRequestWorkitemLinkRetrieve)
	exerciseChildInputs(t, f.service.WorkitemActivity, api.NewWorkitemActivityRequestWorkitemReadList, api.NewWorkitemActivityRequestWorkitemActivityRetrieve)
}

func exerciseChildInputs[Request, List, Detail, Result any](t *testing.T, call func(context.Context, *Request) (*Result, error), list func(*List) Request, detail func(*Detail) Request) {
	t.Helper()
	for _, input := range []*Request{nil, new(Request)} {
		_, err := call(t.Context(), input)
		require.Error(t, err)
	}
	emptyList, emptyDetail := list(nil), detail(nil)
	for _, input := range []*Request{&emptyList, &emptyDetail} {
		_, err := call(t.Context(), input)
		require.Error(t, err)
	}
	for _, raw := range []string{
		`{"action":"wrong"}`, `{"action":"list","project_id":"invalid"}`,
		`{"action":"list","project_id":"30000000-0000-4000-8000-000000000001","workitem_id":"invalid"}`,
		`{"action":"list","project_id":"30000000-0000-4000-8000-000000000001","workitem_id":"80000000-0000-4000-8000-000000000001","per_page":-1}`,
		`{"action":"list","project_id":"30000000-0000-4000-8000-000000000001","workitem_id":"80000000-0000-4000-8000-000000000001","per_page":1001}`,
		`{"action":"list","project_id":"30000000-0000-4000-8000-000000000001","workitem_id":"80000000-0000-4000-8000-000000000001","per_page":0,"cursor":""}`,
	} {
		var branch List
		require.NoError(t, json.Unmarshal([]byte(raw), &branch))
		input := list(&branch)
		_, err := call(t.Context(), &input)
		require.Error(t, err)
	}
	for _, raw := range []string{`{"action":"wrong"}`, `{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001","workitem_id":"80000000-0000-4000-8000-000000000001","comment_id":"invalid","link_id":"invalid","activity_id":"invalid"}`} {
		var branch Detail
		require.NoError(t, json.Unmarshal([]byte(raw), &branch))
		input := detail(&branch)
		_, err := call(t.Context(), &input)
		require.Error(t, err)
	}
}
