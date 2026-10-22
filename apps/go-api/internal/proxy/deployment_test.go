package proxy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestShippingTopologiesRunNativeBackendOnly(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"docker-compose.yml", "docker-compose-local.yml"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			var topology struct {
				Services map[string]struct {
					Image      string   `yaml:"image"`
					Command    []string `yaml:"command"`
					Entrypoint []string `yaml:"entrypoint"`
				} `yaml:"services"`
			}
			// Several unrelated services use string commands; inspect only the backend
			// through YAML nodes after resolving the real top-level service map.
			var document map[string]any
			if err := yaml.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			services, ok := document["services"].(map[string]any)
			if !ok {
				t.Fatal("services missing")
			}
			for _, retired := range []string{"api", "worker", "workspace-deletion-worker", "beat-worker", "migrator", "plane-mq", "plane-rustfs-init"} {
				if _, found := services[retired]; found {
					t.Errorf("retired service %s remains", retired)
				}
			}
			selected := map[string]any{"services": map[string]any{"go-api": services["go-api"], "go-migrator": services["go-migrator"]}}
			encoded, err := yaml.Marshal(selected)
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(encoded, &topology); err != nil {
				t.Fatal(err)
			}
			api := topology.Services["go-api"]
			bootstrap := topology.Services["go-migrator"]
			if len(api.Command) != 1 || api.Command[0] != "--go-background-tasks" {
				t.Errorf("native runtime not enabled: %v", api.Command)
			}
			if len(bootstrap.Entrypoint) != 1 || bootstrap.Entrypoint[0] != "/go-bootstrap" || len(bootstrap.Command) != 1 || bootstrap.Command[0] != "install" {
				t.Errorf("Go bootstrap missing: %#v", bootstrap)
			}
		})
	}
	for _, name := range []string{"apps/proxy/Caddyfile.ce", "apps/proxy/Caddyfile.local", "apps/proxy/Caddyfile.aio.ce"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, old := range []string{"api:8000", "localhost:3004", "DJANGO_API_UPSTREAM"} {
			if strings.Contains(string(data), old) {
				t.Errorf("%s retains Python upstream %s", name, old)
			}
		}
	}
}
