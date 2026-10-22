package djangoinput

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

func TestParseJSONRejectsInvalidUTF8WithSourceParseErrors(t *testing.T) {
	prefix := []byte(`{"name":"Changed","unused":"`)
	tests := []struct {
		name string
		tail []byte
		want string
	}{
		{
			name: "invalid start byte",
			tail: []byte{0xff, '"', '}'},
			want: "JSON parse error - 'utf-8' codec can't decode byte 0xff in position 28: invalid start byte",
		},
		{
			name: "invalid continuation byte",
			tail: []byte{0xc3, '"', '}'},
			want: "JSON parse error - 'utf-8' codec can't decode byte 0xc3 in position 28: invalid continuation byte",
		},
		{
			name: "invalid continuation after two valid prefix bytes",
			tail: []byte{0xf0, 0x9f, 0x92, 0x28, '"', '}'},
			want: "JSON parse error - 'utf-8' codec can't decode bytes in position 28-30: invalid continuation byte",
		},
		{
			name: "invalid continuation after one valid prefix byte",
			tail: []byte{0xe2, 0x82, 0x28, '"', '}'},
			want: "JSON parse error - 'utf-8' codec can't decode bytes in position 28-29: invalid continuation byte",
		},
		{
			name: "invalid second byte has singular source position",
			tail: []byte{0xed, 0xa0, 0x80, '"', '}'},
			want: "JSON parse error - 'utf-8' codec can't decode byte 0xed in position 28: invalid continuation byte",
		},
		{
			name: "invalid lead byte has singular source position",
			tail: []byte{0xc0, 0xaf, '"', '}'},
			want: "JSON parse error - 'utf-8' codec can't decode byte 0xc0 in position 28: invalid start byte",
		},
		{
			name: "truncated string remains a JSON syntax error",
			tail: []byte{0xc3},
			want: "JSON parse error - Unterminated string starting at: line 1 column 28 (char 27)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := append(append([]byte(nil), prefix...), test.tail...)
			_, err := Parse(bytes.NewReader(body), "application/json")
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 400 {
				t.Fatalf("Parse error=%v, want HTTP 400", err)
			}
			if response.Body["detail"] != test.want {
				t.Fatalf("parse detail=%v, want %q", response.Body["detail"], test.want)
			}
		})
	}
}

func TestParseJSONTruncatedUTF8AfterCommaDoesNotReportUnterminatedString(t *testing.T) {
	body := []byte(`{"name":"ok",`)
	body = append(body, 0xc3)
	_, err := Parse(bytes.NewReader(body), "application/json")
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 {
		t.Fatalf("Parse error=%v, want HTTP 400", err)
	}
	if strings.Contains(response.Body["detail"].(string), "Unterminated string") {
		t.Fatalf("parse detail=%v, should retain object syntax error instead of misclassifying a string", response.Body["detail"])
	}
}

func TestParseJSONRejectsIncompleteUTF8AfterCompleteValue(t *testing.T) {
	body := append([]byte(`{"name":"Changed"}`), 0xc3)
	_, err := Parse(bytes.NewReader(body), "application/json")
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 {
		t.Fatalf("Parse error=%v, want HTTP 400", err)
	}
	if response.Body["detail"] != "JSON parse error - 'utf-8' codec can't decode byte 0xc3 in position 18: unexpected end of data" {
		t.Fatalf("parse detail=%v, want incomplete UTF-8 error", response.Body["detail"])
	}
}

func TestParseJSONKeepsValidUTF8AndEscapedSurrogates(t *testing.T) {
	body := []byte(`{"unused":"ÿ","icon_prop":{"value":"\ud800"}}`)
	input, err := Parse(bytes.NewReader(body), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if input.Fields["unused"] != "ÿ" || input.Fields["icon_prop"].(map[string]any)["value"] != "�" {
		t.Fatalf("decoded fields=%#v", input.Fields)
	}
	if !bytes.Equal(input.Raw, body) {
		t.Fatalf("valid UTF-8 JSON bytes changed: %q", input.Raw)
	}
}

func TestParseJSONUsesUTF8BodyBytesRegardlessOfCharsetLabel(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "absent charset", contentType: "application/json"},
		{name: "utf8 charset", contentType: "application/json; charset=utf-8"},
		{name: "uppercase utf8 charset", contentType: "application/json; charset=UTF-8"},
		{name: "legacy charset label", contentType: "application/json; charset=iso-8859-1"},
		{name: "unknown charset label", contentType: "application/json; charset=not-a-codec"},
		{name: "empty charset label", contentType: `application/json; charset=""`},
		{name: "extended charset parameter", contentType: "application/json; charset*=UTF-8''iso-8859-1"},
		{name: "discarded extended charset parameter", contentType: "application/json; charset*=iso-8859-1''iso-8859-1"},
		{name: "continued charset parameter", contentType: "application/json; charset*0*=utf-8''iso-; charset*1=8859-1"},
		{name: "extended unknown parameter", contentType: "application/json; x-label*=UTF-8''some%20value"},
		{name: "equal duplicate charset parameters", contentType: "application/json; charset=utf-8; charset=utf-8"},
		{name: "equal duplicate unrelated parameters", contentType: "application/json; x-label=value; x-label=value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := []byte(`{"ascii":"ok","unicode":"café"}`)
			input, err := Parse(bytes.NewReader(body), test.contentType)
			if err != nil {
				t.Fatal(err)
			}
			if input.Fields["ascii"] != "ok" || input.Fields["unicode"] != "café" {
				t.Fatalf("decoded fields=%#v", input.Fields)
			}
			if !bytes.Equal(input.Raw, body) {
				t.Fatalf("valid UTF-8 JSON bytes changed: %q", input.Raw)
			}
		})
	}
}

