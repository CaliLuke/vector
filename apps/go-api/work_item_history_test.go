package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workitemhistory"
	api "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	"github.com/google/uuid"
)

type workItemHistorySessionStub struct{ principal djangosession.Principal }

func (stub workItemHistorySessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return stub.principal, nil
}

type workItemHistoryRepositoryStub struct {
	request       store.Request
	result        json.RawMessage
	err           error
	entityAllowed bool
}

func (stub *workItemHistoryRepositoryStub) Read(_ context.Context, request store.Request) (json.RawMessage, error) {
	stub.request = request
	return stub.result, stub.err
}

func (stub *workItemHistoryRepositoryStub) ProjectEntityAllowed(context.Context, string, string, uuid.UUID) (bool, error) {
	return stub.entityAllowed, nil
}

func TestWorkItemHistoryReadReturnsRawJSONAndMapsAuthorization(t *testing.T) {
	userID := uuid.MustParse("59961101-634c-4f9e-a436-5f8ec26e5f7b")
	projectID := "1d989b61-12b9-4e3c-9fe3-1491efef66be"
	issueID := "f74111b0-0dc6-4d3e-96e1-0c8d32471ff5"
	repository := &workItemHistoryRepositoryStub{result: json.RawMessage(`[{"id":"x"}]`), entityAllowed: true}
	service, err := NewWorkItemHistory(repository, workItemHistorySessionStub{principal: djangosession.Principal{UserID: userID.String(), UserTimezone: "UTC"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(*workItemHistoryService).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	metadata, body, err := service.History(ctx, &api.HistoryPayload{Slug: "sample", ProjectID: projectID, IssueID: issueID})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `[{"id":"x"}]` || metadata.ContentType != "application/json" {
		t.Fatalf("response metadata/body = %#v %s", metadata, encoded)
	}
	if repository.request.Operation != "history" || repository.request.Slug != "sample" || repository.request.ProjectID.String() != projectID || repository.request.ResourceID.String() != issueID || repository.request.UserID != userID {
		t.Fatalf("repository request = %#v", repository.request)
	}

	repository.err = store.ErrForbidden
	_, _, err = service.History(ctx, &api.HistoryPayload{Slug: "sample", ProjectID: projectID, IssueID: issueID})
	var response *api.WorkItemHistoryErrorResponse
	if !errors.As(err, &response) || response.Name != "status_403" {
		t.Fatalf("forbidden error = %v", err)
	}
	encoded, _ = json.Marshal(response.Body)
	if !strings.Contains(string(encoded), "required permissions") {
		t.Fatalf("forbidden body = %s", encoded)
	}
}

func TestWorkItemHistoryProtocolAppliesActivityClassPermissionOnly(t *testing.T) {
	repository := &workItemHistoryRepositoryStub{result: json.RawMessage(`[]`), entityAllowed: true}
	service, err := NewWorkItemHistory(repository, workItemHistorySessionStub{principal: djangosession.Principal{
		UserID: "59961101-634c-4f9e-a436-5f8ec26e5f7b", UserTimezone: "UTC",
	}})
	if err != nil {
		t.Fatal(err)
	}
	concrete := service.(*workItemHistoryService)
	ctx, err := concrete.APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	resource := SessionProtocolResource{
		Method:    "OPTIONS",
		Pattern:   "/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/",
		Slug:      "sample",
		ProjectID: "1d989b61-12b9-4e3c-9fe3-1491efef66be",
	}
	if status, _ := concrete.AuthorizeSessionResourceProtocol(ctx, "session", resource); status != 0 {
		t.Fatalf("active member OPTIONS status = %d", status)
	}
	repository.entityAllowed = false
	if status, body := concrete.AuthorizeSessionResourceProtocol(ctx, "session", resource); status != 403 || body["detail"] != "You do not have permission to perform this action." {
		t.Fatalf("nonmember history OPTIONS = %d %#v", status, body)
	}
	resource.Pattern = "/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/meta/"
	if status, _ := concrete.AuthorizeSessionResourceProtocol(ctx, "session", resource); status != 0 {
		t.Fatalf("meta OPTIONS unexpectedly applied method permission: %d", status)
	}
}

func TestWorkItemHistoryPermissionPrecedesTimezoneForActivityProtocol(t *testing.T) {
	repository := &workItemHistoryRepositoryStub{entityAllowed: false}
	service, err := NewWorkItemHistory(repository, workItemHistorySessionStub{principal: djangosession.Principal{
		UserID: "59961101-634c-4f9e-a436-5f8ec26e5f7b", UserTimezone: "Unknown/Timezone",
	}})
	if err != nil {
		t.Fatal(err)
	}
	concrete := service.(*workItemHistoryService)
	ctx, err := concrete.APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	resource := SessionProtocolResource{
		Method:    "OPTIONS",
		Pattern:   "/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/",
		Slug:      "sample",
		ProjectID: "1d989b61-12b9-4e3c-9fe3-1491efef66be",
	}
	if status, _ := concrete.AuthorizeSessionResourceProtocol(ctx, "session", resource); status != 403 {
		t.Fatalf("class permission should precede timezone validation, got %d", status)
	}
	resource.Pattern = "/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/meta/"
	if status, _ := concrete.AuthorizeSessionResourceProtocol(ctx, "session", resource); status != 400 {
		t.Fatalf("method timezone validation should precede GET decorator, got %d", status)
	}
}
