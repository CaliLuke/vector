package design_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	client "github.com/CaliLuke/go-plane/ticket_collaboration/gen/http/ticket_collaboration/client"
	api "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	loomhttp "github.com/CaliLuke/loom/http"
)

const collaborationIssueID = "01234567-89ab-4def-8123-456789abcdef"

type cookieClientDoer struct{ client *http.Client }

func (d cookieClientDoer) Do(request *http.Request) (*http.Response, error) {
	return d.client.Do(request)
}

type sessionClientDoer struct{ client *http.Client }

func (d sessionClientDoer) Do(request *http.Request) (*http.Response, error) {
	request.Header.Set("Cookie", "session-id=session")
	return d.client.Do(request)
}

type statusCapturingDoer struct {
	doer   loomhttp.Doer
	status int
}

func (d *statusCapturingDoer) Do(request *http.Request) (*http.Response, error) {
	response, err := d.doer.Do(request)
	if response != nil {
		d.status = response.StatusCode
	}
	return response, err
}

func generatedHTTPClient(remote *httptest.Server, doer loomhttp.Doer) *client.Client {
	host := strings.TrimPrefix(strings.TrimPrefix(remote.URL, "http://"), "https://")
	scheme := "http"
	if strings.HasPrefix(remote.URL, "https://") {
		scheme = "https"
	}
	return client.NewClient(scheme, host, doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, true)
}

func TestGeneratedClientAllowsAnonymousPublicCommentsAndProtectsReactions(t *testing.T) {
	service := &publicTransportService{allow: false}
	remote := httptest.NewServer(publicTransport(t, service))
	defer remote.Close()
	transport := generatedHTTPClient(remote, cookieClientDoer{client: remote.Client()})

	comments, err := transport.API.GetAPIPublicAnchorByAnchorIssuesByIssueIDComments()(context.Background(), &api.GetAPIPublicAnchorByAnchorIssuesByIssueIDCommentsPayload{
		Anchor:  "board",
		IssueID: collaborationIssueID,
	})
	if err != nil {
		t.Fatalf("anonymous public comment request failed: %v", err)
	}
	commentResponse, ok := comments.(*api.GetAPIPublicAnchorByAnchorIssuesByIssueIDCommentsResponseData)
	if !ok {
		t.Fatalf("anonymous comment response has type %T", comments)
	}
	defer testsupport.Close(t, commentResponse.Body)
	commentBody, err := io.ReadAll(commentResponse.Body)
	if err != nil || string(commentBody) != "[]" || !service.called {
		t.Fatalf("anonymous comments body=%q called=%v err=%v", commentBody, service.called, err)
	}

	service.called = false
	_, err = transport.API.GetAPIPublicAnchorByAnchorIssuesByIssueIDReactions()(context.Background(), &api.GetAPIPublicAnchorByAnchorIssuesByIssueIDReactionsPayload{
		Anchor:  "board",
		IssueID: collaborationIssueID,
	})
	var collaborationErr *api.CollaborationError
	if !errors.As(err, &collaborationErr) || !strings.Contains(string(collaborationErr.Body), "Authentication credentials were not provided.") || service.called {
		t.Fatalf("anonymous reaction error=%#v called=%v, want 401 without entering service", err, service.called)
	}
}

func (s *publicTransportService) PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe(_ context.Context, _ *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribePayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	data, readErr := io.ReadAll(body)
	err := errors.Join(readErr, body.Close())
	s.body, s.called = string(data), true
	return &api.CollaborationResponse{}, io.NopCloser(strings.NewReader(`{"subscribed":true}`)), err
}

func TestGeneratedClientReceivesSelfSubscribeCreatedResponse(t *testing.T) {
	service := &publicTransportService{allow: true}
	remote := httptest.NewServer(publicTransport(t, service))
	defer remote.Close()
	doer := &statusCapturingDoer{doer: sessionClientDoer{client: remote.Client()}}
	transport := generatedHTTPClient(remote, doer)
	const requestBody = `{"ignored":true}`

	value, err := transport.API.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribe()(context.Background(), &api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribeRequestData{
		Payload: &api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribePayload{
			Slug: "demo", ProjectID: "11111111-1111-4111-8111-111111111111", IssueID: collaborationIssueID,
		},
		Body: io.NopCloser(strings.NewReader(requestBody)),
	})
	if err != nil {
		t.Fatalf("generated self-subscribe POST failed: %v", err)
	}
	response, ok := value.(*api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDSubscribeResponseData)
	if !ok {
		t.Fatalf("self-subscribe response has type %T", value)
	}
	defer testsupport.Close(t, response.Body)
	body, err := io.ReadAll(response.Body)
	if err != nil || string(body) != `{"subscribed":true}` || doer.status != http.StatusCreated || !service.called || service.body != requestBody {
		t.Fatalf("status=%d called=%v request=%q response=%q err=%v", doer.status, service.called, service.body, body, err)
	}
}