func TestParseJSONRejectsInvalidUTF8RegardlessOfCharsetLabel(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "absent charset", contentType: "application/json"},
		{name: "utf8 charset", contentType: "application/json; charset=utf-8"},
		{name: "legacy charset label", contentType: "application/json; charset=iso-8859-1"},
		{name: "unknown charset label", contentType: "application/json; charset=not-a-codec"},
		{name: "empty charset label", contentType: `application/json; charset=""`},
		{name: "extended charset parameter", contentType: "application/json; charset*=UTF-8''iso-8859-1"},
		{name: "discarded extended charset parameter", contentType: "application/json; charset*=iso-8859-1''iso-8859-1"},
		{name: "continued charset parameter", contentType: "application/json; charset*0*=utf-8''iso-; charset*1=8859-1"},
	}
	invalidBodies := []struct {
		name string
		body []byte
	}{
		{name: "invalid start byte", body: []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}},
		{name: "invalid continuation byte", body: []byte{'{', '"', 'x', '"', ':', '"', 0xc3, '"', '}'}},
	}
	for _, test := range tests {
		for _, invalidBody := range invalidBodies {
			t.Run(test.name+"/"+invalidBody.name, func(t *testing.T) {
				_, err := Parse(bytes.NewReader(invalidBody.body), test.contentType)
				var response *djangoapi.HTTPError
				if !errors.As(err, &response) || response.Status != 400 {
					t.Fatalf("Parse error=%v, want HTTP 400 for invalid UTF-8 body bytes", err)
				}
			})
		}
	}
}

func TestParseJSONRejectsInvalidMIMEParameters(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		nilBody     bool
	}{
		{name: "unquoted empty charset with body", contentType: "application/json; charset=", body: []byte(`{"name":"Changed"}`)},
		{name: "unquoted empty charset with empty body", contentType: "application/json; charset="},
		{name: "malformed trailing parameter with body", contentType: "application/json; charset=utf-8; unused=", body: []byte(`{"name":"Changed"}`)},
		{name: "malformed trailing parameter with empty body", contentType: "application/json; charset=utf-8; unused="},
		{name: "malformed trailing parameter with nil body", contentType: "application/json; charset=utf-8; unused=", nilBody: true},
		{name: "conflicting duplicate parameters with body", contentType: "application/json; charset=utf-8; charset=iso-8859-1", body: []byte(`{"name":"Changed"}`)},
		{name: "conflicting duplicate parameters with empty body", contentType: "application/json; charset=utf-8; charset=iso-8859-1"},
		{name: "conflicting duplicate parameters with nil body", contentType: "application/json; charset=utf-8; charset=iso-8859-1", nilBody: true},
		{name: "conflicting duplicate parameters despite quoted escape", contentType: `application/json; charset="utf-8"; note="a\\\"b"; charset=iso-8859-1`, body: []byte(`{"name":"Changed"}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body io.Reader = bytes.NewReader(test.body)
			if test.nilBody {
				body = nil
			}
			_, err := Parse(body, test.contentType)
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 415 {
				t.Fatalf("Parse error=%v, want HTTP 415", err)
			}
			if response.Body["detail"] != "Invalid Content-Type for application/json." {
				t.Fatalf("parse detail=%v, want invalid Content-Type detail", response.Body["detail"])
			}
		})
	}
}

func TestParseJSONKeepsNilAndEmptyBodiesWithoutContentType(t *testing.T) {
	for _, body := range []io.Reader{nil, bytes.NewReader(nil)} {
		input, err := Parse(body, "")
		if err != nil {
			t.Fatalf("Parse empty body without Content-Type: %v", err)
		}
		if len(input.Fields) != 0 {
			t.Fatalf("empty body fields=%#v, want empty fields", input.Fields)
		}
	}
}

func TestParseJSONValidationPreservesNonJSONEmptyBodyBehavior(t *testing.T) {
	for _, contentType := range []string{"text/plain; =", "multipart/form-data; boundary="} {
		for _, body := range []io.Reader{nil, bytes.NewReader(nil)} {
			if _, err := Parse(body, contentType); err != nil {
				t.Fatalf("non-JSON empty body with %q changed behavior: %v", contentType, err)
			}
		}
	}
}
