package systemapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

type unavailableOAuthConfigRepository struct {
	oauthRedirectTestRepository
	failure error
}

func (r *unavailableOAuthConfigRepository) ConfigValue(context.Context, string, string) (string, error) {
	return "", r.failure
}

func TestOAuthIdentityRejectsUnavailableProviderConfiguration(t *testing.T) {
	for _, provider := range []string{"github", "gitlab", "gitea"} {
		t.Run(provider, func(t *testing.T) {
			unavailable := errors.New("configuration storage unavailable")
			requests := 0
			service := &oauthRedirectService{
				repository: &unavailableOAuthConfigRepository{failure: unavailable},
				client: &http.Client{Transport: externalIntegrationsRoundTripper(func(request *http.Request) (*http.Response, error) {
					requests++
					return oauthIdentityResponse(request, http.StatusOK), nil
				})},
			}
			identity, err := service.providerIdentity(t.Context(), provider, &oauth2.Token{AccessToken: "fixture"}, oauthConfig{Config: &oauth2.Config{}})
			if !errors.Is(err, unavailable) || identity != (oauthIdentity{}) {
				t.Errorf("configuration read failure returned identity=%+v error=%v", identity, err)
			}
			allowedRequests := 0
			if provider == "github" {
				allowedRequests = 1
			}
			if requests != allowedRequests {
				t.Errorf("provider requests=%d, want %d before configuration failure", requests, allowedRequests)
			}
		})
	}
}

func TestOAuthIdentityUsesCurrentProviderConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, provider, key, value string
		endpoints                  []string
		status                     int
	}{
		{"GitHub without organization restriction", "github", "GITHUB_ORGANIZATION_ID", "", []string{"https://api.github.com/user", "https://api.github.com/user/emails"}, http.StatusOK},
		{"GitHub organization member", "github", "GITHUB_ORGANIZATION_ID", "example", []string{"https://api.github.com/user", "https://api.github.com/orgs/example/memberships/fixture", "https://api.github.com/user/emails"}, http.StatusOK},
		{"GitHub rejects non-member", "github", "GITHUB_ORGANIZATION_ID", "example", []string{"https://api.github.com/user", "https://api.github.com/orgs/example/memberships/fixture"}, http.StatusNotFound},
		{"GitLab host", "gitlab", "GITLAB_HOST", "https://gitlab.example/", []string{"https://gitlab.example/api/v4/user"}, http.StatusOK},
		{"Gitea host", "gitea", "GITEA_HOST", "https://gitea.example/", []string{"https://gitea.example/api/v1/user", "https://gitea.example/api/v1/user/emails"}, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			service := &oauthRedirectService{
				repository: &oauthRedirectTestRepository{config: map[string]string{test.key: test.value}},
				client: &http.Client{Transport: externalIntegrationsRoundTripper(func(request *http.Request) (*http.Response, error) {
					if requests >= len(test.endpoints) {
						t.Fatalf("unexpected request %s", request.URL)
					}
					if got := request.URL.String(); got != test.endpoints[requests] {
						t.Errorf("request URL=%q, want %q", got, test.endpoints[requests])
					}
					requests++
					status := http.StatusOK
					if strings.Contains(request.URL.Path, "/memberships/") {
						status = test.status
					}
					return oauthIdentityResponse(request, status), nil
				})},
			}
			identity, err := service.providerIdentity(t.Context(), test.provider, &oauth2.Token{AccessToken: "fixture"}, oauthConfig{Config: &oauth2.Config{}})
			if test.status == http.StatusNotFound {
				var failure *oauthFailure
				if !errors.As(err, &failure) || failure.name != "GITHUB_USER_NOT_IN_ORG" || identity != (oauthIdentity{}) {
					t.Errorf("non-member identity=%+v error=%v", identity, err)
				}
			} else if err != nil || identity.ProviderID != "42" || identity.Email != "fixture@example.test" {
				t.Errorf("configured provider identity=%+v error=%v", identity, err)
			}
			if requests != len(test.endpoints) {
				t.Errorf("requests=%d, want %d", requests, len(test.endpoints))
			}
		})
	}
}
