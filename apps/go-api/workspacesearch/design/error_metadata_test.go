package design

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestWorkspaceSearchServerFailureIsClassifiedAsFault(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	service := expr.Root.Service("workspace_search")
	if service == nil || len(service.Methods) != 2 {
		t.Fatalf("missing 2-method workspace search design: %+v", service)
	}
	for _, method := range service.Methods {
		failure := method.Error("status_500")
		if failure == nil {
			t.Fatalf("%s missing status_500", method.Name)
		}
		_, fault := failure.Meta["loom:error:fault"]
		_, temporary := failure.Meta["loom:error:temporary"]
		_, timeout := failure.Meta["loom:error:timeout"]
		if !fault || temporary || timeout {
			t.Errorf("%s/status_500 fault=%v temporary=%v timeout=%v; want fault=true temporary=false timeout=false", method.Name, fault, temporary, timeout)
		}
	}
}
