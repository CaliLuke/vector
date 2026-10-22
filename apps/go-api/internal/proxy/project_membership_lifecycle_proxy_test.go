//go:build integration

package proxy_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestProjectMembershipLifecycleRoutesAllMethodsAndNeighborsToGo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "django")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)
	id := "00000000-0000-4000-8000-000000009601"
	roots := []string{"/api/v1/workspaces/membership/projects/" + id + "/members/", "/api/v1/workspaces/membership/projects/" + id + "/project-members/"}
	paths := []string{"/api/v1/workspaces/membership/projects/" + id + "/project-members-lite/", "/api/workspaces/membership/projects/" + id + "/members/", "/api/workspaces/membership/projects/" + id + "/members/" + id + "/", "/api/workspaces/membership/projects/" + id + "/members/leave/"}
	for _, root := range roots {
		paths = append(paths, root, root+id+"/")
	}
	for _, path := range paths {
		for _, form := range []string{path, strings.TrimSuffix(path, "/")} {
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodPut, http.MethodOptions} {
				assertTarget(t, method, proxyURL+form, "go")
			}
			assertHeadTarget(t, proxyURL+form, "go")
			assertTargetWithoutAllow(t, http.MethodOptions, proxyURL+form, "go", map[string]string{"Origin": "http://localhost:3000", "Access-Control-Request-Method": http.MethodPost})
		}
	}
	for _, path := range []string{"/api/v1/workspaces/membership/projects/not-a-uuid/members/", "/api/v1/workspaces/membership/projects/" + id + "/members/not-a-uuid/", "/api/workspaces/membership/projects/not-a-uuid/members/", "/api/workspaces/membership/projects/" + id + "/members/not-a-uuid/", "/api/workspaces/membership/projects/" + id + "/preferences/member/" + id + "/"} {
		assertTarget(t, http.MethodPost, proxyURL+path, "go")
	}
	// Workspace rosters now belong to invitations_membership rather than this family.
	for _, path := range []string{"/api/v1/workspaces/membership/members/", "/api/v1/workspaces/membership/members-lite/"} {
		assertTarget(t, http.MethodGet, proxyURL+path, "go")
		assertHeadTarget(t, proxyURL+path, "go")
	}
}
