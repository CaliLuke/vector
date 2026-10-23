package devtools

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStackComposeContract(t *testing.T) {
	root := findRepositoryRoot(t)
	services := composeServices(t, root, nil)
	proxy := mapValue(services["proxy"])
	ports := arrayOfMaps(t, proxy["ports"])
	if len(ports) == 0 || stringValue(ports[0]["published"]) != "8000" || stringValue(ports[0]["host_ip"]) != "127.0.0.1" {
		t.Fatalf("browser proxy must publish localhost:8000 only: %#v", proxy["ports"])
	}
	if _, ok := mapValue(proxy["depends_on"])["go-api"]; !ok {
		t.Fatal("proxy must depend on the Go API")
	}
	for _, retired := range []string{"api", "worker", "workspace-deletion-worker", "beat-worker", "migrator", "plane-mq"} {
		if _, ok := services[retired]; ok {
			t.Errorf("retired Django service %q is still in the local stack", retired)
		}
	}
	goAPI := mapValue(services["go-api"])
	if !strings.Contains(strings.Join(stringSlice(goAPI["command"]), " "), "--go-background-tasks") {
		t.Error("local Go API must start background tasks")
	}
	goAPIEnv := mapValue(goAPI["environment"])
	for _, key := range []string{"APP_BASE_URL", "ADMIN_BASE_URL", "SPACE_BASE_URL"} {
		if stringValue(goAPIEnv[key]) != localOrigin {
			t.Errorf("go-api %s = %q, want %q", key, goAPIEnv[key], localOrigin)
		}
	}
	if !strings.Contains(stringValue(goAPIEnv["CORS_ALLOWED_ORIGINS"]), localOrigin) {
		t.Errorf("CORS_ALLOWED_ORIGINS does not include proxy origin: %v", goAPIEnv["CORS_ALLOWED_ORIGINS"])
	}
	goDepends := mapValue(goAPI["depends_on"])
	if stringValue(mapValue(goDepends["go-migrator"])["condition"]) != "service_completed_successfully" {
		t.Error("Go API must wait for native bootstrap to finish")
	}
	if stringValue(mapValue(goDepends["logal"])["condition"]) != "service_healthy" {
		t.Error("Go API must wait for Logal health")
	}
	if stringValue(goAPIEnv["OTEL_EXPORTER_OTLP_ENDPOINT"]) != "http://logal:4318" {
		t.Errorf("unexpected OTLP endpoint: %v", goAPIEnv["OTEL_EXPORTER_OTLP_ENDPOINT"])
	}
	logal := mapValue(services["logal"])
	if stringValue(mapValue(logal["environment"])["LOGAL_RETENTION_HOURS"]) != "48" {
		t.Error("Logal retention must remain 48 hours")
	}
	if !containsVolumeMount(logal["volumes"], "/data", "logaldata", "volume") {
		t.Fatal("Logal must persist its data in the named volume")
	}
	for name, service := range services {
		options := mapValue(mapValue(mapValue(service)["logging"])["options"])
		if stringValue(options["max-size"]) != "10m" || stringValue(options["max-file"]) != "3" {
			t.Errorf("service %s has unexpected log rotation: %#v", name, options)
		}
	}
}

func TestSelectedLocalPortsReachComposeAndCaddy(t *testing.T) {
	root := findRepositoryRoot(t)
	services := composeServices(t, root, map[string]string{"PLANE_ADMIN_PORT": "3004", "PLANE_SPACE_PORT": "3005"})
	proxyEnv := mapValue(mapValue(services["proxy"])["environment"])
	if stringValue(proxyEnv["PLANE_ADMIN_PORT"]) != "3004" || stringValue(proxyEnv["PLANE_SPACE_PORT"]) != "3005" {
		t.Fatalf("selected admin/space ports did not reach proxy: %#v", proxyEnv)
	}
	cors := strings.Split(stringValue(mapValue(mapValue(services["go-api"])["environment"])["CORS_ALLOWED_ORIGINS"]), ",")
	if !contains(cors, "http://localhost:3004") || !contains(cors, "http://localhost:3005") {
		t.Fatalf("selected upstream origins missing from CORS: %v", cors)
	}
	caddy, err := os.ReadFile(filepath.Join(root, "apps/proxy/Caddyfile.local"))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"host.docker.internal:{$PLANE_ADMIN_PORT:3001}", "host.docker.internal:{$PLANE_SPACE_PORT:3002}"} {
		if !strings.Contains(string(caddy), value) {
			t.Errorf("Caddyfile is missing upstream port binding %q", value)
		}
	}
}

