package routing

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNativeResourceMethodsAndGeneratedPolicy(t *testing.T) {
	policy, err := Load()
	require.NoError(t, err)
	var accountMethods []string
	for _, resource := range policy.Resources {
		if resource.Path == "/api/users/me/" {
			accountMethods = resource.Methods
		}
	}
	require.Equal(t, []string{"GET", "PATCH", "DELETE", "HEAD", "OPTIONS"}, accountMethods)
	generated, err := RenderCaddy(policy)
	require.NoError(t, err)
	actual, err := os.ReadFile("../../../proxy/api-allow.caddy")
	require.NoError(t, err)
	require.Equal(t, string(generated), string(actual), "regenerate the native resource headers")
	require.NotContains(t, string(generated), "reverse_proxy", "Caddyfiles own API forwarding")
}

func TestNativePolicyRejectsAmbiguousOrMalformedConfiguration(t *testing.T) {
	for _, data := range []string{
		`{`, `{"version":1,"unknown":true}`, `{"version":1,"resources":[]} true`, `{"version":1,"resources":[]} !`,
		`{"version":2,"resources":[]}`, `{"version":1,"resources":[]}`,
		`{"version":1,"resources":[{"path":"/api/a/","methods":[]}]}`,
		`{"version":1,"resources":[{"path":"/api/a/","methods":["GET","GET"]}]}`,
		`{"version":1,"resources":[{"path":"/api/a/","methods":["INVALID"]}]}`,
		`{"version":1,"resources":[{"path":"/app/a/","methods":["GET"]}]}`,
		`{"version":1,"resources":[{"path":"/api/a/?query","methods":["GET"]}]}`,
		`{"version":1,"resources":[{"path":"/api/{bad/","methods":["GET"]}]}`,
		`{"version":1,"resources":[{"path":"/api/{bad}/broken}/","methods":["GET"]}]}`,
		`{"version":1,"resources":[{"path":"/api/{id}/","methods":["GET"]},{"path":"/api/{pk}/","methods":["POST"]}]}`,
		`{"version":1,"resources":[{"path":"/api/{id}/","methods":["GET"],"uuid_parameters":["missing"]}]}`,
		`{"version":1,"resources":[{"path":"/api/{id}/","methods":["GET"],"uuid_parameters":["id","id"]}]}`,
	} {
		_, err := Parse([]byte(data))
		require.Error(t, err, data)
	}
	_, err := RenderCaddy(Policy{})
	require.Error(t, err)
}

func TestResourcePatternsAnchorParametersAndEscapeLiterals(t *testing.T) {
	pattern, err := resourcePattern(Resource{Path: "/api/workspaces/{slug}/projects/{id}/release.v1/"})
	require.NoError(t, err)
	matcher := regexp.MustCompile(pattern)
	require.True(t, matcher.MatchString("/api/workspaces/demo/projects/id/release.v1/"))
	for _, path := range []string{
		"/api/workspaces//projects/id/release.v1/", "/api/workspaces/demo/projects/id/releaseXv1/",
		"/prefix/api/workspaces/demo/projects/id/release.v1/", "/api/workspaces/demo/projects/id/release.v1/extra/",
	} {
		require.False(t, matcher.MatchString(path), path)
	}
	require.True(t, strings.HasPrefix(pattern, "^/api/"))
}

func TestResourceUUIDDoesNotCaptureLiteralActions(t *testing.T) {
	pattern, err := resourcePattern(Resource{Path: "/api/workspaces/{slug}/members/{id}/", UUIDParameters: []string{"id"}})
	require.NoError(t, err)
	matcher := regexp.MustCompile(pattern)
	require.True(t, matcher.MatchString("/api/workspaces/demo/members/00000000-0000-4000-8000-000000000001/"))
	require.False(t, matcher.MatchString("/api/workspaces/demo/members/leave/"))
}
