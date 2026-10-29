//go:build integration

package proxy_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.yaml.in/yaml/v3"
)

const (
	caddyImage       = "caddy:2.11.7-alpine"
	caddyHTTPPort    = "80/tcp"
	stickyCollection = "/api/workspaces/example/stickies/"
)

func TestInventoryRoutesRetiredOperationsToGo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	django := newNamedUpstream(t, "retired-python-must-not-be-used")
	goAPI := newNamedUpstream(t, "go")

	states := []struct {
		name    string
		enabled *bool
	}{
		{name: "default uses inventory"},
		{name: "legacy true flag cannot bypass inventory", enabled: boolPointer(true)},
		{name: "legacy false flag cannot disable inventory", enabled: boolPointer(false)},
	}
	for _, state := range states {
		t.Run(state.name, func(t *testing.T) {
			proxyURL := startCaddy(t, ctx, django.port, goAPI.port, state.enabled)
			detail := stickyCollection + "00000000-0000-4000-8000-000000000000"
			assertTarget(t, http.MethodGet, proxyURL+"/api/users/me/", "go")
			assertGoProxyResponse(t, http.MethodHead, proxyURL+"/api/users/me/")
			assertTarget(t, http.MethodPatch, proxyURL+"/api/users/me/", "go")
			assertTarget(t, http.MethodDelete, proxyURL+"/api/users/me/", "go")
			assertGoResponseWithAllow(t, http.MethodOptions, proxyURL+"/api/users/me/", "GET, PATCH, DELETE, HEAD, OPTIONS")
			assertGoResponseWithAllow(t, http.MethodPost, proxyURL+"/api/users/me/", "GET, PATCH, DELETE, HEAD, OPTIONS")
			assertGoResponseWithAllow(t, http.MethodTrace, proxyURL+"/api/users/me/", "GET, PATCH, DELETE, HEAD, OPTIONS")
			assertGoPreflightWithoutAllow(t, proxyURL+"/api/users/me/", map[string]string{
				"Origin":                        "http://localhost:3000",
				"Access-Control-Request-Method": http.MethodGet,
			})
			assertTarget(t, http.MethodGet, proxyURL+"/api/timezones/", "go")
			assertTarget(t, http.MethodGet, proxyURL+"/api/workspace-slug-check/", "go")
			assertTarget(t, http.MethodGet, proxyURL+"/api/instances/workspace-slug-check/", "go")
			assertTarget(t, http.MethodGet, proxyURL+"/api/users/me/instance-admin/", "go")
			assertTarget(t, http.MethodPatch, proxyURL+"/api/users/me/onboard/", "go")
			assertTarget(t, http.MethodPatch, proxyURL+"/api/users/me/tour-completed/", "go")
			assertTarget(t, http.MethodGet, proxyURL+"/api/users/me/notification-preferences/", "go")
			assertTarget(t, http.MethodPatch, proxyURL+"/api/users/me/notification-preferences/", "go")
			assertTarget(t, http.MethodGet, proxyURL+stickyCollection, "go")
			assertTarget(t, http.MethodPost, proxyURL+stickyCollection, "go")
			assertTarget(t, http.MethodGet, proxyURL+detail+"/", "go")
			assertTarget(t, http.MethodGet, proxyURL+detail, "go")
			assertTarget(t, http.MethodPatch, proxyURL+detail+"/", "go")
			assertTarget(t, http.MethodDelete, proxyURL+detail, "go")
			assertTarget(t, http.MethodOptions, proxyURL+"/api/timezones", "go")
			assertTarget(t, http.MethodOptions, proxyURL+"/api/timezones?format=json", "go")
			assertGoPreflightWithoutAllow(t, proxyURL+"/api/timezones", map[string]string{
				"Origin":                        "http://localhost:3000",
				"Access-Control-Request-Method": http.MethodGet,
			})
			assertGoPreflightWithoutAllow(t, proxyURL+"/api/timezones", map[string]string{
				"Access-Control-Request-Method": "",
			})
			assertTarget(t, http.MethodOptions, proxyURL+strings.TrimSuffix(stickyCollection, "/"), "go")
			assertGoPreflightWithoutAllow(t, proxyURL+strings.TrimSuffix(stickyCollection, "/"), map[string]string{
				"Origin":                        "http://localhost:3000",
				"Access-Control-Request-Method": http.MethodGet,
			})
			assertAPIV1StickyRouting(t, proxyURL)
			assertTarget(t, http.MethodGet, proxyURL+"/api/workspaces/example/projects/", "go")
			workspaceMemberID := "00000000-0000-4000-8000-000000000001"
			projectID := "00000000-0000-4000-8000-000000000002"
			assertProjectLifecycleRouting(t, proxyURL, projectID)
			assertTarget(t, http.MethodGet, proxyURL+"/api/workspaces/example/members/"+workspaceMemberID+"/", "go")
			assertTarget(t, http.MethodGet, proxyURL+"/api/workspaces/example/members/not-a-uuid/", "go")
			assertTarget(t, http.MethodGet, proxyURL+"/api/workspaces/example/projects/"+projectID+"/members/", "go")
			assertTarget(t, http.MethodGet, proxyURL+"/api/workspaces/example/projects/not-a-uuid/members/", "go")
			assertTarget(
				t,
				http.MethodGet,
				proxyURL+"/api/v1/workspaces/example/projects/not-a-uuid/issues/"+projectID+"/activities",
				"go",
			)
			assertOptionsTarget(
				t,
				proxyURL+"/api/workspaces/example/projects/"+projectID+"/members/",
				"go",
				"GET, POST, HEAD, OPTIONS",
			)
			assertOptionsTarget(
				t,
				proxyURL+"/api/workspaces/example/members/",
				"go",
				"",
			)
			assertGoResponseWithAllow(t, http.MethodPatch, proxyURL+"/api/workspaces/example/members/"+workspaceMemberID+"/", "GET, PATCH, DELETE, HEAD, OPTIONS")
			assertGoResponseWithAllow(t, http.MethodPost, proxyURL+"/api/workspaces/example/projects/"+projectID+"/members/", "GET, POST, HEAD, OPTIONS")
			assertGoResponseWithAllow(t, http.MethodPatch, proxyURL+"/api/workspaces/example/projects/"+projectID+"/members/"+workspaceMemberID+"/", "GET, PATCH, DELETE, HEAD, OPTIONS")
			assertGoResponseWithAllow(t, http.MethodPatch, proxyURL+"/api/workspaces/example/projects/"+projectID+"/preferences/member/"+workspaceMemberID+"/", "GET, PATCH, HEAD, OPTIONS")
			for _, path := range []string{
				"/api/workspaces/example/members",
				"/api/workspaces/example/project-members",
			} {
				assertTarget(t, http.MethodGet, proxyURL+path, "go")
				assertGoProxyResponse(t, http.MethodHead, proxyURL+path)
				assertTarget(t, http.MethodOptions, proxyURL+path, "go")
				assertTarget(t, http.MethodGet, proxyURL+path+"?search=Alex%20Smith", "go")
			}
		})
	}
}

