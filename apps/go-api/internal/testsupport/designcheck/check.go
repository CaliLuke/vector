// Package designcheck validates authored API contracts in their isolated test binary.
package designcheck

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Validate evaluates and validates deferred DSL bodies. Compiling generated
// transport alone cannot detect an invalid edit to its source design.
func Validate(t testing.TB) {
	t.Helper()
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	if len(expr.Root.Services) == 0 {
		t.Fatal("authored contract has no services")
	}
}
