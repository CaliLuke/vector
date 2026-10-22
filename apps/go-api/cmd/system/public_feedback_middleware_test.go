package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	publicfeedback "github.com/CaliLuke/go-plane/gen/public_feedback"
	system "github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	metadata "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

type publicFeedbackMiddlewareMetadata struct{}

func (publicFeedbackMiddlewareMetadata) ResolvePublicBoard(context.Context, string) (metadata.PublicBoard, error) {
	return metadata.PublicBoard{
		WorkspaceSlug: "fixture",
		WorkspaceID:   uuid.MustParse("00000000-0000-4000-8000-000000000004"),
		ProjectID:     uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		VotesEnabled:  true,
	}, nil
}

func (publicFeedbackMiddlewareMetadata) IssueDetail(context.Context, metadata.PublicBoard, string) (json.RawMessage, error) {
	return json.RawMessage(`null`), nil
}

func (publicFeedbackMiddlewareMetadata) ProjectBoards(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage(`[]`), nil
}

type publicFeedbackMiddlewareItems struct {
	voteListCalls     int
	voteCreateCalls   int
	voteDeleteErr     error
	intakeDetailCalls int
	taskCount         int
}

type publicFeedbackMiddlewareAudit struct {
	records []metadata.APIActivityLog
}

func (audit *publicFeedbackMiddlewareAudit) Log(_ context.Context, record metadata.APIActivityLog) error {
	audit.records = append(audit.records, record)
	return nil
}

func (items *publicFeedbackMiddlewareItems) PublicVoteList(context.Context, workitems.PublicFeedbackScope, string) (workitems.Result, error) {
	items.voteListCalls++
	return workitems.Result{Status: http.StatusOK, Body: []any{}}, nil
}

func (*publicFeedbackMiddlewareItems) PublicIssueList(context.Context, workitems.PublicFeedbackScope, string, map[string]string, map[string]bool) (workitems.Result, error) {
	return workitems.Result{Status: http.StatusOK, Body: []any{}}, nil
}

func (items *publicFeedbackMiddlewareItems) PublicVoteCreate(context.Context, workitems.PublicFeedbackScope, string, string, io.Reader, string) (workitems.Result, error) {
	items.voteCreateCalls++
	items.taskCount++
	return workitems.Result{Status: http.StatusCreated, Body: map[string]any{"vote": 1}}, nil
}

func (items *publicFeedbackMiddlewareItems) PublicVoteDelete(context.Context, workitems.PublicFeedbackScope, string, string) (workitems.Result, error) {
	if items.voteDeleteErr != nil {
		return workitems.Result{}, items.voteDeleteErr
	}
	return workitems.Result{Status: http.StatusNoContent}, nil
}

func (*publicFeedbackMiddlewareItems) PublicIntakeList(context.Context, workitems.PublicFeedbackScope, string, string, map[string]string, map[string]bool) (workitems.Result, error) {
	return workitems.Result{Status: http.StatusOK, Body: []any{}}, nil
}

func (*publicFeedbackMiddlewareItems) PublicIntakeCreate(context.Context, workitems.PublicFeedbackScope, string, string, io.Reader, string) (workitems.Result, error) {
	return workitems.Result{Status: http.StatusOK, Body: map[string]any{}}, nil
}

func (items *publicFeedbackMiddlewareItems) PublicIntakeDetail(context.Context, workitems.PublicFeedbackScope, string, string) (workitems.Result, error) {
	items.intakeDetailCalls++
	return workitems.Result{Status: http.StatusOK, Body: map[string]any{}}, nil
}

func (*publicFeedbackMiddlewareItems) PublicIntakeUpdate(context.Context, workitems.PublicFeedbackScope, string, string, string, io.Reader, string) (workitems.Result, error) {
	return workitems.Result{Status: http.StatusOK, Body: map[string]any{}}, nil
}

func (*publicFeedbackMiddlewareItems) PublicIntakeDelete(context.Context, workitems.PublicFeedbackScope, string, string, string) (workitems.Result, error) {
	return workitems.Result{Status: http.StatusNoContent}, nil
}

