package modulereads

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateModuleLinkFieldsMatchesSerializerBranches(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		requiredURL bool
		fields      map[string]any
		wantValues  map[string]any
		wantErrors  map[string]any
	}{
		{
			name:   "create normalizes scheme and ignores read-only deleted_at",
			method: "POST", requiredURL: true,
			fields:     map[string]any{"url": "example.com", "deleted_at": "2020-01-01T00:00:00Z"},
			wantValues: map[string]any{"url": "http://example.com"}, wantErrors: map[string]any{},
		},
		{
			name:       "title-only patch is rejected by url validation",
			method:     "PATCH",
			fields:     map[string]any{"title": "renamed"},
			wantValues: map[string]any{"title": "renamed"},
			wantErrors: map[string]any{"error": "Invalid URL format."},
		},
		{
			name:       "field error precedes missing URL save validation",
			method:     "PATCH",
			fields:     map[string]any{"title": strings.Repeat("x", 256)},
			wantValues: map[string]any{},
			wantErrors: map[string]any{"title": []string{"Ensure this field has no more than 255 characters."}},
		},
		{
			name:   "put requires url",
			method: "PUT", requiredURL: true,
			fields:     map[string]any{"title": "renamed"},
			wantValues: map[string]any{"title": "renamed"},
			wantErrors: map[string]any{"url": []string{"This field is required."}},
		},
		{
			name:   "url must satisfy django URL validator",
			method: "POST", requiredURL: true,
			fields:     map[string]any{"url": "http://bad host"},
			wantValues: map[string]any{}, wantErrors: map[string]any{"url": []string{"Enter a valid URL."}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values, failures := validateModuleLinkFields(test.fields, nil, test.method, test.requiredURL)
			if !reflect.DeepEqual(values, test.wantValues) {
				t.Errorf("values = %#v, want %#v", values, test.wantValues)
			}
			if !reflect.DeepEqual(failures, test.wantErrors) {
				t.Errorf("field errors = %#v, want %#v", failures, test.wantErrors)
			}
		})
	}
}