func assertAPIV1StickyRouting(t *testing.T, proxyURL string) {
	t.Helper()
	collection := "/api/v1/workspaces/example/stickies/"
	detail := collection + "00000000-0000-4000-8000-000000000001/"
	for _, path := range []string{collection, detail} {
		base := strings.TrimSuffix(path, "/")
		for _, variant := range []string{path, base, base + ".json", base + ".json/", base + ".xml"} {
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodTrace} {
				// The complete route belongs to Go, including unsupported methods.
				assertTarget(t, method, proxyURL+variant, "go")
			}
			assertGoProxyResponse(t, http.MethodHead, proxyURL+variant)
		}
	}
	// Unknown API paths reach Go, which owns the application 404 response.
	for _, path := range []string{collection + "nested/extra/", collection + ".json", collection + "item.JSON", collection + "item.json.more", "/api/v1/workspaces/example/sticky/"} {
		assertTarget(t, http.MethodGet, proxyURL+path, "go")
	}
}

func assertProjectLifecycleRouting(t *testing.T, proxyURL, projectID string) {
	t.Helper()
	appCollection := "/api/workspaces/example/projects/"
	apiCollection := "/api/v1/workspaces/example/projects/"
	for _, route := range []struct {
		method, path string
	}{
		{http.MethodPost, appCollection},
		{http.MethodPatch, appCollection + projectID + "/"},
		{http.MethodPut, appCollection + projectID + "/"},
		{http.MethodDelete, appCollection + projectID + "/"},
		{http.MethodPost, appCollection + projectID + "/archive/"},
		{http.MethodDelete, appCollection + projectID + "/archive/"},
		{http.MethodPost, apiCollection},
		{http.MethodPatch, apiCollection + projectID + "/"},
		{http.MethodDelete, apiCollection + projectID + "/"},
		{http.MethodPost, apiCollection + projectID + "/archive/"},
		{http.MethodDelete, apiCollection + projectID + "/archive/"},
		{http.MethodGet, "/api/v1/workspaces/example/projects-lite/"},
		{http.MethodGet, apiCollection + projectID + "/summary/"},
	} {
		for _, path := range []string{route.path, strings.TrimSuffix(route.path, "/")} {
			assertTarget(t, route.method, proxyURL+path, "go")
		}
	}
	// API project reads and writes now share the Go backend.
	for _, path := range []string{apiCollection, apiCollection + projectID + "/"} {
		assertTarget(t, http.MethodGet, proxyURL+path, "go")
		assertTarget(t, http.MethodGet, proxyURL+strings.TrimSuffix(path, "/"), "go")
		assertGoProxyResponse(t, http.MethodHead, proxyURL+path)
	}
	assertTarget(t, http.MethodPatch, proxyURL+apiCollection+"not-a-uuid/", "go")
	assertTarget(t, http.MethodPost, proxyURL+appCollection+"not-a-uuid/archive/", "go")
}

