package cyclereads

import (
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func TestFrozenPropertyBindingsRejectUnevaluatedOperands(t *testing.T) {
	bindings := &cyclePropertyBindings{
		entries: map[string]*cyclePropertyEntry{},
		validate: func(string, any) (*workitemtypes.ScopedPropertyPredicate, bool, error) {
			return &workitemtypes.ScopedPropertyPredicate{}, true, nil
		},
	}
	args := []any{}
	const field = "customproperty_10000000-0000-4000-8000-000000000001__exact"
	if _, handled, err := bindings.field(field, float64(2), &args); !handled || err != nil {
		t.Fatalf("prepare: %v %v", handled, err)
	}
	bindings.validate = nil
	_, handled, err := bindings.field(field, float64(9), &args)
	var invalid *FilterError
	if !handled || !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("unevaluated operand silently accepted: %v", err)
	}
	if _, handled, err = bindings.field(field, float64(2), &args); !handled || err != nil {
		t.Fatalf("frozen prepared operand rejected: %v %v", handled, err)
	}
}
