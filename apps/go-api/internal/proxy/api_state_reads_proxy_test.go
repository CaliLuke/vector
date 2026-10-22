//go:build integration

package proxy_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestAPIStateRoutingIncludesRetiredWriters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "django")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)
	list := "/api/v1/workspaces/api-state-reads/projects/00000000-0000-4000-8000-000000009601/states/"
	detail := list + "00000000-0000-4000-8000-000000009701/"

	for _, path := range []string{list, detail} {
		assertTarget(t, http.MethodGet, proxyURL+path, "go")
		assertHeadTarget(t, proxyURL+path, "go")
	}
	assertTargetWithAllow(t, http.MethodOptions, proxyURL+list, "go", "GET, POST")
	assertTargetWithAllow(t, http.MethodOptions, proxyURL+detail, "go", "GET, PATCH, DELETE")
	assertTargetWithoutAllow(t, http.MethodOptions, proxyURL+list, "go", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": http.MethodGet,
	})
	assertTargetWithAllow(t, http.MethodPost, proxyURL+list, "go", "GET, POST")
	assertTargetWithAllow(t, http.MethodPatch, proxyURL+detail, "go", "GET, PATCH, DELETE")
	assertTargetWithAllow(t, http.MethodDelete, proxyURL+detail, "go", "GET, PATCH, DELETE")
}
