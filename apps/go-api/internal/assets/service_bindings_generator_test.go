package assets

import (
	"os/exec"
	"testing"
)

func TestServiceBindingsMatchGeneratedInterface(t *testing.T) {
	command := exec.Command("go", "run", "./bindingsgen", "-check", "-output", "service_bindings.go")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bindings generator check failed: %v\n%s", err, output)
	}
}
