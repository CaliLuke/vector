package logredact

import "testing"

func TestRedactSensitiveTextMasksCredentialValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "quoted JSON values",
			input: `{"password":"p@ss","access_token":"bearer","client_secret":"secret"}`,
			want:  `{"password":"[redacted]","access_token":"[redacted]","client_secret":"[redacted]"}`,
		},
		{
			name:  "single quoted text value",
			input: `secret = 'private value'` + "\nnext line",
			want:  `secret = '[redacted]'` + "\nnext line",
		},
		{
			name:  "unquoted value",
			input: "token: value\nstatus: 500",
			want:  "token: [redacted]\nstatus: 500",
		},
		{
			name:  "authorization header",
			input: "Authorization: Bearer private\r\nX-Request-ID: req-1",
			want:  "Authorization: [redacted]\r\nX-Request-ID: req-1",
		},
		{
			name:  "cookie header",
			input: "Set-Cookie: session=private; HttpOnly\npath=/",
			want:  "Set-Cookie: [redacted]\npath=/",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := RedactSensitiveText(test.input); got != test.want {
				t.Fatalf("RedactSensitiveText() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRedactSensitiveTextPreservesNonSecretText(t *testing.T) {
	const input = "request_id=req-1 status=503 message=upstream unavailable\n"
	if got := RedactSensitiveText(input); got != input {
		t.Fatalf("non-secret text changed: got %q, want %q", got, input)
	}
}

func TestRedactSensitiveTextHandlesCaseAndCredentialKeyVariants(t *testing.T) {
	const input = "PASSWORD=\"a\"\nAUTHORIZATION='b'\nCookie: c\nAPI-TOKEN=d"
	const want = "PASSWORD=\"[redacted]\"\nAUTHORIZATION='[redacted]'\nCookie: [redacted]\nAPI-TOKEN=[redacted]"
	if got := RedactSensitiveText(input); got != want {
		t.Fatalf("case-insensitive credential redaction = %q, want %q", got, want)
	}
}