func TestTicketCollaborationOpenAPIPreservesPublicAndPathContracts(t *testing.T) {
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]struct {
			Security   []map[string][]string `json:"security"`
			Parameters []struct {
				Ref    string `json:"$ref"`
				Name   string `json:"name"`
				Schema struct {
					Type   string `json:"type"`
					Format string `json:"format"`
				} `json:"schema"`
			} `json:"parameters"`
			Responses map[string]json.RawMessage `json:"responses"`
		} `json:"paths"`
		Components struct {
			Parameters map[string]struct {
				Name   string `json:"name"`
				Schema struct {
					Type   string `json:"type"`
					Format string `json:"format"`
				} `json:"schema"`
			} `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	operation := func(path, method string) struct {
		Security   []map[string][]string `json:"security"`
		Parameters []struct {
			Ref    string `json:"$ref"`
			Name   string `json:"name"`
			Schema struct {
				Type   string `json:"type"`
				Format string `json:"format"`
			} `json:"schema"`
		} `json:"parameters"`
		Responses map[string]json.RawMessage `json:"responses"`
	} {
		t.Helper()
		methods, ok := document.Paths[path]
		if !ok {
			t.Fatalf("OpenAPI path %s is missing", path)
		}
		value, ok := methods[method]
		if !ok {
			t.Fatalf("OpenAPI operation %s %s is missing", method, path)
		}
		return value
	}

	commentsPath := "/api/public/anchor/{anchor}/issues/{issue_id}/comments/"
	comments := operation(commentsPath, "get")
	if !reflect.DeepEqual(comments.Security, []map[string][]string{{}, {"PlaneSessionCookie": {}}}) {
		t.Fatalf("public comment GET security=%#v, want anonymous or session-cookie", comments.Security)
	}
	for _, tc := range []struct{ path, method string }{
		{commentsPath, "post"},
		{"/api/public/anchor/{anchor}/issues/{issue_id}/reactions/", "get"},
	} {
		secured := operation(tc.path, tc.method)
		if !reflect.DeepEqual(secured.Security, []map[string][]string{{"PlaneSessionCookie": {}}}) {
			t.Fatalf("%s %s security=%#v, want required session cookie", tc.method, tc.path, secured.Security)
		}
	}

	pathSchema := func(op struct {
		Security   []map[string][]string `json:"security"`
		Parameters []struct {
			Ref    string `json:"$ref"`
			Name   string `json:"name"`
			Schema struct {
				Type   string `json:"type"`
				Format string `json:"format"`
			} `json:"schema"`
		} `json:"parameters"`
		Responses map[string]json.RawMessage `json:"responses"`
	}, name string) (string, string) {
		t.Helper()
		for _, parameter := range op.Parameters {
			if parameter.Name == name {
				return parameter.Schema.Type, parameter.Schema.Format
			}
			if strings.HasPrefix(parameter.Ref, "#/components/parameters/") {
				ref := strings.TrimPrefix(parameter.Ref, "#/components/parameters/")
				resolved, ok := document.Components.Parameters[ref]
				if ok && resolved.Name == name {
					return resolved.Schema.Type, resolved.Schema.Format
				}
			}
		}
		t.Fatalf("path parameter %q not found", name)
		return "", ""
	}
	for _, tc := range []struct {
		name, wantType, wantFormat string
	}{
		{"anchor", "string", ""},
		{"issue_id", "string", "uuid"},
	} {
		if typ, format := pathSchema(comments, tc.name); typ != tc.wantType || format != tc.wantFormat {
			t.Fatalf("comment GET path parameter %s has %s/%s, want %s/%s", tc.name, typ, format, tc.wantType, tc.wantFormat)
		}
	}
	reactionDelete := operation("/api/public/anchor/{anchor}/issues/{issue_id}/reactions/{reaction_code}/", "delete")
	if typ, format := pathSchema(reactionDelete, "reaction_code"); typ != "string" || format != "" {
		t.Fatalf("reaction_code path schema=%s/%s, want string", typ, format)
	}

	selfSubscribe := operation("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/subscribe/", "post")
	if !reflect.DeepEqual(selfSubscribe.Security, []map[string][]string{{"PlaneSessionCookie": {}}}) {
		t.Fatalf("self-subscribe POST security=%#v, want required session cookie", selfSubscribe.Security)
	}
	if _, ok := selfSubscribe.Responses["201"]; !ok {
		t.Fatalf("self-subscribe POST responses lack 201: %v", selfSubscribe.Responses)
	}
}