func TestRootComposePackagesNativeGoServices(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatalf("decode root Compose topology: %v", err)
	}
	value := func(node *yaml.Node, key string) *yaml.Node {
		if node == nil || node.Kind != yaml.MappingNode {
			return nil
		}
		for index := 0; index+1 < len(node.Content); index += 2 {
			if node.Content[index].Value == key {
				return node.Content[index+1]
			}
		}
		return nil
	}
	scalar := func(node *yaml.Node) string {
		if node == nil || node.Kind != yaml.ScalarNode {
			return ""
		}
		return node.Value
	}
	list := func(node *yaml.Node) []string {
		if node == nil {
			return nil
		}
		if node.Kind == yaml.ScalarNode {
			return []string{node.Value}
		}
		if node.Kind != yaml.SequenceNode {
			return nil
		}
		values := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			values = append(values, scalar(item))
		}
		return values
	}
	hasDependency := func(node *yaml.Node, key string) bool {
		if node == nil {
			return false
		}
		if node.Kind == yaml.MappingNode {
			return value(node, key) != nil
		}
		for _, dependency := range list(node) {
			if dependency == key {
				return true
			}
		}
		return false
	}
	services := value(document.Content[0], "services")
	goAPI := value(services, "go-api")
	if goAPI == nil {
		t.Fatal("root Compose topology omits go-api")
	}
	build := value(goAPI, "build")
	if scalar(value(build, "context")) != "./apps/go-api" || scalar(value(build, "dockerfile")) != "Dockerfile" {
		t.Fatalf("go-api build context=%q Dockerfile=%q, want native apps/go-api Dockerfile", scalar(value(build, "context")), scalar(value(build, "dockerfile")))
	}
	if got := list(value(goAPI, "command")); len(got) != 1 || got[0] != "--go-background-tasks" {
		t.Fatalf("go-api command = %v, want native background task runtime", got)
	}
	dependencies := value(goAPI, "depends_on")
	for _, dependency := range []string{"go-migrator", "plane-db", "plane-redis"} {
		if !hasDependency(dependencies, dependency) {
			t.Errorf("go-api is missing %s dependency", dependency)
		}
	}
	migrator := value(services, "go-migrator")
	if migrator == nil || scalar(value(value(migrator, "build"), "context")) != "./apps/go-api" || scalar(value(value(migrator, "build"), "dockerfile")) != "Dockerfile" {
		t.Fatalf("go-migrator build context=%q Dockerfile=%q", scalar(value(value(migrator, "build"), "context")), scalar(value(value(migrator, "build"), "dockerfile")))
	}
	if got := list(value(migrator, "entrypoint")); len(got) != 1 || got[0] != "/go-bootstrap" {
		t.Fatalf("go-migrator entrypoint=%v", got)
	}
	if got := list(value(migrator, "command")); len(got) != 1 || got[0] != "install" {
		t.Fatalf("go-migrator command=%v", got)
	}
	proxy := value(services, "proxy")
	if proxy == nil {
		t.Fatal("root Compose topology omits proxy")
	}
	if !hasDependency(value(proxy, "depends_on"), "go-api") {
		t.Fatal("proxy does not depend on go-api")
	}
}

