package authredirects

import (
	"net/url"
	"strings"
	"testing"
)

func TestValidateNextPathMatchesDjangoPathRules(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "relative path rejected", input: "onboarding", want: ""},
		{name: "local path", input: "/workspace/issues", want: "/workspace/issues"},
		{name: "absolute url retains path", input: "https://outside.example/path?q=secret", want: "/path"},
		{name: "backslashes removed", input: `/a\\b`, want: "/ab"},
		{name: "traversal rejected", input: "/a/../b", want: ""},
		{name: "suspicious scheme rejected", input: "/javascript:alert(1)", want: ""},
		{name: "unicode length counts runes", input: "/" + strings.Repeat("界", 500), want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidateNextPath(test.input); got != test.want {
				t.Errorf("ValidateNextPath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestSafeRedirectURLPreservesDestinationAndOrderedErrors(t *testing.T) {
	got := SafeRedirectURL("https://app.example/", "/projects?tab=issues", []QueryParam{
		{Name: "error_code", Value: "5070"},
		{Name: "error_message", Value: "REQUIRED_EMAIL_PASSWORD_SIGN_IN"},
		{Name: "email", Value: "a+b@example.test"},
	}, []string{"app.example"})
	want := "https://app.example/?next_path=%2Fprojects%3Ftab%3Dissues&error_code=5070&error_message=REQUIRED_EMAIL_PASSWORD_SIGN_IN&email=a%2Bb%40example.test"
	if got != want {
		t.Fatalf("SafeRedirectURL() = %q, want %q", got, want)
	}
}

func TestSpaceRedirectURLFallsBackWhenHostIsNotConfigured(t *testing.T) {
	if got := SpaceRedirectURL("https://space.example/spaces/", "/path", []string{"other.example"}); got != "https://space.example/spaces/" {
		t.Fatalf("SpaceRedirectURL() = %q, want configured base fallback", got)
	}
}

func TestSafeRedirectURLKeepsNestedQueryAndAnchorInDestination(t *testing.T) {
	destination := "/projects?view=board&label=a+b#selected"
	parsed, err := url.Parse(SafeRedirectURL("https://app.example/", destination, nil, []string{"app.example"}))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("next_path") != destination || len(parsed.Query()) != 1 || parsed.Fragment != "" {
		t.Fatalf("destination escaped its query value: %s", parsed)
	}
}
