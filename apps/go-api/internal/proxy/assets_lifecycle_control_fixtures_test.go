package proxy_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var requiredAssetsLifecycleControlFixtureRefs = []string{
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/anonymous-get-06-rejected.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/source-native-method-not-allowed-legacy-attachment-detail.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/source-native-method-not-allowed-legacy-user-detail.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/source-native-method-not-allowed-legacy-workspace-detail.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/source-native-method-not-allowed-public-anchor-detail.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/source-native-method-not-allowed-v2-attachment-detail.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/source-native-method-not-allowed-v2-user-collection.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/source-native-method-not-allowed-v2-workspace-detail.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/duplicate-asset-throttle-unsupported-method-consumes.json",
	"apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/v2-user-asset-detail-get-not-implemented.json",
}

func validateAssetsLifecycleControlFixtureRefs(refs []string) error {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("control fixture ref is empty")
		}
		if !strings.HasPrefix(ref, "apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/") || filepath.Clean(ref) != ref {
			return fmt.Errorf("control fixture ref %q is outside the assets lifecycle fixture directory", ref)
		}
		if _, ok := seen[ref]; ok {
			return fmt.Errorf("duplicate control fixture ref %q", ref)
		}
		seen[ref] = struct{}{}
	}

	for _, required := range requiredAssetsLifecycleControlFixtureRefs {
		if _, ok := seen[required]; !ok {
			return fmt.Errorf("required control fixture ref %q is missing", required)
		}
	}
	return nil
}

func TestAssetsLifecycleControlFixtureRefs(t *testing.T) {
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "../../../.."))
	contractPath := filepath.Join(root, "docs/go-backend-replacement/contracts/assets_lifecycle.json")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		ControlFixtureRefs []string `json:"control_fixture_refs"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	if err := validateAssetsLifecycleControlFixtureRefs(contract.ControlFixtureRefs); err != nil {
		t.Fatalf("assets lifecycle contract control refs: %v", err)
	}

	for _, missing := range requiredAssetsLifecycleControlFixtureRefs {
		missing := missing
		t.Run("missing "+filepath.Base(missing), func(t *testing.T) {
			refs := make([]string, 0, len(contract.ControlFixtureRefs)-1)
			for _, ref := range contract.ControlFixtureRefs {
				if ref != missing {
					refs = append(refs, ref)
				}
			}
			if err := validateAssetsLifecycleControlFixtureRefs(refs); err == nil {
				t.Fatalf("validator accepted a contract missing required control fixture %q", missing)
			}
		})
	}
	t.Run("empty ref", func(t *testing.T) {
		refs := append([]string(nil), contract.ControlFixtureRefs...)
		refs = append(refs, " ")
		if err := validateAssetsLifecycleControlFixtureRefs(refs); err == nil {
			t.Fatal("validator accepted an empty control fixture")
		}
	})
	t.Run("duplicate ref", func(t *testing.T) {
		refs := append([]string(nil), contract.ControlFixtureRefs...)
		refs = append(refs, refs[0])
		if err := validateAssetsLifecycleControlFixtureRefs(refs); err == nil {
			t.Fatal("validator accepted a duplicate control fixture")
		}
	})
	t.Run("additional ref accepted", func(t *testing.T) {
		refs := append([]string(nil), contract.ControlFixtureRefs...)
		refs = append(refs, "apps/go-api/internal/parity/testdata/fixtures/assets_lifecycle/future-control-case.json")
		if err := validateAssetsLifecycleControlFixtureRefs(refs); err != nil {
			t.Fatalf("validator rejected an additional control fixture: %v", err)
		}
	})
}
