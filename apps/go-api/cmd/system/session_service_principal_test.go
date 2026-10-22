package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSessionServicePrincipalLifecycleSelectsDeferredInspectionByProtocol(t *testing.T) {
	original := readServiceProtocols
	deferred := newSessionServiceProtocol("/test/deferred-principal/", "POST, OPTIONS", "Deferred test", false)
	deferred.permissionBeforeTimezone = true
	ordinary := newSessionServiceProtocol("/test/ordinary-principal/", "POST, OPTIONS", "Ordinary test", false)
	readServiceProtocols = append(append([]readServiceProtocol(nil), original...), deferred, ordinary)
	t.Cleanup(func() { readServiceProtocols = original })

	for _, scenario := range []struct {
		name, path         string
		deferred, ordinary int
	}{
		{"permission first", "/test/deferred-principal/", 1, 0},
		{"ordinary", "/test/ordinary-principal/", 0, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{userID: "user-id", userTimezone: "Mars/Olympus"}
			lifecycle := sessionServicePrincipalLifecycle(sessions)
			request := httptest.NewRequest(http.MethodPost, scenario.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-key"})

			_, _, _, principal, err := lifecycle.InspectReadServiceSession(context.Background(), request)

			if err != nil || principal.UserID != "user-id" || principal.UserTimezone != "Mars/Olympus" {
				t.Fatalf("principal=%+v err=%v", principal, err)
			}
			if sessions.deferredTimezoneCalls != scenario.deferred || sessions.timezoneCalls != scenario.ordinary || sessions.calls != 1 || sessions.inspectedCookie != "session-key" {
				t.Fatalf("deferred=%d ordinary=%d calls=%d cookie=%q", sessions.deferredTimezoneCalls, sessions.timezoneCalls, sessions.calls, sessions.inspectedCookie)
			}
		})
	}
}

func TestSessionServicePrincipalLifecycleRequiresDeferredInspectorForPermissionFirst(t *testing.T) {
	original := readServiceProtocols
	protocol := newSessionServiceProtocol("/test/deferred-inspector-required/", "POST, OPTIONS", "Deferred test", false)
	protocol.permissionBeforeTimezone = true
	readServiceProtocols = append(append([]readServiceProtocol(nil), original...), protocol)
	t.Cleanup(func() { readServiceProtocols = original })

	sessions := &publicProjectMetadataThrottleStub{userID: "user-id", userTimezone: "Mars/Olympus"}
	// Hide the optional deferred method while retaining the lifecycle interface.
	lifecycle := sessionServicePrincipalLifecycle(struct{ readServicesSessionLifecycle }{sessions})
	request := httptest.NewRequest(http.MethodPost, "/test/deferred-inspector-required/", nil)

	_, _, _, principal, err := lifecycle.InspectReadServiceSession(context.Background(), request)

	if err == nil || !strings.Contains(err.Error(), "without timezone is unavailable") || principal.UserID != "" {
		t.Fatalf("principal=%+v err=%v", principal, err)
	}
	if sessions.calls != 0 || sessions.timezoneCalls != 0 {
		t.Fatalf("ordinary inspector unexpectedly called: calls=%d timezone=%d", sessions.calls, sessions.timezoneCalls)
	}
}

func TestSessionServicePrincipalLifecycleKeepsNilDisabled(t *testing.T) {
	if sessionServicePrincipalLifecycle(nil) != nil {
		t.Fatal("nil lifecycle should stay disabled")
	}
}

func TestPageLifecycleMethodMetadataKeepsRetiredDescriptionReadsOnNormalTimezonePath(t *testing.T) {
	sessions := &publicProjectMetadataThrottleStub{userID: "user-id", userTimezone: "Mars/Olympus"}
	lifecycle := sessionServicePrincipalLifecycle(sessions)
	for _, scenario := range []struct {
		method             string
		suffix             string
		deferred, ordinary int
	}{
		{http.MethodGet, "description/", 0, 1},
		{http.MethodHead, "description/", 0, 1},
		{http.MethodPatch, "description/", 1, 0},
		{http.MethodPut, "description/", 1, 0},
		{http.MethodTrace, "description/", 1, 0},
		{"PURGE", "description/", 1, 0},
		{http.MethodOptions, "description/", 1, 0},
		{http.MethodGet, "access/", 1, 0},
		{http.MethodHead, "lock/", 1, 0},
	} {
		t.Run(scenario.method+"/"+scenario.suffix, func(t *testing.T) {
			sessions.calls, sessions.timezoneCalls, sessions.deferredTimezoneCalls = 0, 0, 0
			request := httptest.NewRequest(scenario.method, "/api/workspaces/demo/projects/71000000-0000-4000-8000-000000000001/pages/71000000-0000-4000-8000-000000000002/"+scenario.suffix, nil)
			_, _, _, _, err := lifecycle.InspectReadServiceSession(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if sessions.deferredTimezoneCalls != scenario.deferred || sessions.timezoneCalls != scenario.ordinary {
				t.Fatalf("deferred=%d ordinary=%d, want %d/%d", sessions.deferredTimezoneCalls, sessions.timezoneCalls, scenario.deferred, scenario.ordinary)
			}
		})
	}
}