func TestGoAPIDockerfileBuildsNativeRuntimeBinaries(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "apps", "go-api", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	dockerfile := string(content)
	for _, required := range []string{
		"FROM golang:1.27.3-trixie AS build",
		"go build -trimpath -o /out/go-plane ./cmd/system",
		"go build -trimpath -o /out/go-bootstrap ./cmd/bootstrap",
		"go build -trimpath -o /out/go-background-worker ./cmd/background-worker",
		"go build -trimpath -o /out/go-support-migrate ./cmd/support-migrate",
		"go build -trimpath -o /out/river github.com/riverqueue/river/cmd/river",
		"FROM gcr.io/distroless/static-debian13:nonroot",
	} {
		if !strings.Contains(dockerfile, required) {
			t.Errorf("Go API Dockerfile omits %q", required)
		}
	}
	for _, forbidden := range []string{"python", "celery", "supervisor"} {
		if strings.Contains(strings.ToLower(dockerfile), forbidden) {
			t.Errorf("Go API Dockerfile contains retired runtime %q", forbidden)
		}
	}
}

func TestAIOInventoryRoutesRetiredActivityReadsToGo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	django := newNamedUpstream(t, "retired-python-must-not-be-used")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddyWithConfig(t, ctx, "../../../proxy/Caddyfile.aio.ce", django.port, goAPI.port, nil)
	projectID := "00000000-0000-4000-8000-000000000001"
	issueID := "00000000-0000-4000-8000-000000000002"
	activityID := "00000000-0000-4000-8000-000000000003"
	for _, path := range []string{
		"/api/users/me/activities/",
		"/api/v1/workspaces/example/projects/" + projectID + "/issues/" + issueID + "/activities/",
		"/api/v1/workspaces/example/projects/" + projectID + "/issues/" + issueID + "/activities/" + activityID + "/",
		"/api/v1/workspaces/example/projects/" + projectID + "/work-items/" + issueID + "/activities/",
		"/api/v1/workspaces/example/projects/" + projectID + "/work-items/" + issueID + "/activities/" + activityID + "/",
	} {
		for _, routedPath := range []string{path, strings.TrimSuffix(path, "/")} {
			assertTarget(t, http.MethodGet, proxyURL+routedPath, "go")
			assertTarget(t, http.MethodPost, proxyURL+routedPath, "go")
			assertGoPreflightWithoutAllow(t, proxyURL+routedPath, map[string]string{
				"Origin":                        "http://localhost:3000",
				"Access-Control-Request-Method": http.MethodGet,
			})
		}
	}
	assertTarget(t, http.MethodGet, proxyURL+"/api/not-migrated/", "go")
}

func TestAIORetiredSplitMembershipPathsRepairPartialAllow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	django := newNamedUpstream(t, "retired-python-must-not-be-used")
	goAPI := newMembershipAllowUpstream(t)
	proxyURL := startCaddyWithConfig(t, ctx, "../../../proxy/Caddyfile.aio.ce", django.port, goAPI.port, nil)

	for _, test := range []struct {
		name      string
		path      string
		wantAllow string
	}{
		{
			name:      "app split path",
			path:      "/api/workspaces/example/projects/00000000-0000-4000-8000-000000000001/cycles/00000000-0000-4000-8000-000000000002/cycle-issues/",
			wantAllow: "GET, POST, HEAD, OPTIONS",
		},
		{
			name:      "API v1 alias",
			path:      "/api/v1/workspaces/example/projects/00000000-0000-4000-8000-000000000001/cycles/00000000-0000-4000-8000-000000000002/cycle-issues/",
			wantAllow: "GET, POST",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertMembershipAllow(t, http.MethodPost, proxyURL+test.path, true, http.StatusMethodNotAllowed, test.wantAllow)
			assertMembershipAllow(t, http.MethodPost, proxyURL+test.path, false, http.StatusMethodNotAllowed, "")
			assertMembershipAllow(t, http.MethodOptions, proxyURL+test.path, false, http.StatusOK, "")
		})
	}
}

