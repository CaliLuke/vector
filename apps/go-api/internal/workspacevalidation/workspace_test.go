package workspacevalidation

import (
	"strings"
	"testing"
)

func TestContainsURLUsesSourceWorkspaceNamePatterns(t *testing.T) {
	for _, value := range []string{
		"https://example.test/path",
		"WWW.example.test",
		"Visit example.com today",
		"example.technology", // The source regexp searches for a match without a domain boundary.
		"example.ıı",         // Python's Unicode IGNORECASE expands [a-z] to dotless i.
		"example.İİ",         // Python's Unicode IGNORECASE expands [a-z] to dotted I.
		"Contact 192.0.2.1",
	} {
		if !ContainsURL(value) {
			t.Errorf("ContainsURL(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"plain workspace", "example", "http://", "http://\u00a0"} {
		if ContainsURL(value) {
			t.Errorf("ContainsURL(%q) = true, want false", value)
		}
	}
	if ContainsURL("http://" + strings.Repeat("x", 1001)) {
		t.Fatal("URL detection must skip input longer than the source 1000-character limit")
	}
	if ContainsURL(strings.Repeat("x", 501) + "http://example.com") {
		t.Fatal("URL detection must truncate each source line to 500 characters")
	}
}

func TestSlugValidationMatchesDRFAndReservedSourceList(t *testing.T) {
	for _, value := range []string{"admin", "api", "accounts"} {
		if !RestrictedSlug(value) {
			t.Errorf("RestrictedSlug(%q) = false, want true", value)
		}
	}
	if RestrictedSlug("Admin") {
		t.Fatal("restricted slug comparison must preserve the source case-sensitive check")
	}
	for _, value := range []string{"valid-slug_1", "Admin"} {
		if !ValidSlug(value) {
			t.Errorf("ValidSlug(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"has.dot", "has space", "has/slash", "東"} {
		if ValidSlug(value) {
			t.Errorf("ValidSlug(%q) = true, want false", value)
		}
	}
}