func publicFeedbackProductionHandler(session *publicProjectMetadataThrottleStub, items *publicFeedbackMiddlewareItems) http.Handler {
	return publicFeedbackProductionHandlerWith(session, items, nil, 1<<20)
}

func publicFeedbackProductionHandlerWith(session *publicProjectMetadataThrottleStub, items *publicFeedbackMiddlewareItems, audit metadata.ActivityLogger, bodyLimit int64) http.Handler {
	service, err := systemapi.NewPublicFeedback(publicFeedbackMiddlewareMetadata{}, items)
	if err != nil {
		panic(err)
	}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false),
		system.NewEndpoints(systemapi.NewSystem()), false, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicFeedback:                   publicfeedback.NewEndpoints(service),
			publicProjectMetadataThrottle:    session,
			publicProjectMetadataCORS:        []string{"https://app.example"},
			publicProjectMetadataCookie:      djangoSessionCookieConfig{Name: "session-id", Configured: true},
			publicProjectMetadataAdminCookie: djangoSessionCookieConfig{Name: "admin-session-id", Configured: true},
			publicProjectMetadataActivity:    audit,
			publicProjectMetadataSecret:      "public-feedback-test-secret",
			publicProjectMetadataBodyLimit:   bodyLimit,
			publicProjectMetadataBodySet:     true,
		},
	)
}

