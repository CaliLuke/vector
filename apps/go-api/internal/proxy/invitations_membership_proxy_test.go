//go:build integration

package proxy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	invitationsMembershipOperationCount = 28
	invitationsMembershipPathCount      = 14
)

type invitationsMembershipContract struct {
	OperationCount int `json:"operation_count"`
	Operations     map[string]struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	} `json:"operations"`
}

func TestInvitationsMembershipProxyRoutesAllMethodsToGo(t *testing.T) {
	root := invitationsMembershipRepositoryRoot(t)
	contractPath := filepath.Join(root, "docs/go-backend-replacement/contracts/invitations_membership.json")
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read invitations_membership contract: %v", err)
	}
	var contract invitationsMembershipContract
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		t.Fatalf("decode invitations_membership contract: %v", err)
	}
	if contract.OperationCount != invitationsMembershipOperationCount || len(contract.Operations) != invitationsMembershipOperationCount {
		t.Fatalf("contract operations=%d/%d, want %d", contract.OperationCount, len(contract.Operations), invitationsMembershipOperationCount)
	}

	methodsByPath := make(map[string]map[string]bool)
	for _, operation := range contract.Operations {
		if methodsByPath[operation.Path] == nil {
			methodsByPath[operation.Path] = make(map[string]bool)
		}
		methodsByPath[operation.Path][operation.Method] = true
	}
	if len(methodsByPath) != invitationsMembershipPathCount {
		t.Fatalf("contract paths=%d, want %d", len(methodsByPath), invitationsMembershipPathCount)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "django")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)

	for template, authoredMethods := range methodsByPath {
		path := invitationsMembershipConcretePath(template)
		for _, form := range []struct {
			name string
			path string
		}{
			{name: "canonical", path: path},
			{name: "slashless", path: strings.TrimSuffix(path, "/")},
		} {
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodTrace} {
				label := "unsupported " + method
				if authoredMethods[method] {
					label = "authored " + method
				}
				t.Run(form.name+" "+label+" "+template, func(t *testing.T) {
					assertTarget(t, method, proxyURL+form.path, "go")
				})
			}
			t.Run(form.name+" HEAD "+template, func(t *testing.T) {
				assertGoProxyResponse(t, http.MethodHead, proxyURL+form.path)
			})
			t.Run(form.name+" OPTIONS "+template, func(t *testing.T) {
				assertTarget(t, http.MethodOptions, proxyURL+form.path, "go")
			})
			t.Run(form.name+" preflight "+template, func(t *testing.T) {
				assertGoPreflightWithoutAllow(t, proxyURL+form.path, map[string]string{
					"Origin":                        "http://localhost:3000",
					"Access-Control-Request-Method": http.MethodPost,
				})
			})
		}
	}

	t.Run("Go handles uppercase and compact UUID path forms", func(t *testing.T) {
		upperUUID := "00000000-0000-4000-8000-00000000ABCD"
		compactUUID := "00000000000040008000000000000001"
		paths := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/workspaces/invitation-proxy/projects/" + upperUUID + "/invitations/"},
			{http.MethodGet, "/api/workspaces/invitation-proxy/projects/" + compactUUID + "/invitations/"},
			{http.MethodGet, "/api/workspaces/invitation-proxy/invitations/" + upperUUID + "/"},
			{http.MethodGet, "/api/workspaces/invitation-proxy/invitations/" + compactUUID + "/"},
			{http.MethodGet, "/api/workspaces/invitation-proxy/projects/00000000-0000-4000-8000-000000000001/invitations/" + upperUUID + "/"},
			{http.MethodPost, "/api/workspaces/invitation-proxy/projects/" + upperUUID + "/join/00000000-0000-4000-8000-000000000002/"},
			{http.MethodPost, "/api/workspaces/invitation-proxy/projects/00000000-0000-4000-8000-000000000001/join/" + compactUUID + "/"},
			{http.MethodPatch, "/api/workspaces/invitation-proxy/members/" + upperUUID + "/"},
		}
		for _, test := range paths {
			assertTarget(t, test.method, proxyURL+test.path, "go")
		}
	})

	t.Run("Go handles arbitrary pk strings and dotted aliases", func(t *testing.T) {
		base := "/api/v1/workspaces/invitation-proxy/invitations/"
		for _, pk := range []string{
			"Invite-A1",
			"00000000-0000-4000-8000-00000000ABCD",
			"00000000000040008000000000000001",
		} {
			assertTarget(t, http.MethodGet, proxyURL+base+pk+"/", "go")
		}
		for _, path := range []string{
			base + "Invite.A1/",
			base + "00000000-0000-4000-8000-000000000001.json",
			base + "00000000-0000-4000-8000-000000000001.json/",
		} {
			assertTarget(t, http.MethodGet, proxyURL+path, "go")
		}
	})

	t.Run("retired multisegment neighbors also route to Go", func(t *testing.T) {
		project := "00000000-0000-4000-8000-000000000001"
		invite := "00000000-0000-4000-8000-000000000002"
		member := "00000000-0000-4000-8000-000000000003"
		neighbors := []struct {
			method string
			path   string
			want   string
		}{
			{http.MethodPost, "/api/workspaces/invitation-proxy/projects/" + project + "/project-views/", "go"},
			{http.MethodPost, "/api/workspaces/invitation-proxy/projects/" + project + "/project-deploy-boards/", "go"},
			{http.MethodPatch, "/api/workspaces/invitation-proxy/projects/" + project + "/project-deploy-boards/" + invite + "/", "go"},
			{http.MethodDelete, "/api/workspaces/invitation-proxy/projects/" + project + "/project-deploy-boards/" + invite + "/", "go"},
			{http.MethodPatch, "/api/workspaces/invitation-proxy/projects/" + project + "/preferences/member/" + member + "/", "go"},
			{http.MethodPost, "/api/workspaces/invitation-proxy/user-activity/" + member + "/export/", "go"},
			{http.MethodPost, "/api/workspaces/invitation-proxy/workspace-views/", "go"},
		}
		for _, neighbor := range neighbors {
			for _, path := range []string{neighbor.path, strings.TrimSuffix(neighbor.path, "/")} {
				assertTarget(t, neighbor.method, proxyURL+path, neighbor.want)
			}
		}
	})
}

func invitationsMembershipConcretePath(path string) string {
	path = strings.ReplaceAll(path, "{slug}", "invitation-proxy")
	path = strings.ReplaceAll(path, "{project_id}", "00000000-0000-4000-8000-000000000001")
	pk := "00000000-0000-4000-8000-000000000002"
	if strings.HasPrefix(path, "/api/v1/") {
		pk = "Invite-A1"
	}
	path = strings.ReplaceAll(path, "{pk}", pk)
	return path
}

func invitationsMembershipRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate invitations_membership proxy test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
}
