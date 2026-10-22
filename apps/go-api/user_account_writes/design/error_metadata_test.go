package design

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestUserAccountWritesRateLimitIsTemporary(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	service := expr.Root.Service("user_account_writes")
	if service == nil || len(service.Methods) != 4 {
		t.Fatalf("missing four-method user account writes design: %+v", service)
	}
	for _, method := range service.Methods {
		err := method.Error("rate_limited")
		if err == nil {
			t.Fatalf("%s missing rate_limited error", method.Name)
		}
		if _, temporary := err.Meta["loom:error:temporary"]; !temporary {
			t.Errorf("%s rate_limited error is not classified as temporary", method.Name)
		}
	}
}
