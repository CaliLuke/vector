package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
)

type accountAuthenticationOptionsAuthorizer struct{}

func (accountAuthenticationOptionsAuthorizer) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	return 0, nil
}

func (accountAuthenticationOptionsAuthorizer) AuthorizeSessionResourceProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any) {
	return 0, nil
}

func TestAccountAuthenticationMountedOptionsIgnoresInvalidSessionTimezone(t *testing.T) {
	repository := &accountAuthenticationMiddlewareRepository{}
	recorder := &cycleMiddlewareRecorder{}
	sessions := &accountAuthenticationMiddlewareSessions{
		cycleMiddlewareSessions: &cycleMiddlewareSessions{
			recorder:     recorder,
			invalidZone:  true,
			userTimezone: "Not/AZone",
		},
		userID: "user-1",
	}
	service, err := systemapi.NewAccountAuthentication(
		repository,
		accountAuthenticationHTTPThrottle{},
		accountAuthenticationHTTPCache{},
		accountAuthenticationHTTPTasks{},
		accountAuthenticationHTTPMagicIssuer{},
		accountAuthenticationHTTPResetIssuer{},
		password.Default(),
		&djangosession.Validator{},
		"https://app.example",
		"https://space.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{
		accountAuthentication:         api.NewEndpoints(service),
		accountAuthenticationMetadata: accountAuthenticationOptionsAuthorizer{},
		readServicesSession:           sessions,
		readServicesCookie:            djangoSessionCookieConfig{Name: "session-id", Configured: true},
		readServicesAuthSession:       sessions,
		readServicesAuthCookie:        djangoSessionCookieConfig{Name: "session-id", Configured: true},
	})

	protocols := accountAuthenticationProtocols()
	for _, protocol := range protocols {
		t.Run(protocol.pattern, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, protocol.pattern, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), protocol.name) {
				t.Fatalf("mounted OPTIONS status=%d body=%s; want 200 metadata for %q", response.Code, response.Body.String(), protocol.name)
			}
		})
	}
	if sessions.inspections != len(protocols) {
		t.Fatalf("session inspections=%d, want one per mounted OPTIONS request (%d)", sessions.inspections, len(protocols))
	}
}
