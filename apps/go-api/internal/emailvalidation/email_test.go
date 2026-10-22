package emailvalidation

import "testing"

func TestDjangoEmailCompatibility(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{name: "localhost domain", email: "person@localhost", valid: true},
		{name: "bracketed IPv6 domain", email: "person@[2001:db8::1]", valid: true},
		{name: "quoted local part", email: `"quoted"@example.com`, valid: true},
		{name: "domain without dot", email: "person@example", valid: false},
		{name: "display name", email: "Person <person@example.com>", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidDjangoEmail(test.email); got != test.valid {
				t.Errorf("ValidDjangoEmail(%q)=%t, want %t", test.email, got, test.valid)
			}
		})
	}
}

func TestNormalizePythonEmailUsesUnicodeStripAndFullLowercase(t *testing.T) {
	got := NormalizePythonEmail("\u2003İ@LOCALHOST\u001c")
	want := "i\u0307@localhost"
	if got != want {
		t.Fatalf("NormalizePythonEmail()=%q, want %q", got, want)
	}
}