func newMembershipAllowUpstream(t *testing.T) namedUpstream {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost && request.Header.Get("X-Test-Emit-Allow") != "" {
			response.Header().Set("Allow", "POST, OPTIONS")
		}
		if request.Method == http.MethodPost {
			response.WriteHeader(http.StatusMethodNotAllowed)
		}
		_, _ = io.WriteString(response, "go")
	}))
	t.Cleanup(server.Close)
	port := server.Listener.Addr().(*net.TCPAddr).Port
	return namedUpstream{port: port}
}

func assertMembershipAllow(t *testing.T, method, url string, emitAllow bool, wantStatus int, wantAllow string) {
	t.Helper()
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if emitAllow {
		request.Header.Set("X-Test-Emit-Allow", "partial")
	}
	if method == http.MethodOptions {
		request.Header.Set("Origin", "http://localhost:3000")
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	}
	response, err := (&http.Client{Timeout: 5 * time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close HTTP response body: %v", closeErr)
		}
	}()
	if response.StatusCode != wantStatus || response.Header.Get("Allow") != wantAllow {
		t.Fatalf("%s %s status=%d Allow=%q, want status=%d Allow=%q", method, url, response.StatusCode, response.Header.Get("Allow"), wantStatus, wantAllow)
	}
}

func assertOptionsTarget(t *testing.T, url, wantBody, wantAllow string) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest(http.MethodOptions, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if wantAllow != "" {
		request.Header.Set("X-Test-Emit-Allow", "django-owned methods")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close HTTP response body: %v", closeErr)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != wantBody || (wantAllow != "" && response.Header.Get("Allow") != wantAllow) {
		t.Fatalf("OPTIONS %s body=%q Allow=%q, want body=%q Allow=%q", url, body, response.Header.Get("Allow"), wantBody, wantAllow)
	}
}

type namedUpstream struct {
	port int
}

func newNamedUpstream(t *testing.T, name string) namedUpstream {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Test-Upstream", name)
		if allow := request.Header.Get("X-Test-Emit-Allow"); allow != "" {
			response.Header().Set("Allow", allow)
		}
		_, _ = io.WriteString(response, name)
	}))
	t.Cleanup(server.Close)
	port := server.Listener.Addr().(*net.TCPAddr).Port
	return namedUpstream{port: port}
}

func startCaddy(t *testing.T, ctx context.Context, djangoPort, goPort int, enabled *bool) string {
	t.Helper()
	return startCaddyWithConfig(t, ctx, "../../../proxy/Caddyfile.ce", djangoPort, goPort, enabled)
}

