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
		assertGoProxyResponse(t, http.MethodHead, proxyURL+path)
	}
	assertGoResponseWithAllow(t, http.MethodOptions, proxyURL+list, "GET, POST")
	assertGoResponseWithAllow(t, http.MethodOptions, proxyURL+detail, "GET, PATCH, DELETE")
	assertGoPreflightWithoutAllow(t, proxyURL+list, map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": http.MethodGet,
	})
	assertGoResponseWithAllow(t, http.MethodPost, proxyURL+list, "GET, POST")
	assertGoResponseWithAllow(t, http.MethodPatch, proxyURL+detail, "GET, PATCH, DELETE")
	assertGoResponseWithAllow(t, http.MethodDelete, proxyURL+detail, "GET, PATCH, DELETE")
}
