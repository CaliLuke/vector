package design

import (
	"strconv"
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestExternalIntegrationErrorMetadataClassifiesRetryableFailures(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	service := expr.Root.Service("external_integrations")
	if service == nil || len(service.Methods) != 3 {
		t.Fatalf("missing 3-method external integration design: %+v", service)
	}
	unsplash := service.Method("unsplash")
	if unsplash == nil {
		t.Fatal("missing unsplash method")
	}
	for status := 300; status <= 599; status++ {
		name := "status_" + strconv.Itoa(status)
		failure := unsplash.Error(name)
		if failure == nil {
			t.Fatalf("unsplash missing %s", name)
		}
		_, temporary := failure.Meta["loom:error:temporary"]
		_, fault := failure.Meta["loom:error:fault"]
		_, timeout := failure.Meta["loom:error:timeout"]
		wantTemporary := status == 429 || status == 502 || status == 503
		wantFault := status >= 500
		if temporary != wantTemporary || fault != wantFault || timeout {
			t.Errorf("%s temporary=%v fault=%v timeout=%v; want temporary=%v fault=%v timeout=false", name, temporary, fault, timeout, wantTemporary, wantFault)
		}
	}
}
