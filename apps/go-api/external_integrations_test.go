package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"

	api "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type externalIntegrationsTestRepository struct {
	configuration map[string]string
}

func (r externalIntegrationsTestRepository) Configuration(context.Context, []string, map[string]string, bool) (map[string]string, error) {
	return r.configuration, nil
}
func (externalIntegrationsTestRepository) WorkspaceRole(context.Context, string, string) (int, error) {
	return 0, nil
}
func (externalIntegrationsTestRepository) ProjectAllowed(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (externalIntegrationsTestRepository) WorkspaceLite(context.Context, string) (json.RawMessage, error) {
	return nil, nil
}
func (externalIntegrationsTestRepository) ProjectLite(context.Context, string) (json.RawMessage, error) {
	return nil, nil
}

type externalIntegrationsTestSessions struct{}

func (externalIntegrationsTestSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{}, nil
}

type externalIntegrationsRoundTripper func(*http.Request) (*http.Response, error)

func (f externalIntegrationsRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestUnsplashForwardsEncodedQueryAndUnusualJSONError(t *testing.T) {
	client := &http.Client{Transport: externalIntegrationsRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/search/photos/" {
			t.Fatalf("upstream path = %q", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("client_id") != "key&fragment" || query.Get("query") != "plane & colors" || query.Get("page") != "2" || query.Get("per_page") != "7" {
			t.Fatalf("upstream query = %v", query)
		}
		if got := request.URL.RawQuery; got != "client_id=key%26fragment&page=2&per_page=7&query=plane+%26+colors" {
			t.Fatalf("encoded query = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusTeapot,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`["upstream error",{"retry":false}]`)),
		}, nil
	})}
	service, err := NewExternalIntegrations(externalIntegrationsTestRepository{
		configuration: map[string]string{"UNSPLASH_ACCESS_KEY": "key&fragment"},
	}, externalIntegrationsTestSessions{}, client)
	if err != nil {
		t.Fatal(err)
	}
	query, page, perPage := "plane & colors", "2", "7"
	_, _, err = service.Unsplash(context.Background(), &api.UnsplashPayload{Query: &query, Page: &page, PerPage: &perPage})
	var response *api.ExternalIntegrationErrorResponse
	if !errors.As(err, &response) || response.LoomErrorName() != "status_418" {
		t.Fatalf("upstream error = %#v", err)
	}
	var got any
	if err := json.Unmarshal([]byte(response.Body), &got); err != nil {
		t.Fatalf("decode preserved upstream JSON body: %v", err)
	}
	want := []any{"upstream error", map[string]any{"retry": false}}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("upstream JSON body = %s, want %s", gotJSON, wantJSON)
	}
}

func TestExternalAIInvalidTypesAreRejectedBeforeUpstream(t *testing.T) {
	for _, body := range []string{`[1]`, `{"task":3,"prompt":"fixture"}`, `{"task":"Summarize"}`} {
		policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			_, err = parseExternalAIRequest(request.Context(), request.Body)
		})).ServeHTTP(httptest.NewRecorder(), request)
		if !errors.Is(err, errExternalAIExpectedObject) && !errors.Is(err, errExternalAIStringFields) {
			t.Fatalf("body=%s error=%v", body, err)
		}
	}
}

func TestExternalIntegrationsValidatesConfiguredUnsplashOrigin(t *testing.T) {
	for _, base := range []string{"file:///tmp/data", "http://user:secret@example.com", "https://example.com/?key=x"} {
		t.Setenv("UNSPLASH_API_BASE_URL", base)
		if _, err := NewExternalIntegrations(externalIntegrationsTestRepository{}, externalIntegrationsTestSessions{}, nil); err == nil {
			t.Fatalf("accepted base %q", base)
		}
	}
}

func TestUnsplashSuccessPreservesJSONValue(t *testing.T) {
	for _, payload := range []string{`[{"id":"photo"}]`, `{"results":[{"id":"photo"}]}`} {
		client := &http.Client{Transport: externalIntegrationsRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload))}, nil
		})}
		service, err := NewExternalIntegrations(externalIntegrationsTestRepository{configuration: map[string]string{"UNSPLASH_ACCESS_KEY": "fixture"}}, externalIntegrationsTestSessions{}, client)
		if err != nil {
			t.Fatal(err)
		}
		_, body, err := service.Unsplash(context.Background(), &api.UnsplashPayload{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(body)
		body.Close()
		if err != nil || string(got) != payload {
			t.Fatalf("body=%s error=%v, want %s", got, err, payload)
		}
	}
}
