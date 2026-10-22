package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type identityProtocolStub struct{ calls int }

func (s *identityProtocolStub) AuthorizeAPIIdentityProtocol(context.Context, string) (int, map[string]any) {
	s.calls++
	return 0, nil
}
func TestAPIIdentityProtocolPolicyKeepsProjectAuthorizationSeparate(t *testing.T) {
	identity := &identityProtocolStub{}
	project := &activityHTTPService{}
	authorizer := activityReadsMethodAuthorizer{identity: identity, readOnly: project, primary: project}
	handler := activityReadsAPIMethodNotAllowedHandler(authorizer)
	for _, method := range []string{http.MethodHead, http.MethodOptions, http.MethodPost} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/api/v1/users/me/", nil))
		if response.Code != 405 || response.Header().Get("Allow") != "GET" {
			t.Fatalf("%s status=%d headers=%v", method, response.Code, response.Header())
		}
	}
	if identity.calls != 3 || project.protocolCalls != 0 {
		t.Fatalf("identity=%d project=%d", identity.calls, project.protocolCalls)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodHead, "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/cycles/", nil)
	handler.ServeHTTP(response, request)
	if project.protocolCalls != 1 || identity.calls != 3 {
		t.Fatalf("identity=%d project=%d", identity.calls, project.protocolCalls)
	}
}
