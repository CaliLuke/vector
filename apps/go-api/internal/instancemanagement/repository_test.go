package instancemanagement

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
)

func TestWorkspacePageWindowMatchesDjangoCursorProjection(t *testing.T) {
	tests := []struct {
		name    string
		perPage string
		cursor  string
		want    [3]int64
		wantErr string
	}{
		{name: "defaults", want: [3]int64{10, 0, 0}},
		{name: "cursor limit does not control request page size", perPage: "10", cursor: "5:1:0", want: [3]int64{10, 1, 10}},
		{name: "float cursor limit", perPage: "10", cursor: "10.0:1:0", want: [3]int64{10, 1, 10}},
		{name: "python decimal integer syntax", perPage: "1_0", cursor: "10:1:0", want: [3]int64{10, 1, 10}},
		{name: "reject malformed cursor", perPage: "10", cursor: "not-a-cursor", wantErr: "Invalid cursor parameter."},
		{name: "reject malformed per page", perPage: "ten", wantErr: "Invalid per_page parameter."},
		{name: "reject zero page size", perPage: "0", wantErr: "Invalid per_page parameter."},
		{name: "reject negative per page", perPage: "-1", wantErr: "Invalid per_page parameter."},
		{name: "reject page size above source maximum", perPage: "11", wantErr: "Invalid per_page value. Cannot exceed 10."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limit, page, offset, err := workspacePageWindow(test.perPage, test.cursor)
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("workspacePageWindow() error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil || [3]int64{limit, page, offset} != test.want {
				t.Fatalf("workspacePageWindow() = (%d,%d,%d), %v; want %v", limit, page, offset, err, test.want)
			}
		})
	}
}

func TestWorkspaceBackgroundColorMatchesDjangoDefaultFormat(t *testing.T) {
	for range 100 {
		color, err := randomWorkspaceBackgroundColor()
		if err != nil {
			t.Fatal(err)
		}
		if len(color) != 7 || color[0] != '#' {
			t.Fatalf("randomWorkspaceBackgroundColor() = %q, want # plus six hex digits", color)
		}
		for _, digit := range color[1:] {
			if !strings.ContainsRune("0123456789abcdefABCDEF", digit) {
				t.Fatalf("randomWorkspaceBackgroundColor() = %q, contains non-source hex digit %q", color, digit)
			}
		}
	}
}

func TestWorkspaceCharFieldUsesDRFUnicodeWhitespaceTrim(t *testing.T) {
	for _, test := range []struct {
		raw, want, wantError string
	}{
		{raw: `"http:// "`, want: "http://"},
		{raw: `"東 "`, want: "東"},
		{raw: `" "`, wantError: "This field may not be blank."},
	} {
		got, present, truthy, failure := workspaceCharField(json.RawMessage(test.raw))
		if !present || !truthy || got != test.want || failure != test.wantError {
			t.Fatalf("workspaceCharField(%s) = (%q,%t,%t,%q), want (%q,true,true,%q)", test.raw, got, present, truthy, failure, test.want, test.wantError)
		}
	}
}

func TestWorkspaceNameValidationUsesUnicodeAlphanumericAndPythonLength(t *testing.T) {
	for _, name := range []string{"東", "١", "Ⅷ", "a"} {
		if !hasAlphaNumeric(name) {
			t.Errorf("hasAlphaNumeric(%q) = false, want true", name)
		}
	}
	if hasAlphaNumeric("-_________-") {
		t.Fatal("symbol-only workspace name unexpectedly passed alphanumeric validation")
	}

	name80, name81 := strings.Repeat("東", 80), strings.Repeat("東", 81)
	if utf8.RuneCountInString(name80) != 80 || len(name80) <= 80 {
		t.Fatal("test vector must be 80 Python characters and over 80 UTF-8 bytes")
	}
	if workspaceFieldsExceedLength(name80, strings.Repeat("s", 48)) {
		t.Fatal("80-code-point name and 48-code-point slug must fit source limits")
	}
	if !workspaceFieldsExceedLength(name81, "slug") || !workspaceFieldsExceedLength("name", strings.Repeat("s", 49)) {
		t.Fatal("81-code-point name and 49-code-point slug must exceed source limits")
	}
}

