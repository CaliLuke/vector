package parity

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParityFilesRejectInvalidInputs(t *testing.T) {
	readers := []struct {
		name string
		load func(string) error
	}{
		{"normalizations", func(path string) error { _, err := LoadNormalizations(path); return err }},
		{"response", func(path string) error { _, err := LoadCapturedResponse(path); return err }},
	}
	for _, reader := range readers {
		t.Run(reader.name, func(t *testing.T) {
			for _, test := range []struct {
				name, data string
				missing    bool
			}{
				{name: "missing", missing: true},
				{name: "malformed", data: "{"},
				{name: "unknown field", data: `{"unreviewed":true}`},
				{name: "trailing value", data: `{} {}`},
				{name: "trailing malformed", data: `{} !`},
			} {
				t.Run(test.name, func(t *testing.T) {
					path := filepath.Join(t.TempDir(), "input.json")
					if !test.missing {
						if err := os.WriteFile(path, []byte(test.data), 0600); err != nil {
							t.Fatal(err)
						}
					}
					err := reader.load(path)
					if err == nil {
						t.Fatal("invalid input was accepted")
					}
					if test.missing && !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("missing-file cause lost: %v", err)
					}
				})
			}
		})
	}
}

func TestLoadParityFilesPreserveDeclaredData(t *testing.T) {
	dir := t.TempDir()
	normalizations := filepath.Join(dir, "normalizations.json")
	if err := os.WriteFile(normalizations, []byte(`{"volatile_fields":[{"table":"users","column":"updated_at"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	rules, err := LoadNormalizations(normalizations)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules.VolatileFields) != 1 || rules.VolatileFields[0].Table != "users" || rules.VolatileFields[0].Column != "updated_at" || string(rules.VolatileFields[0].Replacement) != `"<volatile>"` {
		t.Fatalf("normalization rules changed: %#v", rules)
	}
	responsePath := filepath.Join(dir, "response.json")
	if writeErr := os.WriteFile(responsePath, []byte(`{"status":200,"body":{"ok":true},"headers":{"ETag":"tag"},"cookies":{}}`), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}
	response, err := LoadCapturedResponse(responsePath)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != 200 || string(response.Body) != `{"ok":true}` || string(response.Headers["ETag"]) != `"tag"` || len(response.Cookies) != 0 {
		t.Fatalf("captured response changed: %#v", response)
	}
}
