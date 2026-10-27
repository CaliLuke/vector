package mcptransport_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memberResponseService struct {
	body    string
	failure error
}

func (s memberResponseService) Member(context.Context, *api.MemberRequest) (*api.MemberResult, error) {
	if s.failure != nil {
		return nil, s.failure
	}
	var result api.MemberResult
	err := json.Unmarshal([]byte(s.body), &result)
	return &result, err
}

const identityResult = `{"id":"10000000-0000-4000-8000-000000000001","first_name":null,"last_name":null,"email":null,"avatar":null,"avatar_url":null,"display_name":null}`
const rosterResult = `[{"id":"10000000-0000-4000-8000-000000000001","first_name":null,"last_name":null,"email":null,"avatar":null,"avatar_url":null,"display_name":null,"role":null,"role_slug":null,"is_active":null,"is_bot":null}]`

func TestGeneratedMemberResultsPreservePinnedContentAcrossRevisions(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, tc := range []struct {
			name, body, action string
			failure            error
			contentCount       int
		}{
			{"identity", identityResult, "me", nil, 1},
			{"roster", rosterResult, "list_project", nil, 1},
			{"empty roster", `[]`, "list_project", nil, 0},
			{"domain failure", "", "list_project", api.MakeForbidden(errors.New("safe denial")), 1},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				opts := options()
				opts.Service = memberResponseService{tc.body, tc.failure}
				h, err := mcptransport.New(config("header"), opts, http.NotFoundHandler())
				require.NoError(t, err)
				arguments := map[string]any{"action": tc.action}
				if tc.action == "list_project" {
					arguments["project_id"] = "30000000-0000-4000-8000-000000000001"
				}
				params := map[string]any{"name": "member", "arguments": arguments}
				headers := http.Header{"Mcp-Protocol-Version": []string{version}}
				if version == "2026-07-28" {
					params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": version, "io.modelcontextprotocol/clientInfo": map[string]string{"name": "test", "version": "1"}, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}
					headers.Set("Mcp-Method", "tools/call")
					headers.Set("Mcp-Name", "member")
				}
				body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": params})
				require.NoError(t, err)
				response := call(h, "/mcp/api-key", string(body), headers)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				var message struct {
					Error  json.RawMessage
					Result struct {
						Content           []struct{ Text string }
						StructuredContent json.RawMessage
						IsError           bool
					}
				}
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &message))
				require.Empty(t, message.Error)
				require.Len(t, message.Result.Content, tc.contentCount)
				assert.Equal(t, tc.failure != nil, message.Result.IsError)
				if tc.action == "me" {
					assert.JSONEq(t, tc.body, string(message.Result.StructuredContent))
				} else {
					assert.Empty(t, message.Result.StructuredContent)
				}
				if tc.failure == nil && tc.contentCount != 0 {
					assert.JSONEq(t, tc.body, message.Result.Content[0].Text)
				}
			})
		}
	}
}

func (memberResponseService) Project(context.Context, *api.ProjectRequest) (loom.Nullable[api.ProjectResult], error) {
	return loom.Nullable[api.ProjectResult]{}, errors.New("this fixture does not execute project reads")
}

func (memberResponseService) DiscoverActions(context.Context) (mcptransport.ActionVisibility, error) {
	return mcptransport.ActionVisibility{"member": {"me": true, "list_workspace": true, "list_project": true, "list_roles": true, "retrieve_role": true}, "project": {"list": true, "retrieve": true, "archive": true, "unarchive": true}}, nil
}

type nullProjectService struct{ transportTestService }

func (nullProjectService) Project(context.Context, *api.ProjectRequest) (loom.Nullable[api.ProjectResult], error) {
	return loom.NullValue[api.ProjectResult](), nil
}

func TestGeneratedNullResultPreservesPinnedEmptyContent(t *testing.T) {
	opts := options()
	opts.Service = nullProjectService{}
	h, err := mcptransport.New(config("header"), opts, http.NotFoundHandler())
	require.NoError(t, err)
	response := call(h, "/mcp/api-key", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"project","arguments":{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001"}}}`, http.Header{"Mcp-Protocol-Version": {"2025-11-25"}})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var message struct {
		Result struct {
			Content           []json.RawMessage
			StructuredContent json.RawMessage
			IsError           bool
		}
		Error json.RawMessage
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &message))
	require.Empty(t, message.Error)
	require.False(t, message.Result.IsError)
	require.Empty(t, message.Result.Content)
	require.Empty(t, message.Result.StructuredContent)
}

func TestEnabledTransportRequiresDiscoveryAuthorization(t *testing.T) {
	opts := options()
	// Restrict the exposed method set to the generated service contract.
	opts.Service = struct{ api.Service }{transportTestService{}}
	_, err := mcptransport.New(config("header"), opts, http.NotFoundHandler())
	require.ErrorContains(t, err, "discovery authorization is required")
}

func (memberResponseService) Cycle(context.Context, *api.CycleRequest) (*api.CycleResult, error) {
	return nil, errors.New("this fixture does not execute cycle reads")
}

func (memberResponseService) Module(context.Context, *api.ModuleRequest) (*api.ModuleResult, error) {
	return nil, nil
}

func (memberResponseService) Label(context.Context, *api.LabelRequest) (*api.LabelResult, error) {
	return nil, nil
}

func (memberResponseService) WorkitemComment(context.Context, *api.WorkitemCommentRequest) (*api.WorkitemCommentResult, error) {
	return nil, nil
}

func (memberResponseService) WorkitemLink(context.Context, *api.WorkitemLinkRequest) (*api.WorkitemLinkResult, error) {
	return nil, nil
}

func (memberResponseService) WorkitemActivity(context.Context, *api.WorkitemActivityRequest) (*api.WorkitemActivityResult, error) {
	return nil, nil
}

func (memberResponseService) Milestone(context.Context, *api.MilestoneRequest) (loom.Nullable[api.MilestoneResult], error) {
	return loom.NullValue[api.MilestoneResult](), nil
}
