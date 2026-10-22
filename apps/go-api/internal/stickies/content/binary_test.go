package content

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestValidateBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want error
	}{
		{name: "nil"},
		{name: "empty", data: []byte{}},
		{name: "four bytes", data: []byte("okay")},
		{name: "three bytes", data: []byte("abc"), want: ErrBinaryTooShort},
		{name: "HTML", data: []byte("prefix<HTML lang=en>"), want: ErrSuspiciousBinary},
		{name: "doctype", data: []byte("<!DOCTYPE html>"), want: ErrSuspiciousBinary},
		{name: "script", data: []byte("xx<ScRiPt>x"), want: ErrSuspiciousBinary},
		{name: "JavaScript URL", data: []byte("xxJAVASCRIPT:alert(1)"), want: ErrSuspiciousBinary},
		{name: "data URL", data: []byte("xxDATA:text/plain,x"), want: ErrSuspiciousBinary},
		{name: "iframe", data: []byte("xx<IFRAME src=x>"), want: ErrSuspiciousBinary},
		{name: "pattern after 200 decoded characters", data: []byte(strings.Repeat("a", 200) + "<script>okay"), want: nil},
		{name: "invalid UTF-8 ignored", data: append([]byte{0xff, 0xfe}, []byte("safe")...), want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidateBinary(test.data); !errors.Is(got, test.want) {
				t.Fatalf("ValidateBinary() error = %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateBinarySizeBoundary(t *testing.T) {
	t.Parallel()

	atLimit := make([]byte, MaxContentBytes)
	for i := range atLimit {
		atLimit[i] = 0xff
	}
	if err := ValidateBinary(atLimit); err != nil {
		t.Fatalf("ValidateBinary(at limit) error = %v", err)
	}
	if err := ValidateBinary(append(atLimit, 0xff)); !errors.Is(err, ErrBinaryTooLarge) {
		t.Fatalf("ValidateBinary(over limit) error = %v, want %v", err, ErrBinaryTooLarge)
	}
}

func TestValidateBase64BinaryUsesPythonDecodeSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  error
	}{
		{name: "empty"},
		{name: "valid", value: base64.StdEncoding.EncodeToString([]byte("document"))},
		{name: "whitespace and nonalphabet discarded", value: "Z G9j\n!dW1lbnQ="},
		{name: "junk decodes empty", value: "!!!!", want: ErrBinaryTooShort},
		{name: "extra padding accepted", value: "ZG9jdW1lbnQ===", want: nil},
		{name: "data after complete padding ignored", value: "ZG9jZA==junk", want: nil},
		{name: "incorrect padding", value: "YQ", want: ErrInvalidBase64},
		{name: "insufficient padding", value: "YQ=", want: ErrInvalidBase64},
		{name: "non-ASCII", value: "é", want: ErrInvalidBase64},
		{name: "decoded too short", value: "YQ==", want: ErrBinaryTooShort},
		{name: "decoded suspicious", value: base64.StdEncoding.EncodeToString([]byte("<script>")), want: ErrSuspiciousBinary},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := ValidateBase64Binary(test.value)
			if !errors.Is(got, test.want) {
				t.Fatalf("ValidateBase64Binary() error = %v, want %v", got, test.want)
			}
			if test.want != nil && got.Error() != test.want.Error() {
				t.Fatalf("ValidateBase64Binary() message = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDecodeBase64BinaryReturnsValidatedDecodedContent(t *testing.T) {
	decoded, err := DecodeBase64Binary("SGVsbG8=")
	if err != nil {
		t.Fatalf("DecodeBase64Binary() error = %v", err)
	}
	if string(decoded) != "Hello" {
		t.Fatalf("DecodeBase64Binary() = %q, want Hello", decoded)
	}

	if decoded, err := DecodeBase64Binary("not-base64!"); err != ErrInvalidBase64 {
		t.Fatalf("DecodeBase64Binary(invalid) = (%q, %v), want ErrInvalidBase64", decoded, err)
	}
}

func TestDecodePythonBase64MatchesCPython312Matrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  string
		err   bool
	}{
		{value: "", want: ""},
		{value: "!!!!", want: ""},
		{value: "Y W\nJj", want: "abc"},
		{value: "YQ", err: true},
		{value: "YQ=", err: true},
		{value: "YQ===", want: "a"},
		{value: "YQ==junk", want: "a"},
		{value: "YWJj=", want: "abc"},
		{value: "YWJj=junk", want: "abc\x8e\xe9\xe4"},
		{value: "YWJjZA==junk", want: "abcd"},
		{value: "=junk", want: "\x8e\xe9\xe4"},
		{value: "=abc", err: true},
		{value: "====", want: ""},
		{value: "AA==xx", want: "\x00"},
		{value: "AA=x", err: true},
		{value: "é", err: true},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			t.Parallel()
			got, err := decodePythonBase64(test.value)
			if (err != nil) != test.err {
				t.Fatalf("decodePythonBase64() error = %v, want error %t", err, test.err)
			}
			if string(got) != test.want {
				t.Fatalf("decodePythonBase64() = %x, want %x", got, test.want)
			}
		})
	}
}

func TestValidationMessagesMatchPythonSource(t *testing.T) {
	t.Parallel()

	errorsToCheck := []error{
		ErrInvalidBase64,
		ErrBinaryTooLarge,
		ErrBinaryTooShort,
		ErrSuspiciousBinary,
		ErrHTMLTooLarge,
		ErrHTMLSanitizationFailed,
	}
	want := []string{
		"Invalid base64 encoding",
		"Binary data exceeds maximum size limit (10MB)",
		"Binary data too short to be valid document format",
		"Binary data contains suspicious content patterns",
		"HTML content exceeds maximum size limit (10MB)",
		"Failed to sanitize HTML",
	}
	for i, validationError := range errorsToCheck {
		if got := validationError.Error(); got != want[i] {
			t.Fatalf("message %d = %q, want %q", i, got, want[i])
		}
	}
}

func FuzzValidateBinaryDoesNotPanic(f *testing.F) {
	f.Add([]byte("document"))
	f.Add([]byte("<script>alert(1)</script>"))
	f.Add([]byte{0xff, 0xfe, 0xfd, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = ValidateBinary(data)
	})
}