func TestLocalDevScriptUsesGoLogForwarderAndGeneratedEnvironment(t *testing.T) {
	root := findRepositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts/dev.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(content)
	for _, expected := range []string{"scripts/dev-env.sh", "go -C apps/go-api run ./cmd/devtools start", "apps/go-api/.env", "VITE_LOCAL_LOGGING=1"} {
		if !strings.Contains(script, expected) {
			t.Errorf("scripts/dev.sh is missing required local-stack behavior %q", expected)
		}
	}
}

func TestLocalDevBuildCompletesBeforeWatchersStart(t *testing.T) {
	repository := findRepositoryRoot(t)
	for _, scenario := range []struct {
		name      string
		buildExit string
		want      string
	}{
		{name: "successful build", buildExit: "0", want: "build\nwatchers\n"},
		{name: "failed build prevents watchers", buildExit: "7", want: "build\n"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			for _, directory := range []string{"scripts/quality", "bin", "apps/go-api", "apps/web", "apps/admin", "apps/space", "apps/live"} {
				if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"dev.sh", "dev-env.sh", "quality/run-bounded.mjs"} {
				content, err := os.ReadFile(filepath.Join(repository, "scripts", name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "scripts", name), content, 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, directory := range []string{"", "apps/go-api", "apps/web", "apps/admin", "apps/space", "apps/live"} {
				if err := os.WriteFile(filepath.Join(root, directory, ".env"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			bin := filepath.Join(root, "bin")
			writeExecutable(t, filepath.Join(bin, "docker"), "#!/bin/sh\nexit 0\n")
			writeExecutable(t, filepath.Join(bin, "pnpm"), "#!/bin/sh\nprintf 'build\\n' >> \"$DEV_TEST_EVENTS\"\nexit \"$DEV_TEST_BUILD_EXIT\"\n")
			writeExecutable(t, filepath.Join(bin, "go"), "#!/bin/sh\nprintf 'watchers\\n' >> \"$DEV_TEST_EVENTS\"\n")
			events := filepath.Join(root, "events")
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("DEV_TEST_EVENTS", events)
			t.Setenv("DEV_TEST_BUILD_EXIT", scenario.buildExit)
			command := exec.Command("bash", filepath.Join(root, "scripts/dev.sh"))
			output, err := command.CombinedOutput()
			if (err == nil) != (scenario.buildExit == "0") {
				t.Fatalf("startup error=%v output=%s", err, output)
			}
			if got := readFile(t, events); got != scenario.want {
				t.Fatalf("startup sequence=%q, want %q", got, scenario.want)
			}
		})
	}
}

func composeServices(t *testing.T, root string, env map[string]string) map[string]any {
	t.Helper()
	composeFile, err := os.ReadFile(filepath.Join(root, "docker-compose-local.yml"))
	if err != nil {
		t.Fatal(err)
	}
	// Compose requires these machine-specific env files to exist even for config inspection.
	// Point only the temporary config's env_file entries at /dev/null; topology and paths stay real.
	configText := strings.ReplaceAll(string(composeFile), "./apps/go-api/.env", "/dev/null")
	configText = strings.ReplaceAll(configText, "      - .env\n", "      - /dev/null\n")
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "docker-compose-local.yml")
	if err := os.WriteFile(configPath, []byte(configText), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("docker", "compose", "--project-directory", root, "-f", configPath, "config", "--format", "json")
	if env != nil {
		childEnv := make([]string, 0, len(os.Environ())+len(env))
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if _, overridden := env[key]; !overridden {
				childEnv = append(childEnv, entry)
			}
		}
		for key, value := range env {
			childEnv = append(childEnv, key+"="+value)
		}
		cmd.Env = childEnv
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if _, lookErr := exec.LookPath("docker"); lookErr != nil {
			t.Skip("docker is required to inspect the local compose topology")
		}
		t.Fatalf("docker compose config failed: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var config struct {
		Services map[string]any `json:"services"`
	}
	if err := json.Unmarshal(output, &config); err != nil {
		t.Fatalf("decode compose config: %v", err)
	}
	return config.Services
}

func findRepositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "docker-compose-local.yml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find repository root")
		}
		dir = parent
	}
}

func mapValue(value any) map[string]any {
	result, _ := value.(map[string]any)
	if result == nil {
		return map[string]any{}
	}
	return result
}

func arrayOfMaps(t *testing.T, value any) []map[string]any {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", value)
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, mapValue(item))
	}
	return result
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if values, ok := value.([]string); ok {
			return values
		}
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, stringValue(item))
	}
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsVolumeMount(value any, target, source, mountType string) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		mount := mapValue(item)
		if stringValue(mount["target"]) == target && stringValue(mount["source"]) == source && stringValue(mount["type"]) == mountType {
			return true
		}
	}
	return false
}
