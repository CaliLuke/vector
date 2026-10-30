package instancemail

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	netmail "net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type sinkSession struct {
	messages chan string
	failure  Kind
}

func (s *sinkSession) Reset()                   {}
func (s *sinkSession) Logout() error            { return nil }
func (s *sinkSession) AuthMechanisms() []string { return []string{sasl.Plain} }
func (s *sinkSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, username, password string) error {
		if s.failure == KindAuthentication || username != "fixture" || password != "fixture-password" {
			return &smtp.SMTPError{Code: 535, Message: "authentication rejected"}
		}
		return nil
	}), nil
}
func (s *sinkSession) Mail(string, *smtp.MailOptions) error {
	if s.failure == KindSender {
		return &smtp.SMTPError{Code: 550, Message: "sender rejected"}
	}
	return nil
}
func (s *sinkSession) Rcpt(string, *smtp.RcptOptions) error {
	if s.failure == KindRecipients {
		return &smtp.SMTPError{Code: 550, Message: "recipient rejected"}
	}
	return nil
}
func (s *sinkSession) Data(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err == nil {
		s.messages <- string(data)
	}
	return err
}

func startSink(t *testing.T, failure Kind) (SMTPConfig, <-chan string) {
	t.Helper()
	messages := make(chan string, 1)
	server := smtp.NewServer(smtp.BackendFunc(func(*smtp.Conn) (smtp.Session, error) {
		return &sinkSession{messages: messages, failure: failure}, nil
	}))
	server.Domain = "localhost"
	server.AllowInsecureAuth = true
	server.ReadTimeout, server.WriteTimeout = 2*time.Second, 2*time.Second
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		testsupport.Close(t, server)
		select {
		case serveErr := <-done:
			if serveErr != nil {
				t.Errorf("SMTP fixture failed: %v", serveErr)
			}
		case <-time.After(2 * time.Second):
			t.Error("SMTP fixture did not stop after Close")
		}
	})
	return SMTPConfig{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, Username: "fixture", Password: "fixture-password"}, messages
}

func TestSMTPDeliveryAndErrorCategories(t *testing.T) {
	for _, failure := range []Kind{"", KindAuthentication, KindSender, KindRecipients} {
		t.Run(string(failure), func(t *testing.T) {
			cfg, messages := startSink(t, failure)
			message := Message{From: "sender@example.com", To: "receiver@example.com", Subject: "Email Notification from Plane", Body: "Sample notification 🎵", HTMLBody: "<p>Updated task</p>"}
			err := (Sender{Timeout: time.Second}).Send(context.Background(), cfg, message)
			if failure != "" {
				var categorized *Error
				if !errors.As(err, &categorized) || categorized.Kind != failure {
					t.Fatalf("expected %s, got %v", failure, err)
				}
				select {
				case <-messages:
					t.Fatal("failed send delivered a message")
				default:
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case data := <-messages:
				parsed, err := netmail.ReadMessage(strings.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				if parsed.Header.Get("Subject") != message.Subject {
					t.Fatalf("subject: %q", parsed.Header.Get("Subject"))
				}
				content, err := io.ReadAll(parsed.Body)
				if err != nil {
					t.Fatal(err)
				}
				rawMessage := string(content)
				if !strings.Contains(rawMessage, "Sample notification") || !strings.Contains(rawMessage, "Content-Type: text/html") || !strings.Contains(rawMessage, "Updated task") {
					t.Fatalf("message lacks plain or HTML alternative: %q", rawMessage)
				}
			case <-time.After(time.Second):
				t.Fatal("SMTP sink received no message")
			}
		})
	}
}

func TestHeaderInjectionRejectedBeforeDelivery(t *testing.T) {
	for _, message := range []Message{
		{From: "sender@example.com\r\nBcc: hidden@example.com", To: "receiver@example.com"},
		{From: "sender@example.com", To: "receiver@example.com\nBcc: hidden@example.com"},
		{From: "sender@example.com", To: "receiver@example.com", Subject: "subject\nInjected: yes"},
	} {
		err := (Sender{}).Send(context.Background(), SMTPConfig{}, message)
		var categorized *Error
		if !errors.As(err, &categorized) || categorized.Kind != KindHeader {
			t.Fatalf("expected header rejection, got %v", err)
		}
	}
}

func TestSMTPDeliveryPreservesAttachmentNameTypeAndBytes(t *testing.T) {
	cfg, messages := startSink(t, "")
	attachment := Attachment{Name: "slug-analytics.csv", ContentType: "text/csv", Data: []byte("Issue,Count\nABC-1,3\n")}
	message := Message{
		From: "sender@example.com", To: "receiver@example.com", Subject: "Your Export is ready",
		Body: "Your export is attached.", Attachments: []Attachment{attachment},
	}
	if err := (Sender{Timeout: time.Second}).Send(context.Background(), cfg, message); err != nil {
		t.Fatal(err)
	}
	select {
	case raw := <-messages:
		parsed, err := netmail.ReadMessage(strings.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		attachments, err := readAttachments(textproto.MIMEHeader(parsed.Header), parsed.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(attachments) != 1 {
			t.Fatalf("got %d attachments, want 1", len(attachments))
		}
		got := attachments[0]
		if got.name != attachment.Name || got.contentType != attachment.ContentType || string(got.data) != string(attachment.Data) {
			t.Fatalf("attachment mismatch: name=%q type=%q bytes=%q", got.name, got.contentType, got.data)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP sink received no message")
	}
}

type decodedAttachment struct {
	name, contentType string
	data              []byte
}

func readAttachments(header textproto.MIMEHeader, body io.Reader) ([]decodedAttachment, error) {
	contentType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(contentType, "multipart/") {
		reader := multipart.NewReader(body, params["boundary"])
		var attachments []decodedAttachment
		for {
			part, partErr := reader.NextPart()
			if partErr == io.EOF {
				return attachments, nil
			}
			if partErr != nil {
				return nil, partErr
			}
			nested, readErr := readAttachments(part.Header, part)
			if attachmentErr := errors.Join(readErr, part.Close()); attachmentErr != nil {
				return nil, attachmentErr
			}
			attachments = append(attachments, nested...)
		}
	}
	rawDisposition := header.Get("Content-Disposition")
	if rawDisposition == "" {
		return nil, nil
	}
	disposition, dispositionParams, err := mime.ParseMediaType(rawDisposition)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(disposition, "attachment") {
		return nil, nil
	}
	decoded := body
	switch strings.ToLower(header.Get("Content-Transfer-Encoding")) {
	case "base64":
		decoded = base64.NewDecoder(base64.StdEncoding, body)
	case "quoted-printable":
		decoded = quotedprintable.NewReader(body)
	}
	data, err := io.ReadAll(decoded)
	if err != nil {
		return nil, err
	}
	mediaType, _, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	return []decodedAttachment{{name: dispositionParams["filename"], contentType: mediaType, data: data}}, nil
}
