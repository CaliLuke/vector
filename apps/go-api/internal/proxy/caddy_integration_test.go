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

	"github.com/CaliLuke/go-plane/internal/testsupport"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.yaml.in/yaml/v3"
)

const (
	caddyImage    = "caddy:2.11.7-alpine"
	caddyHTTPPort = "80/tcp"
)

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
		switch request.Header.Get("X-Test-Emit-Status") {
		case "401":
			response.WriteHeader(http.StatusUnauthorized)
		case "403":
			response.WriteHeader(http.StatusForbidden)
		}
		testsupport.Write(t, response, []byte(name))
	}))
	t.Cleanup(server.Close)
	port := server.Listener.Addr().(*net.TCPAddr).Port
	return namedUpstream{port: port}
}

func startCaddyWithConfig(t *testing.T, ctx context.Context, configPath string, goPort int) string {
	t.Helper()
	caddyfile, err := filepath.Abs(configPath)
	if err != nil {
		t.Fatalf("resolve Caddyfile path: %v", err)
	}
	routePolicy, err := filepath.Abs("../../../proxy/api-allow.caddy")
	if err != nil {
		t.Fatalf("resolve route policy path: %v", err)
	}
	environment := map[string]string{
		"SITE_ADDRESS":    ":80",
		"FILE_SIZE_LIMIT": "5242880",
		"BUCKET_NAME":     "uploads",

		"GO_API_UPSTREAM": hostUpstream(goPort),
	}

	options := []testcontainers.ContainerCustomizer{
		testcontainers.WithHostPortAccess(goPort),
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
				ContainerFilePath: "/etc/caddy/api-allow.caddy",
				FileMode:          0o644,
			},
		),
		testcontainers.WithWaitStrategy(
			wait.ForLog("serving initial configuration").WithStartupTimeout(time.Minute),
		),
	}
	container, err := testcontainers.Run(ctx, caddyImage, options...)
	if err != nil {
		t.Fatalf("start Caddy: %v", err)
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
		"X-Test-Emit-Allow": "partial methods",
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
