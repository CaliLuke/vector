package design

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport/designcheck"
)

// Validate the authored contract, not only the previously generated transport.
func TestAuthoredContractIsValid(t *testing.T) { designcheck.Validate(t) }