func TestWorkspaceSearchPatternEscapesDjangoContainsWildcards(t *testing.T) {
	const search = `%_\`
	const want = `%\%\_\\%`
	got := "%" + issuequery.EscapeLikePattern(search) + "%"
	if got != want {
		t.Fatalf("workspace search pattern = %q, want %q", got, want)
	}
}

func TestCreateWorkspaceReturnsSourceValidationFailuresBeforeDatabaseAccess(t *testing.T) {
	tests := []struct {
		name, workspaceName, slug, field, message string
	}{
		{name: "bare domain", workspaceName: "example.com", slug: "has.dot", field: "name", message: "Name must not contain URLs"},
		{name: "IPv4 address", workspaceName: "192.0.2.1", slug: "has.dot", field: "name", message: "Name must not contain URLs"},
		{name: "reserved slug", workspaceName: "Workspace", slug: "admin", field: "slug", message: "Slug is not valid"},
		{name: "invalid slug field", workspaceName: "Workspace", slug: "has.dot", field: "slug", message: `Enter a valid "slug" consisting of letters, numbers, underscores or hyphens.`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, _ := json.Marshal(test.workspaceName)
			slug, _ := json.Marshal(test.slug)
			repository := &Repository{}
			_, err := repository.CreateWorkspace(t.Context(), uuid.Nil(), map[string]json.RawMessage{"name": name, "slug": slug}, "UTC")
			var validation *WorkspaceValidationError
			if !errors.As(err, &validation) || workspaceValidationMessage(validation, test.field) != test.message {
				t.Fatalf("CreateWorkspace() error = %#v, want %s validation %q", err, test.field, test.message)
			}
		})
	}

	name, _ := json.Marshal(strings.Repeat("東", 80) + " ")
	slug, _ := json.Marshal("raw-length-before-trim")
	_, err := (&Repository{}).CreateWorkspace(t.Context(), uuid.Nil(), map[string]json.RawMessage{"name": name, "slug": slug}, "UTC")
	var validation *WorkspaceValidationError
	if !errors.As(err, &validation) || validation.Message != "The maximum length for name is 80 and for slug is 48" {
		t.Fatalf("CreateWorkspace with 80 code points plus NBSP error = %#v, want raw length error before CharField trim", err)
	}

	for _, test := range []struct {
		name, slug json.RawMessage
		field      string
	}{
		{name: json.RawMessage(`123`), slug: json.RawMessage(`"has.dot"`), field: "name"},
		{name: json.RawMessage(`true`), slug: json.RawMessage(`"has.dot"`), field: "name"},
		{name: json.RawMessage(`"Workspace"`), slug: json.RawMessage(`123`), field: "slug"},
		{name: json.RawMessage(`"Workspace"`), slug: json.RawMessage(`true`), field: "slug"},
	} {
		_, err := (&Repository{}).CreateWorkspace(t.Context(), uuid.Nil(), map[string]json.RawMessage{"name": test.name, "slug": test.slug}, "UTC")
		var validation *WorkspaceValidationError
		if !errors.As(err, &validation) || workspaceValidationMessage(validation, test.field) != "Not a valid string." {
			t.Fatalf("CreateWorkspace(%s,%s) error = %#v, want %s Not a valid string validation", test.name, test.slug, err, test.field)
		}
	}
	for _, zeroValue := range []json.RawMessage{json.RawMessage(`0`), json.RawMessage(`false`), json.RawMessage(`null`)} {
		_, err := (&Repository{}).CreateWorkspace(t.Context(), uuid.Nil(), map[string]json.RawMessage{"name": zeroValue, "slug": json.RawMessage(`"slug"`)}, "UTC")
		if !errors.As(err, &validation) || validation.Message != "Both name and slug are required" {
			t.Fatalf("CreateWorkspace(%s,slug) error = %#v, want source required-field precedence", zeroValue, err)
		}
	}
}

func workspaceValidationMessage(validation *WorkspaceValidationError, field string) string {
	if validation == nil {
		return ""
	}
	if validation.Field == field {
		return validation.Message
	}
	if messages := validation.Fields[field]; len(messages) > 0 {
		return messages[0]
	}
	return ""
}

func TestWorkspaceBackgroundColorFieldConstraints(t *testing.T) {
	for _, value := range []string{"", strings.Repeat("a", 256)} {
		raw, _ := json.Marshal(value)
		_, err := workspaceBackgroundColor(map[string]json.RawMessage{"background_color": raw})
		var validation *WorkspaceValidationError
		if !errors.As(err, &validation) || validation.Field != "background_color" {
			t.Fatalf("color %q: expected field validation, got %v", value, err)
		}
	}
	got, err := workspaceBackgroundColor(map[string]json.RawMessage{"background_color": json.RawMessage(`" #123456 "`)})
	if err != nil || got != "#123456" {
		t.Fatalf("trimmed color = %q, %v", got, err)
	}
}

func TestWorkspaceCharFieldUsesSharedPythonTruthiness(t *testing.T) {
	for _, raw := range []string{"0", "0.0", "-0.0", "false", "null", "[]", "{}"} {
		_, present, truthy, _ := workspaceCharField(json.RawMessage(raw))
		if !present || truthy {
			t.Fatalf("workspaceCharField(%s) presence/truthiness = %t/%t", raw, present, truthy)
		}
	}
}

func TestWorkspaceCompanyRoleMatchesDjangoTextFieldPreparation(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string]json.RawMessage
		want    any
		wantErr bool
	}{
		{name: "omitted defaults to empty string", fields: map[string]json.RawMessage{}, want: ""},
		{name: "string", fields: map[string]json.RawMessage{"company_role": json.RawMessage(`"Founder"`)}, want: "Founder"},
		{name: "number", fields: map[string]json.RawMessage{"company_role": json.RawMessage(`123`)}, want: "123"},
		{name: "boolean", fields: map[string]json.RawMessage{"company_role": json.RawMessage(`true`)}, want: "True"},
		{name: "list uses Python representation", fields: map[string]json.RawMessage{"company_role": json.RawMessage(`[1,"x"]`)}, want: `[1, 'x']`},
		{name: "object uses Python representation", fields: map[string]json.RawMessage{"company_role": json.RawMessage(`{"a":1}`)}, want: `{'a': 1}`},
		{name: "null remains SQL null", fields: map[string]json.RawMessage{"company_role": json.RawMessage(`null`)}, want: nil},
		{name: "invalid JSON", fields: map[string]json.RawMessage{"company_role": json.RawMessage(`{`)}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := workspaceCompanyRole(test.fields)
			if (err != nil) != test.wantErr {
				t.Fatalf("workspaceCompanyRole() error = %v, wantErr %t", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("workspaceCompanyRole() = %#v, want %#v", got, test.want)
			}
		})
	}
}
