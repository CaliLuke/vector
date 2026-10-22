package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIV1AuditIncludesFinalNegotiationResponseExactlyOnce(t *testing.T) {
	path := "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"
	for _, test := range []struct {
		name, method, query, accept string
		status, authCalls           int
	}{
		{"unsupported accept", "GET", "", "text/plain", 406, 0},
		{"unsupported accept HEAD", "HEAD", "", "text/plain", 406, 0},
		{"unknown format", "GET", "?format=xml", "", 404, 0},
		{"unknown format HEAD", "HEAD", "?format=xml", "", 404, 0},
		{"successful response", "GET", "", "", 200, 1},
		{"indented response", "GET", "", "application/json; indent=2", 200, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &activityHTTPService{}
			logger := &apiV1ActivityLoggerStub{}
			handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit})
			request := httptest.NewRequest(test.method, path+test.query, nil)
			request.Header.Set("X-Api-Key", "token")
			if test.accept != "" {
				request.Header.Set("Accept", test.accept)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if test.status == 404 || test.status == 406 {
				if response.Header().Get("Vary") != "Cookie, origin" {
					t.Fatalf("negotiation Vary=%q, want Cookie, origin", response.Header().Get("Vary"))
				}
			}

			if response.Code != test.status || service.authCalls != test.authCalls || len(logger.records) != 1 {
				t.Fatalf("status=%d auth=%d audit=%#v", response.Code, service.authCalls, logger.records)
			}
			record := logger.records[0]
			if record.ResponseCode != test.status || record.ResponseBody == nil {
				t.Fatalf("audit=%#v", record)
			}
			if test.method == "HEAD" {
				if response.Body.Len() != 0 {
					t.Fatal("HEAD leaked body")
				}
				expected := string(readServicesNotAcceptableBody)
				if test.status == 404 {
					expected = string(readServicesFormatNotFoundBody)
				}
				if record.ResponseBody != expected {
					t.Fatalf("HEAD audit body=%#v, want %q", record.ResponseBody, expected)
				}
			} else if record.ResponseBody != response.Body.String() {
				t.Fatalf("audit body=%#v, response=%q", record.ResponseBody, response.Body.String())
			}
			if strings.Contains(test.accept, "indent") && !strings.Contains(response.Body.String(), "\n  ") {
				t.Fatalf("response not indented: %q", response.Body.String())
			}
		})
	}
}

func TestAPIV1AuditKeepsPreflightAndRequestSizeOutsideAuthentication(t *testing.T) {
	path := "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"
	for _, preflight := range []bool{false, true} {
		service := &activityHTTPService{}
		logger := &apiV1ActivityLoggerStub{}
		handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{activityLogger: logger, activitySecret: "secret", bodySizeLimit: 1})
		method := "GET"
		status := http.StatusRequestEntityTooLarge
		if preflight {
			method = "OPTIONS"
			status = http.StatusOK
		}
		request := httptest.NewRequest(method, path, strings.NewReader("oversized"))
		request.Header.Set("X-Api-Key", "token")
		request.Header.Set("Accept", "text/plain")
		if preflight {
			request.Header.Set("Origin", "https://app.example.com")
			request.Header.Set("Access-Control-Request-Method", "GET")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != status || service.authCalls != 0 || len(logger.records) != 0 {
			t.Fatalf("preflight=%v status=%d auth=%d audits=%d", preflight, response.Code, service.authCalls, len(logger.records))
		}
	}
}

func TestAPIV1NegotiationWithoutKeyPreservesCookieVary(t *testing.T) {
	service := &activityHTTPService{}
	logger := &apiV1ActivityLoggerStub{}
	handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit})
	request := httptest.NewRequest("GET", "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/", nil)
	request.Header.Set("Accept", "text/plain")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 406 || response.Header().Get("Vary") != "Cookie, origin" || service.authCalls != 0 || len(logger.records) != 0 {
		t.Fatalf("status=%d Vary=%q auth=%d audit=%d", response.Code, response.Header().Get("Vary"), service.authCalls, len(logger.records))
	}
}
