package main

import "testing"

func TestWorkspaceLifecycleRetainsPerHandlerAllow(t *testing.T) {
	for _, test := range []struct{ path, method, want string }{
		{"/api/workspaces/", "POST", "POST, OPTIONS"},
		{"/api/workspaces/demo/", "PUT", "PUT, PATCH, DELETE, OPTIONS"},
		{"/api/workspaces/demo/", "PATCH", "PUT, PATCH, DELETE, OPTIONS"},
		{"/api/workspaces/demo/", "DELETE", "PUT, PATCH, DELETE, OPTIONS"},
		{"/api/workspaces/", "GET", "GET, POST, HEAD, OPTIONS"},
		{"/api/workspaces/demo/", "GET", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"},
	} {
		protocol, ok := resolveReadServiceProtocolForMethod(test.path, test.method)
		if !ok || protocol.allow != test.want {
			t.Errorf("%s %s Allow=%q, resolved=%v, want=%q", test.method, test.path, protocol.allow, ok, test.want)
		}
	}
}
