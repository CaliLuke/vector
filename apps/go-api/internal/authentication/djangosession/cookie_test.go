package djangosession

import (
	"net/http"
	"testing"
)

func TestCookieFromHeaderValues(t *testing.T) {
	tests := []struct {
		name    string
		header  []string
		cookie  string
		want    string
		present bool
	}{
		{name: "missing", header: []string{"other=value"}, cookie: "session-id"},
		{name: "empty header", header: []string{""}, cookie: "session-id"},
		{name: "last duplicate", header: []string{"session-id=first; session-id=last"}, cookie: "session-id", want: "last", present: true},
		{name: "joined header fields", header: []string{"session-id=first", "session-id=last"}, cookie: "session-id", want: "first,session-id=last", present: true},
		{name: "nameless chunk", header: []string{"orphan"}, cookie: "", want: "orphan", present: true},
		{name: "octal and escaped quote", header: []string{`session-id="a\141\"b"`}, cookie: "session-id", want: `aa"b`, present: true},
		{name: "latin1", header: []string{"session-id=\xe9"}, cookie: "session-id", want: "é", present: true},
		{name: "unquoted unchanged", header: []string{`session-id=plain\141`}, cookie: "session-id", want: `plain\141`, present: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := CookieFromHeaderValues(test.header, test.cookie)
			if !test.present {
				if got != nil {
					t.Fatalf("CookieFromHeaderValues() = %#v, want nil", got)
				}
				return
			}
			if got == nil || got.Name != test.cookie || got.Value != test.want {
				t.Fatalf("CookieFromHeaderValues() = %#v, want %#v", got, &http.Cookie{Name: test.cookie, Value: test.want})
			}
		})
	}
}
