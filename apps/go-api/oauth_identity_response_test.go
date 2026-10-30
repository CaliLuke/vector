package systemapi

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type oauthResponseBody struct {
	io.Reader
	closeErr error
	closes   int
}

func (b *oauthResponseBody) Close() error {
	b.closes++
	return b.closeErr
}

func TestOAuthIdentityRejectsResponseCleanupFailures(t *testing.T) {
	for _, source := range []struct{ provider, path string }{
		{"google", "/oauth2/v2/userinfo"},
		{"gitea", "/api/v1/user/emails"},
		{"github", "/orgs/example/memberships/fixture"},
	} {
		for _, status := range []int{http.StatusOK, http.StatusUnauthorized} {
			t.Run(source.provider+"/"+http.StatusText(status), func(t *testing.T) {
				closeErr := errors.New("provider connection cleanup failed")
				body := &oauthResponseBody{Reader: strings.NewReader(`[{"email":"fixture@example.test","primary":true,"verified":true}]`), closeErr: closeErr}
				if source.provider == "google" {
					body.Reader = strings.NewReader(`{"id":"42","email":"fixture@example.test","verified_email":true}`)
				}
				service := &oauthRedirectService{
					repository: &oauthRedirectTestRepository{config: map[string]string{"GITEA_HOST": "https://gitea.example", "GITHUB_ORGANIZATION_ID": "example"}},
					client: &http.Client{Transport: externalIntegrationsRoundTripper(func(request *http.Request) (*http.Response, error) {
						response := oauthIdentityResponse(request, http.StatusOK)
						if request.URL.Path == source.path {
							response.StatusCode = status
							response.Body = body
						}
						return response, nil
					})},
				}
				identity, err := service.providerIdentity(t.Context(), source.provider, &oauth2.Token{AccessToken: "fixture"}, oauthConfig{Config: &oauth2.Config{}})
				require.ErrorIs(t, err, closeErr)
				require.Empty(t, identity)
				require.Equal(t, 1, body.closes)
				if status == http.StatusUnauthorized && source.provider == "github" {
					var denied *oauthFailure
					require.ErrorAs(t, err, &denied)
					require.Equal(t, "GITHUB_USER_NOT_IN_ORG", denied.name)
				}
			})
		}
	}
}

func TestOAuthJSONRetainsReadAndCleanupFailures(t *testing.T) {
	for _, list := range []bool{false, true} {
		t.Run(map[bool]string{false: "object", true: "list"}[list], func(t *testing.T) {
			readErr := errors.New("provider body interrupted")
			closeErr := errors.New("provider connection cleanup failed")
			body := &oauthResponseBody{Reader: iotest.ErrReader(readErr), closeErr: closeErr}
			client := &http.Client{Transport: externalIntegrationsRoundTripper(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header), Request: request}, nil
			})}
			var err error
			if list {
				var values []map[string]any
				err = oauthGetJSONList(t.Context(), client, "https://provider.example/emails", &values)
			} else {
				_, err = oauthGetJSON(t.Context(), client, "https://provider.example/user")
			}
			require.ErrorIs(t, err, readErr)
			require.ErrorIs(t, err, closeErr)
			require.Equal(t, 1, body.closes)
		})
	}
}

func oauthIdentityResponse(request *http.Request, status int) *http.Response {
	body := `{"id":42,"login":"fixture","email":"fixture@example.test","confirmed_at":"2026-10-30"}`
	if strings.HasSuffix(request.URL.Path, "/emails") {
		body = `[{"email":"fixture@example.test","primary":true,"verified":true}]`
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}
}
