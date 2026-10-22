package routing

import (
	"regexp"
	"strings"
	"testing"
)

func TestInvitationsCaddyPreservesSessionUUIDConverters(t *testing.T) {
	implementation := "invitations_membership"
	const id = "00000000-0000-0000-0000-000000018111"
	for _, path := range []string{
		"/api/workspaces/{slug}/invitations/{pk}/",
		"/api/workspaces/{slug}/invitations/{pk}/join/",
		"/api/workspaces/{slug}/members/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/invitations/",
		"/api/workspaces/{slug}/projects/{project_id}/invitations/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/join/{pk}/",
	} {
		t.Run(path, func(t *testing.T) {
			pattern := regexp.MustCompile(caddyPathPattern(Operation{Path: path, Implementation: &implementation}))
			fill := strings.NewReplacer("{slug}", "invitation-workspace", "{project_id}", id, "{pk}", id)
			if !pattern.MatchString(fill.Replace(path)) {
				t.Fatal("Django accepts lowercase canonical UUIDs without RFC variant restrictions")
			}
			for _, parameter := range []string{"project_id", "pk"} {
				if !strings.Contains(path, "{"+parameter+"}") {
					continue
				}
				for _, value := range []string{"not-a-uuid", "00000000000000000000000000018111", "00000000-0000-0000-0000-00000000ABCD"} {
					candidate := fill.Replace(strings.Replace(path, "{"+parameter+"}", value, 1))
					if pattern.MatchString(candidate) {
						t.Errorf("invalid Django UUID converter value %q must fall through to Django", candidate)
					}
				}
			}
		})
	}
}

func TestInvitationsCaddyKeepsAPIRouterLookupSeparateFromUUIDConverters(t *testing.T) {
	implementation := "invitations_membership"
	pattern := regexp.MustCompile(caddyPathPattern(Operation{
		Path: "/api/v1/workspaces/{slug}/invitations/{pk}/", Implementation: &implementation,
	}))
	for _, key := range []string{"not-a-uuid", "00000000000000000000000000018111", "00000000-0000-0000-0000-00000000ABCD"} {
		if !pattern.MatchString("/api/v1/workspaces/workspace/invitations/" + key + "/") {
			t.Errorf("DRF router admits %q for handler-level validation", key)
		}
	}
	if pattern.MatchString("/api/v1/workspaces/workspace/invitations/not.a.uuid/") {
		t.Fatal("DRF's ordinary lookup route excludes dots")
	}
}
