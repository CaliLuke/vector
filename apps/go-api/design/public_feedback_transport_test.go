package design_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "github.com/CaliLuke/go-plane/gen/http/public_feedback/client"
	server "github.com/CaliLuke/go-plane/gen/http/public_feedback/server"
	api "github.com/CaliLuke/go-plane/gen/public_feedback"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type publicFeedbackTransportService struct {
	api.Service
	failure         error
	body            string
	omitContentType bool
}

func (service *publicFeedbackTransportService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (service *publicFeedbackTransportService) IssueDetail(_ context.Context, _ *api.IssueDetailPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	if service.failure != nil {
		return nil, nil, service.failure
	}
	metadata := &api.PublicFeedbackResponseMetadata{}
	if !service.omitContentType {
		contentType := "application/json"
		metadata.ContentType = &contentType
	}
	return metadata, io.NopCloser(strings.NewReader(service.body)), nil
}

func publicFeedbackTransport(t *testing.T, service *publicFeedbackTransportService) http.Handler {
	t.Helper()
	mux := loomhttp.NewMuxer()
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	generated.Mount(mux)
	return mux
}

func TestPublicFeedbackGeneratedTransportPreservesNullableDetail(t *testing.T) {
	service := &publicFeedbackTransportService{body: `null`}
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/fixture/issues/00000000-0000-4000-8000-000000000001/", nil)
	response := httptest.NewRecorder()
	publicFeedbackTransport(t, service).ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != `null` {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestPublicFeedbackGeneratedTransportOmitsAbsentContentTypeMetadata(t *testing.T) {
	service := &publicFeedbackTransportService{omitContentType: true}
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/fixture/issues/00000000-0000-4000-8000-000000000001/", nil)
	response := httptest.NewRecorder()
	publicFeedbackTransport(t, service).ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Content-Type") != "" {
		t.Fatalf("status=%d body=%q content-type=%q", response.Code, response.Body.String(), response.Header().Get("Content-Type"))
	}
}

func TestPublicFeedbackGeneratedTransportPreservesErrorObjectKeys(t *testing.T) {
	service := &publicFeedbackTransportService{
		failure: &api.PublicFeedbackError{Kind: "status_400", Body: loom.JSONValue(`{"issue_id":["Invalid pk."]}`)},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/fixture/issues/invalid/", nil)
	response := httptest.NewRecorder()
	publicFeedbackTransport(t, service).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"issue_id":["Invalid pk."]}` {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestPublicFeedbackGeneratedClientPreservesNullableDetail(t *testing.T) {
	service := &publicFeedbackTransportService{body: `null`}
	remote := httptest.NewServer(publicFeedbackTransport(t, service))
	defer remote.Close()
	transport := client.NewClient("http", strings.TrimPrefix(remote.URL, "http://"), remote.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	result, err := transport.API.IssueDetail()(context.Background(), &api.IssueDetailPayload{Anchor: "fixture", IssueID: "00000000-0000-4000-8000-000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(result.(*api.IssueDetailResponseData).Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `null` {
		t.Fatalf("body=%q", body)
	}
}

func TestPublicFeedbackGeneratedClientPreservesErrorObjectKeys(t *testing.T) {
	service := &publicFeedbackTransportService{
		failure: &api.PublicFeedbackError{Kind: "status_400", Body: loom.JSONValue(`{"issue_id":["Invalid pk."]}`)},
	}
	remote := httptest.NewServer(publicFeedbackTransport(t, service))
	defer remote.Close()
	transport := client.NewClient("http", strings.TrimPrefix(remote.URL, "http://"), remote.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	_, err := transport.API.IssueDetail()(context.Background(), &api.IssueDetailPayload{Anchor: "fixture", IssueID: "invalid"})
	var publicFeedbackError *api.PublicFeedbackError
	if !errors.As(err, &publicFeedbackError) || strings.TrimSpace(string(publicFeedbackError.Body)) != `{"issue_id":["Invalid pk."]}` {
		t.Fatalf("error=%T %v", err, err)
	}
}
