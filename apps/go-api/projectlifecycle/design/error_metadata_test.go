package design

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestProjectLifecycleErrorMetadataRemainsPerError(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	service := expr.Root.Service("ProjectLifecycle")
	if service == nil || len(service.Methods) != 15 {
		t.Fatalf("missing 15-method lifecycle design: %+v", service)
	}
	for _, method := range service.Methods {
		for _, name := range []string{"status_400", "status_401", "status_429", "status_500"} {
			err := method.Error(name)
			// The source-backed detail read has no request body or pagination
			// validation and does not declare a bad-request response.
			if err == nil && method.Name == "GetAPIV1WorkspacesBySlugProjectsByPk" && name == "status_400" {
				continue
			}
			if err == nil {
				t.Fatalf("%s missing %s", method.Name, name)
			}
			_, temporary := err.Meta["loom:error:temporary"]
			_, fault := err.Meta["loom:error:fault"]
			_, timeout := err.Meta["loom:error:timeout"]
			if temporary != (name == "status_429") || fault != (name == "status_500") || timeout {
				t.Errorf("%s/%s temporary=%v fault=%v timeout=%v", method.Name, name, temporary, fault, timeout)
			}
		}
	}
}