func TestPublicFeedbackMiddlewareRequestSizeAuthorizationDispatchAndAudit(t *testing.T) {
	t.Run("request-size rejects before authentication", func(t *testing.T) {
		session := &publicProjectMetadataThrottleStub{allowed: true}
		items := &publicFeedbackMiddlewareItems{}
		audit := &publicFeedbackMiddlewareAudit{}
		handler := publicFeedbackProductionHandlerWith(session, items, audit, 1)
		request := httptest.NewRequest(http.MethodPost, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusRequestEntityTooLarge || session.calls != 0 || items.voteCreateCalls != 0 || items.taskCount != 0 || len(audit.records) != 0 {
			t.Fatalf("request-size status=%d session calls=%d dispatch=%d tasks=%d audit=%d body=%s", response.Code, session.calls, items.voteCreateCalls, items.taskCount, len(audit.records), response.Body.String())
		}
	})

	t.Run("authorization rejects before dispatch", func(t *testing.T) {
		session := &publicProjectMetadataThrottleStub{allowed: true}
		items := &publicFeedbackMiddlewareItems{}
		audit := &publicFeedbackMiddlewareAudit{}
		handler := publicFeedbackProductionHandlerWith(session, items, audit, 1<<20)
		request := httptest.NewRequest(http.MethodPost, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || session.calls != 1 || items.voteCreateCalls != 0 || items.taskCount != 0 || len(audit.records) != 0 {
			t.Fatalf("authorization status=%d session calls=%d dispatch=%d tasks=%d audit=%d body=%s", response.Code, session.calls, items.voteCreateCalls, items.taskCount, len(audit.records), response.Body.String())
		}
	})

	t.Run("audit records an authorized request after dispatch", func(t *testing.T) {
		session := &publicProjectMetadataThrottleStub{allowed: true}
		items := &publicFeedbackMiddlewareItems{}
		audit := &publicFeedbackMiddlewareAudit{}
		handler := publicFeedbackProductionHandlerWith(session, items, audit, 1<<20)
		request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", nil)
		request.Header.Set("X-Api-Key", "audit-only")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || session.calls != 1 || len(audit.records) != 1 || audit.records[0].ResponseCode != http.StatusOK || audit.records[0].Method != http.MethodGet {
			t.Fatalf("audit status=%d session calls=%d dispatch=%d records=%#v", response.Code, session.calls, 0, audit.records)
		}
	})
}

func TestPublicFeedbackProductionMiddlewareOrdersSessionTimezoneAndEffects(t *testing.T) {
	items := &publicFeedbackMiddlewareItems{}
	session := &publicProjectMetadataThrottleStub{allowed: true}
	handler := publicFeedbackProductionHandler(session, items)

	optionalRead := httptest.NewRecorder()
	handler.ServeHTTP(optionalRead, httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", nil))
	if optionalRead.Code != http.StatusOK || session.calls != 1 {
		t.Fatalf("anonymous public read status=%d session calls=%d body=%s", optionalRead.Code, session.calls, optionalRead.Body.String())
	}

	anonymousVoteList := httptest.NewRecorder()
	handler.ServeHTTP(anonymousVoteList, httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", nil))
	if anonymousVoteList.Code != http.StatusUnauthorized || session.calls != 2 || items.voteListCalls != 0 {
		t.Fatalf("anonymous vote list status=%d session calls=%d vote list calls=%d body=%s", anonymousVoteList.Code, session.calls, items.voteListCalls, anonymousVoteList.Body.String())
	}

	optionalOptions := httptest.NewRecorder()
	handler.ServeHTTP(optionalOptions, httptest.NewRequest(http.MethodOptions, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", nil))
	if optionalOptions.Code != http.StatusOK || session.calls != 3 {
		t.Fatalf("optional OPTIONS status=%d session calls=%d body=%s", optionalOptions.Code, session.calls, optionalOptions.Body.String())
	}

	anonymousVoteOptions := httptest.NewRecorder()
	handler.ServeHTTP(anonymousVoteOptions, httptest.NewRequest(http.MethodOptions, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", nil))
	if anonymousVoteOptions.Code != http.StatusUnauthorized || session.calls != 4 {
		t.Fatalf("anonymous vote OPTIONS status=%d session calls=%d body=%s", anonymousVoteOptions.Code, session.calls, anonymousVoteOptions.Body.String())
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", nil)
	preflight.Header.Set("Origin", "https://app.example")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusOK || session.calls != 4 {
		t.Fatalf("vote CORS preflight status=%d session calls=%d body=%s", preflightResponse.Code, session.calls, preflightResponse.Body.String())
	}

	anonymousWrite := httptest.NewRequest(http.MethodPost, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", strings.NewReader(`{}`))
	anonymousWrite.Header.Set("Content-Type", "application/json")
	anonymousResponse := httptest.NewRecorder()
	handler.ServeHTTP(anonymousResponse, anonymousWrite)
	if anonymousResponse.Code != http.StatusUnauthorized || session.calls != 5 || items.voteCreateCalls != 0 || items.taskCount != 0 {
		t.Fatalf("anonymous vote status=%d session calls=%d vote calls=%d task count=%d body=%s", anonymousResponse.Code, session.calls, items.voteCreateCalls, items.taskCount, anonymousResponse.Body.String())
	}

	session.userID = "00000000-0000-4000-8000-000000000002"
	authenticatedWrite := httptest.NewRequest(http.MethodPost, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", strings.NewReader(`{}`))
	authenticatedWrite.Header.Set("Content-Type", "application/json")
	authenticatedWrite.AddCookie(&http.Cookie{Name: "session-id", Value: "valid"})
	authenticatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(authenticatedResponse, authenticatedWrite)
	if authenticatedResponse.Code != http.StatusCreated || session.calls != 6 || items.voteCreateCalls != 1 || items.taskCount != 1 {
		t.Fatalf("authenticated vote status=%d session calls=%d vote calls=%d task count=%d body=%s", authenticatedResponse.Code, session.calls, items.voteCreateCalls, items.taskCount, authenticatedResponse.Body.String())
	}

	timezoneSession := &publicProjectMetadataThrottleStub{allowed: true, timezoneErr: djangosession.ErrTimezoneNotFound}
	timezoneItems := &publicFeedbackMiddlewareItems{}
	timezoneHandler := publicFeedbackProductionHandler(timezoneSession, timezoneItems)
	timezoneResponse := httptest.NewRecorder()
	timezoneHandler.ServeHTTP(timezoneResponse, httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", nil))
	if timezoneResponse.Code != http.StatusBadRequest || timezoneResponse.Body.String() != `{"error":"The required key does not exist."}` || timezoneSession.calls != 1 || timezoneItems.taskCount != 0 {
		t.Fatalf("timezone failure status=%d session calls=%d task count=%d body=%s", timezoneResponse.Code, timezoneSession.calls, timezoneItems.taskCount, timezoneResponse.Body.String())
	}
}

func TestPublicFeedbackRequiredSessionPrecedesAnonymousThrottle(t *testing.T) {
	for _, test := range []struct {
		name        string
		allowed     bool
		throttleErr error
		wantStatus  int
	}{
		{name: "cold throttle", allowed: true, wantStatus: http.StatusUnauthorized},
		{name: "exhausted throttle", allowed: false, wantStatus: http.StatusUnauthorized},
		{name: "failing throttle", throttleErr: errors.New("Redis unavailable"), wantStatus: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := &publicProjectMetadataThrottleStub{allowed: test.allowed, throttleErr: test.throttleErr}
			items := &publicFeedbackMiddlewareItems{}
			handler := publicFeedbackProductionHandler(session, items)
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || session.allowCalls != 0 || session.timezoneCalls != 1 || items.voteListCalls != 0 {
				t.Fatalf("required route status=%d allow calls=%d timezone calls=%d vote list calls=%d body=%s", response.Code, session.allowCalls, session.timezoneCalls, items.voteListCalls, response.Body.String())
			}
		})
	}
}

func TestPublicFeedbackRequiredSessionErrorKeepsNegotiatedIndent(t *testing.T) {
	handler := publicFeedbackProductionHandler(&publicProjectMetadataThrottleStub{allowed: true}, &publicFeedbackMiddlewareItems{})
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", nil)
	request.Header.Set("Accept", "application/json; indent=2")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "{\n  \"detail\"") {
		t.Fatalf("required-session response=%d body=%q, want indented 401", response.Code, response.Body.String())
	}
}

func TestPublicFeedbackRouteClassificationIgnoresAnchorSpelling(t *testing.T) {
	for _, anchor := range []string{"intakes", "intake-issues"} {
		t.Run(anchor, func(t *testing.T) {
			session := &publicProjectMetadataThrottleStub{allowed: true}
			items := &publicFeedbackMiddlewareItems{}
			handler := publicFeedbackProductionHandler(session, items)
			path := "/api/public/anchor/" + anchor + "/issues/00000000-0000-4000-8000-000000000001/"

			get := httptest.NewRecorder()
			handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, path, nil))
			if get.Code != http.StatusOK || get.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
				t.Fatalf("GET status=%d Allow=%q body=%s", get.Code, get.Header().Get("Allow"), get.Body.String())
			}

			post := httptest.NewRecorder()
			handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, path, nil))
			if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
				t.Fatalf("POST status=%d Allow=%q body=%s", post.Code, post.Header().Get("Allow"), post.Body.String())
			}
		})
	}
}

func TestPublicFeedbackMalformedUUIDPathsResolveToNotFound(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "ordinary OPTIONS", method: http.MethodOptions, path: "/api/public/anchor/a/issues/not-a-uuid/"},
		{name: "uppercase UUID", method: http.MethodGet, path: "/api/public/anchor/a/issues/00000000-0000-4000-8000-00000000000A/"},
		{name: "slashless malformed UUID", method: http.MethodGet, path: "/api/public/anchor/a/issues/not-a-uuid"},
		{name: "intake row malformed UUID", method: http.MethodGet, path: "/api/public/anchor/a/intakes/00000000-0000-4000-8000-000000000001/intake-issues/not-a-uuid/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := &publicProjectMetadataThrottleStub{allowed: true}
			items := &publicFeedbackMiddlewareItems{}
			handler := publicFeedbackProductionHandler(session, items)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != http.StatusNotFound || response.Header().Get("Location") != "" || items.voteListCalls+items.intakeDetailCalls != 0 {
				t.Fatalf("status=%d location=%q items=%#v body=%s", response.Code, response.Header().Get("Location"), items, response.Body.String())
			}
		})
	}
}

func TestPublicFeedbackDerivedHeadUsesDesignedGetAndSuppressesBody(t *testing.T) {
	items := &publicFeedbackMiddlewareItems{}
	session := &publicProjectMetadataThrottleStub{allowed: true}
	handler := publicFeedbackProductionHandler(session, items)
	request := httptest.NewRequest(http.MethodHead, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.Len() != 0 {
		t.Fatalf("HEAD status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestPublicFeedbackProductionOptionsUsesSourceMetadata(t *testing.T) {
	cases := []struct {
		name         string
		path         string
		viewName     string
		requiresUser bool
		postAction   bool
	}{
		{name: "issue detail", path: "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", viewName: "Issue Retrieve Public Endpoint"},
		{name: "votes", path: "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", viewName: "Issue Vote Public", requiresUser: true, postAction: true},
		{name: "inbox collection", path: "/api/public/anchor/anchor/intakes/00000000-0000-4000-8000-000000000002/inbox-issues/", viewName: "Intake Issue Public", requiresUser: true, postAction: true},
		{name: "intake collection", path: "/api/public/anchor/anchor/intakes/00000000-0000-4000-8000-000000000002/intake-issues/", viewName: "Intake Issue Public", requiresUser: true, postAction: true},
		{name: "intake detail", path: "/api/public/anchor/anchor/intakes/00000000-0000-4000-8000-000000000002/intake-issues/00000000-0000-4000-8000-000000000003/", viewName: "Intake Issue Public", requiresUser: true},
		{name: "workspace boards", path: "/api/public/workspaces/fixture/project-boards/", viewName: "Workspace Project Deploy Board Endpoint"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			session := &publicProjectMetadataThrottleStub{allowed: true, userTimezone: "UTC"}
			if test.requiresUser {
				session.userID = "00000000-0000-4000-8000-000000000002"
			}
			handler := publicFeedbackProductionHandler(session, &publicFeedbackMiddlewareItems{})
			request := httptest.NewRequest(http.MethodOptions, test.path, nil)
			if test.requiresUser {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || body["name"] != test.viewName {
				t.Fatalf("status=%d name=%v body=%s", response.Code, body["name"], response.Body.String())
			}
			actions, hasActions := body["actions"].(map[string]any)
			if test.postAction != hasActions || test.postAction && actions[http.MethodPost] == nil {
				t.Fatalf("actions=%v body=%s", body["actions"], response.Body.String())
			}
		})
	}
}

func TestPublicFeedbackRequiredOptionsRunsTimezoneBeforeAuthAndDispatch(t *testing.T) {
	cases := []struct {
		name            string
		path            string
		allow           string
		requiresSession bool
	}{
		{name: "issue detail", path: "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", allow: "GET, HEAD, OPTIONS"},
		{name: "votes", path: "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", allow: "GET, POST, DELETE, HEAD, OPTIONS", requiresSession: true},
		{name: "inbox collection", path: "/api/public/anchor/anchor/intakes/00000000-0000-4000-8000-000000000002/inbox-issues/", allow: "GET, POST, HEAD, OPTIONS", requiresSession: true},
		{name: "intake collection", path: "/api/public/anchor/anchor/intakes/00000000-0000-4000-8000-000000000002/intake-issues/", allow: "GET, POST, HEAD, OPTIONS", requiresSession: true},
		{name: "intake detail", path: "/api/public/anchor/anchor/intakes/00000000-0000-4000-8000-000000000002/intake-issues/00000000-0000-4000-8000-000000000003/", allow: "GET, PATCH, DELETE, HEAD, OPTIONS", requiresSession: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			session := &publicProjectMetadataThrottleStub{
				allowed:     true,
				userID:      "00000000-0000-4000-8000-000000000002",
				timezoneErr: djangosession.ErrTimezoneNotFound,
			}
			items := &publicFeedbackMiddlewareItems{}
			handler := publicFeedbackProductionHandler(session, items)
			request := httptest.NewRequest(http.MethodOptions, test.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			wantTimezoneCalls, wantThrottleCalls := 0, 1
			if test.requiresSession {
				wantTimezoneCalls, wantThrottleCalls = 1, 0
			}
			if response.Code != http.StatusBadRequest || response.Body.String() != `{"error":"The required key does not exist."}` || response.Header().Get("Allow") != test.allow || session.timezoneCalls != wantTimezoneCalls || session.allowCalls != wantThrottleCalls || items.voteListCalls != 0 || items.voteCreateCalls != 0 || items.taskCount != 0 {
				t.Fatalf("required OPTIONS status=%d allow=%q timezone calls=%d throttle calls=%d issue list=%d vote list=%d vote creates=%d tasks=%d body=%s", response.Code, response.Header().Get("Allow"), session.timezoneCalls, session.allowCalls, 0, items.voteListCalls, items.voteCreateCalls, items.taskCount, response.Body.String())
			}
		})
	}
}

func TestPublicFeedbackMissingIssueWritesEmptyBodyWithoutContentType(t *testing.T) {
	handler := publicFeedbackProductionHandler(&publicProjectMetadataThrottleStub{allowed: true}, &publicFeedbackMiddlewareItems{})
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Content-Type") != "" {
		t.Fatalf("missing issue status=%d body=%q content-type=%q", response.Code, response.Body.String(), response.Header().Get("Content-Type"))
	}
}

func TestPublicFeedbackProductionRouteCompatibilityRedirectsSlashlessAndPreservesPreflight(t *testing.T) {
	items := &publicFeedbackMiddlewareItems{}
	session := &publicProjectMetadataThrottleStub{allowed: true}
	handler := publicFeedbackProductionHandler(session, items)

	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001", nil)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/" ||
		response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("slashless GET status=%d location=%q content-type=%q body=%q", response.Code, response.Header().Get("Location"), response.Header().Get("Content-Type"), response.Body.String())
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001", nil)
	preflight.Header.Set("Origin", "https://app.example")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", "accept")
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusOK || preflightResponse.Body.Len() != 0 ||
		preflightResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("slashless preflight status=%d allow-origin=%q body=%q", preflightResponse.Code, preflightResponse.Header().Get("Access-Control-Allow-Origin"), preflightResponse.Body.String())
	}
}

func TestPublicFeedbackProductionRouteCompatibilityReturnsSourceMethodErrors(t *testing.T) {
	t.Run("optional route returns DRF 405 headers and body", func(t *testing.T) {
		session := &publicProjectMetadataThrottleStub{allowed: true}
		items := &publicFeedbackMiddlewareItems{}
		handler := publicFeedbackProductionHandler(session, items)
		request := httptest.NewRequest(http.MethodPost, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/", nil)
		request.Header.Set("Accept", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, HEAD, OPTIONS" ||
			response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Vary") != "Cookie, origin" ||
			response.Body.String() != `{"detail":"Method \"POST\" not allowed."}` {
			t.Fatalf("status=%d allow=%q content-type=%q vary=%q body=%s", response.Code, response.Header().Get("Allow"), response.Header().Get("Content-Type"), response.Header().Get("Vary"), response.Body.String())
		}
	})

	t.Run("required route checks authentication before 405", func(t *testing.T) {
		session := &publicProjectMetadataThrottleStub{allowed: true}
		items := &publicFeedbackMiddlewareItems{}
		handler := publicFeedbackProductionHandler(session, items)
		request := httptest.NewRequest(http.MethodPut, "/api/public/anchor/anchor/issues/00000000-0000-4000-8000-000000000001/votes/", nil)
		request.Header.Set("Accept", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || response.Header().Get("Allow") != "GET, POST, DELETE, HEAD, OPTIONS" ||
			response.Body.String() != `{"detail":"Authentication credentials were not provided."}` || items.voteListCalls != 0 || items.voteCreateCalls != 0 || items.taskCount != 0 {
			t.Fatalf("status=%d allow=%q vote lists=%d vote creates=%d tasks=%d body=%s", response.Code, response.Header().Get("Allow"), items.voteListCalls, items.voteCreateCalls, items.taskCount, response.Body.String())
		}
	})
}
