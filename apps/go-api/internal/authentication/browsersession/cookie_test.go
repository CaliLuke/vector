package browsersession

import "testing"

func TestSessionCookieHeaders(t *testing.T) {
	for _, test := range []struct {
		name    string
		headers []string
		want    string
		present bool
	}{
		{name: "missing", headers: []string{"other=value"}},
		{name: "empty header", headers: []string{""}},
		{name: "first matching cookie", headers: []string{"session-id=first; session-id=last"}, want: "first", present: true},
		{name: "multiple header fields", headers: []string{"other=value", "session-id=native"}, want: "native", present: true},
		{name: "quoted ASCII", headers: []string{`session-id="native-value"`}, want: "native-value", present: true},
		{name: "octal escapes rejected", headers: []string{`session-id="native\141"`}},
		{name: "non-ASCII rejected", headers: []string{"session-id=é"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := CookieFromHeaderValues(test.headers, "session-id")
			if !test.present {
				if got != nil {
					t.Fatalf("unexpected cookie: %+v", got)
				}
				return
			}
			if got == nil || got.Name != "session-id" || got.Value != test.want {
				t.Fatalf("cookie=%+v want value=%q", got, test.want)
			}
		})
	}
}