func startCaddyWithConfig(t *testing.T, ctx context.Context, configPath string, djangoPort, goPort int, enabled *bool) string {
	t.Helper()
	caddyfile, err := filepath.Abs(configPath)
	if err != nil {
		t.Fatalf("resolve Caddyfile path: %v", err)
	}
	routePolicy, err := filepath.Abs("../../../proxy/go-active-routes.caddy")
	if err != nil {
		t.Fatalf("resolve route policy path: %v", err)
	}
	environment := map[string]string{
		"SITE_ADDRESS":        ":80",
		"FILE_SIZE_LIMIT":     "5242880",
		"BUCKET_NAME":         "uploads",
		"DJANGO_API_UPSTREAM": hostUpstream(djangoPort),
		"GO_API_UPSTREAM":     hostUpstream(goPort),
	}
	if enabled != nil {
		environment["GO_STICKIES_ENABLED"] = strconv.FormatBool(*enabled)
	}

	options := []testcontainers.ContainerCustomizer{
		testcontainers.WithHostPortAccess(djangoPort, goPort),
		testcontainers.WithExposedPorts(caddyHTTPPort),
		testcontainers.WithEnv(environment),
		testcontainers.WithFiles(
			testcontainers.ContainerFile{
				HostFilePath:      caddyfile,
				ContainerFilePath: "/etc/caddy/Caddyfile",
				FileMode:          0o644,
			},
			testcontainers.ContainerFile{
				HostFilePath:      routePolicy,
				ContainerFilePath: "/etc/caddy/go-active-routes.caddy",
				FileMode:          0o644,
			},
		),
		testcontainers.WithWaitStrategy(
			wait.ForLog("serving initial configuration").WithStartupTimeout(time.Minute),
		),
	}
	container, err := testcontainers.Run(ctx, caddyImage, options...)
	if err != nil {
		t.Fatalf("start Caddy with Sticky flag %v: %v", enabled, err)
	}
	t.Cleanup(func() {
		if terminateContainerErr := testcontainers.TerminateContainer(container); terminateContainerErr != nil {
			t.Errorf("terminate Caddy testcontainer: %v", terminateContainerErr)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get Caddy host: %v", err)
	}
	port, err := container.MappedPort(ctx, caddyHTTPPort)
	if err != nil {
		t.Fatalf("get Caddy port: %v", err)
	}
	return "http://" + net.JoinHostPort(host, port.Port())
}

func hostUpstream(port int) string {
	return net.JoinHostPort(testcontainers.HostInternal, strconv.Itoa(port))
}

func assertTarget(t *testing.T, method, url, want string) {
	t.Helper()
	response := proxyRequestWithRetry(t, method, url, nil)
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close HTTP response body: %v", closeErr)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, url, err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%s %s status = %d, want %d", method, url, response.StatusCode, http.StatusOK)
	}
	if got := string(body); got != want {
		t.Fatalf("%s %s target = %q, want %q", method, url, got, want)
	}
}

func assertGoProxyResponse(t *testing.T, method, url string) {
	t.Helper()
	response := proxyRequestWithRetry(t, method, url, nil)
	t.Cleanup(func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close proxy response: %v", closeErr)
		}
	})
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, url, err)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("X-Test-Upstream") != "go" || (method != http.MethodHead && string(body) != "go") {
		t.Fatalf("%s %s status=%d upstream=%q body=%q, want Go upstream", method, url, response.StatusCode, response.Header.Get("X-Test-Upstream"), body)
	}
}

func assertGoResponseWithAllow(t *testing.T, method, url, wantAllow string) {
	const wantTarget = "go"
	t.Helper()
	response := proxyRequestWithRetry(t, method, url, map[string]string{
		"X-Test-Emit-Allow": "django-owned methods",
	})
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close HTTP response body: %v", closeErr)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, url, err)
	}
	if response.StatusCode != http.StatusOK || string(body) != wantTarget || response.Header.Get("Allow") != wantAllow {
		t.Fatalf(
			"%s %s = status %d target %q Allow %q, want status 200 target %q Allow %q",
			method,
			url,
			response.StatusCode,
			body,
			response.Header.Get("Allow"),
			wantTarget,
			wantAllow,
		)
	}
}

func assertGoPreflightWithoutAllow(t *testing.T, url string, headers map[string]string) {
	const method, wantTarget = http.MethodOptions, "go"
	t.Helper()
	response := proxyRequestWithRetry(t, method, url, headers)
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("close HTTP response body: %v", closeErr)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, url, err)
	}
	if response.StatusCode != http.StatusOK || string(body) != wantTarget || response.Header.Get("Allow") != "" {
		t.Fatalf("%s %s = status %d target %q Allow %q, want status 200 target %q without Allow", method, url, response.StatusCode, body, response.Header.Get("Allow"), wantTarget)
	}
}

