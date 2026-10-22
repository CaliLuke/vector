package backgroundtasks

import (
	"strings"
	"testing"
)

func TestWorkspaceSeedSourceContent(t *testing.T) {
	r, err := decodeSeedRecords()
	if err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"projects": 1, "states": 5, "issues": 7, "issue_sequences": 7, "cycles": 2, "modules": 3, "pages": 2, "issue_activities": 7, "module_issues": 11} {
		if got := len(r.Rows[table]); got != want {
			t.Errorf("%s rows=%d want %d", table, got, want)
		}
	}
	if description := r.Rows["projects"][0]["description"].(string); !strings.Contains(description, "Vector Demo Project") || strings.Contains(description, "Plane") {
		t.Fatalf("workspace seed project description has unexpected product branding: %q", description)
	}
	if name := r.Rows["issues"][2]["name"].(string); name != "Welcome to Vector 👋" {
		t.Fatalf("workspace seed welcome issue name = %q", name)
	}
	if email := r.Rows["users"][0]["email"].(string); !strings.HasSuffix(email, "@localhost") {
		t.Fatalf("workspace seed bot email uses unexpected domain: %q", email)
	}
	for _, row := range r.Rows["issues"] {
		for _, key := range []string{"description_html", "description_stripped"} {
			if content, ok := row[key].(string); ok && (strings.Contains(content, "docs.plane.so") || strings.Contains(content, "media.docs.plane.so")) {
				t.Fatalf("workspace seed %s retains an upstream Plane link", key)
			}
		}
	}
	sequences := map[any]any{}
	for _, row := range r.Rows["issue_sequences"] {
		if _, ok := sequences[row["issue_id"]]; ok {
			t.Fatal("duplicate issue sequence")
		}
		sequences[row["issue_id"]] = row["sequence"]
	}
	for _, row := range r.Rows["issues"] {
		if sequences[row["id"]] != row["sequence_id"] {
			t.Fatal("incorrect issue sequence")
		}
	}
	for input, want := range map[string]string{"  straße 🎸": "STRASS", "é中 1234": "É中123", "ßabcd": "SSABCD"} {
		if got := seedIdentifier(input); got != want {
			t.Errorf("%q: got %q want %q", input, got, want)
		}
	}
}
