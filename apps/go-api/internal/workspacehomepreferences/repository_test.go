package workspacehomepreferences

import (
	"testing"
)

func TestNewRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}