func proxyRequestWithRetry(t *testing.T, method, url string, headers map[string]string) *http.Response {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for {
		request, err := http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("build %s %s: %v", method, url, err)
		}
		for name, value := range headers {
			request.Header.Set(name, value)
		}
		response, err := client.Do(request)
		if err == nil {
			return response
		}
		lastErr = err
		if time.Now().After(deadline) {
			t.Fatalf("%s %s after retries: %v", method, url, lastErr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
func boolPointer(value bool) *bool {
	return &value
}

func TestAPIModuleRoutingIncludesLifecycleWriters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "retired-python-must-not-be-used")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)
	moduleAPIBase := "/api/v1/workspaces/example/projects/00000000-0000-4000-8000-000000000002/"
	detail := moduleAPIBase + "modules/00000000-0000-4000-8000-000000000003/"
	for _, path := range []string{moduleAPIBase + "archived-modules/", moduleAPIBase + "modules-lite/", moduleAPIBase + "modules/", detail, detail + "module-issues/"} {
		assertTarget(t, http.MethodGet, proxyURL+path, "go")
		assertGoProxyResponse(t, http.MethodHead, proxyURL+path)
		assertTarget(t, http.MethodOptions, proxyURL+path, "go")
	}
	assertGoResponseWithAllow(t, http.MethodPost, proxyURL+moduleAPIBase+"modules/", "GET, POST")
	assertGoResponseWithAllow(t, http.MethodPatch, proxyURL+detail, "GET, PATCH, DELETE")
	assertGoResponseWithAllow(t, http.MethodDelete, proxyURL+detail, "GET, PATCH, DELETE")
	assertGoResponseWithAllow(t, http.MethodPost, proxyURL+detail+"module-issues/", "GET, POST")
	assertTarget(t, http.MethodDelete, proxyURL+detail+"module-issues/00000000-0000-4000-8000-000000000004/", "go")
	assertTarget(t, http.MethodPost, proxyURL+detail+"archive/", "go")
	assertTarget(t, http.MethodDelete, proxyURL+moduleAPIBase+"archived-modules/00000000-0000-4000-8000-000000000003/unarchive/", "go")
	assertTarget(t, http.MethodGet, proxyURL+moduleAPIBase+"modules/not-a-uuid/", "go")
}

func TestAppModuleLifecycleRoutingOwnsWritesAndProtocolCompanions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "retired-python-must-not-be-used")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)
	base := "/api/workspaces/example/projects/00000000-0000-4000-8000-000000000002/"
	module := "00000000-0000-4000-8000-000000000003/"
	for _, route := range []struct {
		path    string
		methods []string
	}{
		{"modules/", []string{http.MethodPost}},
		{"modules/" + module, []string{http.MethodPut, http.MethodPatch, http.MethodDelete}},
		{"modules/" + module + "module-links/", []string{http.MethodPost}},
		{"modules/" + module + "module-links/00000000-0000-4000-8000-000000000004/", []string{http.MethodPut, http.MethodPatch, http.MethodDelete}},
		{"modules/" + module + "archive/", []string{http.MethodPost, http.MethodDelete}},
		{"archived-modules/", []string{http.MethodPost, http.MethodDelete}},
		{"archived-modules/" + module, []string{http.MethodPost, http.MethodDelete}},
	} {
		path := base + route.path
		for _, form := range []string{path, strings.TrimSuffix(path, "/")} {
			for _, method := range append(route.methods, http.MethodGet, http.MethodOptions, http.MethodTrace) {
				assertTarget(t, method, proxyURL+form, "go")
			}
			assertGoProxyResponse(t, http.MethodHead, proxyURL+form)
		}
	}
	// Neighboring module paths are not covered by a broad prefix matcher.
	assertTarget(t, http.MethodPost, proxyURL+base+"modules/"+module+"unregistered-action/", "go")
}

func TestAPILabelRoutingRetiresWriters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "retired-python-must-not-be-used")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)
	labelBase := "/api/v1/workspaces/example/projects/00000000-0000-4000-8000-000000000002/labels/"
	labelDetail := labelBase + "00000000-0000-4000-8000-000000000003/"
	for _, path := range []string{labelBase, labelDetail} {
		assertTarget(t, http.MethodGet, proxyURL+path, "go")
		assertGoProxyResponse(t, http.MethodHead, proxyURL+path)
		assertTarget(t, http.MethodOptions, proxyURL+path, "go")
	}
	assertGoResponseWithAllow(t, http.MethodPost, proxyURL+labelBase, "GET, POST")
	assertGoResponseWithAllow(t, http.MethodPatch, proxyURL+labelDetail, "GET, PATCH, DELETE")
	assertGoResponseWithAllow(t, http.MethodDelete, proxyURL+labelDetail, "GET, PATCH, DELETE")
}
