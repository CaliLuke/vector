package main

import (
	"net/http/httptest"
	"testing"
)

func TestPublicFeedbackAllowMethodsMatchRouteTemplate(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/public/anchor/a/issues/", "GET, HEAD, OPTIONS"},
		{"/api/public/anchor/a/issues/i/", "GET, HEAD, OPTIONS"},
		{"/api/public/anchor/a/issues/00000000-0000-4000-8000-000000000001/votes/", "GET, POST, DELETE, HEAD, OPTIONS"},
		{"/api/public/anchor/a/intakes/00000000-0000-4000-8000-000000000001/inbox-issues/", "GET, POST, HEAD, OPTIONS"},
		{"/api/public/anchor/a/intakes/00000000-0000-4000-8000-000000000001/intake-issues/", "GET, POST, HEAD, OPTIONS"},
		{"/api/public/anchor/a/intakes/00000000-0000-4000-8000-000000000001/intake-issues/00000000-0000-4000-8000-000000000002/", "GET, PATCH, DELETE, HEAD, OPTIONS"},
		{"/api/public/workspaces/w/project-boards/", "GET, HEAD, OPTIONS"},
		{"/api/public/anchor/intakes/issues/not-a-uuid/votes/", "GET, POST, DELETE, HEAD, OPTIONS"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest("GET", test.path, nil)
			if got := publicFeedbackAllowMethods(request); got != test.want {
				t.Fatalf("Allow=%q, want %q", got, test.want)
			}
		})
	}
}

func TestClassifyPublicFeedbackPathUsesDjangoUUIDSpelling(t *testing.T) {
	valid := []string{
		"/api/public/anchor/a/issues/00000000-0000-4000-8000-000000000001/",
		"/api/public/anchor/a/issues/00000000-0000-4000-8000-000000000001/votes/",
		"/api/public/anchor/a/intakes/00000000-0000-4000-8000-000000000001/intake-issues/00000000-0000-4000-8000-000000000002/",
	}
	for _, path := range valid {
		route, matched := classifyPublicFeedbackPath(path)
		if !matched || route.malformedUUID {
			t.Errorf("classifyPublicFeedbackPath(%q) = (%#v, %v), want matched valid UUID route", path, route, matched)
		}
	}
	invalid := []string{
		"/api/public/anchor/a/issues/not-a-uuid/",
		"/api/public/anchor/a/issues/00000000-0000-4000-8000-00000000000A/",
		"/api/public/anchor/a/intakes/00000000-0000-4000-8000-000000000001/intake-issues/00000000-0000-4000-8000-00000000000A/",
	}
	for _, path := range invalid {
		route, matched := classifyPublicFeedbackPath(path)
		if !matched || !route.malformedUUID {
			t.Errorf("classifyPublicFeedbackPath(%q) = (%#v, %v), want matched malformed UUID route", path, route, matched)
		}
	}
}
