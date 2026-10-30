package instancemail

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/textproto"
	"strings"
	"testing"
)

func TestAttachmentFixtureRejectsMalformedDisposition(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprintf("multipart=%v", nested), func(t *testing.T) {
			header := textproto.MIMEHeader{"Content-Type": {"text/plain"}, "Content-Disposition": {"attachment; filename"}}
			body := "unexpected malformed attachment"
			if nested {
				body = "--fixture\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename\r\n\r\n" + body + "\r\n--fixture--\r\n"
				header = textproto.MIMEHeader{"Content-Type": {"multipart/mixed; boundary=fixture"}}
			}
			_, err := readAttachments(header, strings.NewReader(body))
			if !errors.Is(err, mime.ErrInvalidMediaParameter) {
				t.Fatalf("malformed attachment must fail inspection, got %v", err)
			}
		})
	}
}

func TestAttachmentFixtureAcceptsInlineBody(t *testing.T) {
	for _, disposition := range []string{"", "inline"} {
		t.Run(disposition, func(t *testing.T) {
			result, err := readAttachments(textproto.MIMEHeader{"Content-Type": {"text/plain"}, "Content-Disposition": {disposition}}, strings.NewReader("message body"))
			if err != nil || len(result) != 0 {
				t.Fatalf("inline message returned attachments=%v error=%v", result, err)
			}
		})
	}
}

type interruptedMIMEReader struct{}

func (interruptedMIMEReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("interrupted body: %w", io.EOF)
}

func TestAttachmentFixtureRetainsWrappedEOF(t *testing.T) {
	_, err := readAttachments(textproto.MIMEHeader{"Content-Type": {"multipart/mixed; boundary=fixture"}}, interruptedMIMEReader{})
	if err == nil || !errors.Is(err, io.EOF) {
		t.Fatalf("interrupted MIME input must remain a failure, got %v", err)
	}
}
