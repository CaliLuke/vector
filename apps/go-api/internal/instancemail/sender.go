// Package instancemail adapts the mail library to instance credential checks.
package instancemail

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net"
	"net/textproto"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	mail "github.com/wneessen/go-mail"
)

type SMTPConfig struct {
	Host               string
	Port               int
	Username, Password string
	UseTLS, UseSSL     bool
}

type Attachment struct {
	Name        string
	ContentType string
	Data        []byte
}

type Message struct {
	From, To, Subject, Body, HTMLBody string
	Attachments                       []Attachment
}

type Kind string

const (
	KindHeader         Kind = "header"
	KindAuthentication Kind = "authentication"
	KindConnect        Kind = "connect"
	KindSender         Kind = "sender"
	KindDisconnected   Kind = "disconnected"
	KindRecipients     Kind = "recipients"
	KindTimeout        Kind = "timeout"
	KindNetwork        Kind = "network"
	KindConfiguration  Kind = "configuration"
)

type Error struct {
	Kind Kind
	Err  error
}

// Error deliberately excludes SMTP responses, which may contain addresses or
// credentials. The domain maps Kind to its stable public response message.
func (e *Error) Error() string { return "instance email check failed: " + string(e.Kind) }
func (e *Error) Unwrap() error { return e.Err }

type Sender struct{ Timeout time.Duration }

func (s Sender) Send(ctx context.Context, cfg SMTPConfig, message Message) error {
	msg, err := buildMessage(message)
	if err != nil {
		return err
	}
	client, err := s.client(cfg)
	if err != nil {
		return &Error{KindConfiguration, err}
	}
	observed := &responseFailure{}
	client.ErrorHandlerRegistry.SetDefaultHandler(observed)
	if err := client.DialWithContext(ctx); err != nil {
		return classify(err, observed, true)
	}
	// An EHLO failure can be recovered by HELO. Only failures in the send phase
	// should influence the delivery classification after a successful dial.
	*observed = responseFailure{}
	sendErr := client.Send(msg)
	classified := classify(sendErr, observed, false)
	closeErr := client.Close()
	if classified != nil {
		return classified
	}
	// Django's SMTP backend tolerates a server that disconnects during QUIT.
	if errors.Is(closeErr, io.EOF) || errors.Is(closeErr, io.ErrUnexpectedEOF) {
		return nil
	}
	return classify(closeErr, nil, false)
}

func buildMessage(message Message) (*mail.Msg, error) {
	for _, value := range []string{message.From, message.To, message.Subject} {
		if strings.ContainsAny(value, "\r\n") {
			return nil, &Error{KindHeader, errors.New("newline in email header")}
		}
	}
	msg := mail.NewMsg()
	if err := msg.From(message.From); err != nil {
		return nil, &Error{KindConfiguration, err}
	}
	if err := msg.To(message.To); err != nil {
		return nil, &Error{KindConfiguration, err}
	}
	msg.Subject(message.Subject)
	msg.SetBodyString(mail.TypeTextPlain, message.Body)
	if message.HTMLBody != "" {
		msg.AddAlternativeString(mail.TypeTextHTML, message.HTMLBody)
	}
	for _, attachment := range message.Attachments {
		if attachment.Name == "" || filepath.Base(attachment.Name) != attachment.Name || strings.ContainsAny(attachment.Name, "\r\n\\") {
			return nil, &Error{KindHeader, errors.New("invalid attachment filename")}
		}
		options := make([]mail.FileOption, 0, 1)
		if attachment.ContentType != "" {
			mediaType, _, err := mime.ParseMediaType(attachment.ContentType)
			if err != nil || strings.ContainsAny(attachment.ContentType, "\r\n") || mediaType == "" {
				return nil, &Error{KindHeader, errors.New("invalid attachment content type")}
			}
			options = append(options, mail.WithFileContentType(mail.ContentType(attachment.ContentType)))
		}
		if err := msg.AttachReader(attachment.Name, bytes.NewReader(attachment.Data), options...); err != nil {
			return nil, &Error{KindConfiguration, err}
		}
	}
	return msg, nil
}

func (s Sender) client(cfg SMTPConfig) (*mail.Client, error) {
	if cfg.UseTLS && cfg.UseSSL {
		return nil, errors.New("TLS and SSL are mutually exclusive")
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = mail.DefaultTimeout
	}
	policy := mail.NoTLS
	if cfg.UseTLS {
		policy = mail.TLSMandatory
	}
	opts := []mail.Option{mail.WithPort(cfg.Port), mail.WithTimeout(timeout), mail.WithTLSPolicy(policy)}
	if cfg.UseSSL {
		opts = append(opts, mail.WithSSL())
	}
	if cfg.Username != "" && cfg.Password != "" {
		// Preserve Django's preferred mechanisms and explicit unencrypted SMTP
		// configuration. The library handles negotiation and authentication.
		opts = append(opts, mail.WithUsername(cfg.Username), mail.WithPassword(cfg.Password),
			mail.WithOpportunisticSMTPAuth(mail.SMTPAuthCramMD5, mail.SMTPAuthPlainNoEnc, mail.SMTPAuthLoginNoEnc))
	}
	return mail.NewClient(cfg.Host, opts...)
}

// The library exposes the command and original response error through this
// callback. Retaining them for one send avoids parsing human error strings and
// preserves network causes when SendError does not unwrap its internal errors.
type responseFailure struct {
	command string
	err     error
}

func (f *responseFailure) HandleError(_ string, command string, _ *textproto.Conn, err error) error {
	if f.err == nil || (command != "rset" && command != "quit") {
		f.command, f.err = command, err
	}
	return err
}

func classify(err error, observed *responseFailure, dialing bool) error {
	if err == nil {
		return nil
	}
	cause := err
	if observed != nil && observed.err != nil {
		cause = observed.err
	}
	kind := KindConfiguration
	var networkError net.Error
	var response *textproto.Error
	var delivery *mail.SendError
	switch {
	case errors.Is(cause, context.DeadlineExceeded), errors.As(cause, &networkError) && networkError.Timeout():
		kind = KindTimeout
	case errors.Is(cause, io.EOF), errors.Is(cause, io.ErrUnexpectedEOF):
		kind = KindDisconnected
	case errors.Is(cause, syscall.ECONNREFUSED), errors.Is(cause, syscall.ECONNRESET), errors.Is(cause, syscall.ECONNABORTED), errors.Is(cause, syscall.EPIPE):
		kind = KindNetwork
	case errors.As(err, &delivery):
		switch delivery.Reason {
		case mail.ErrSMTPMailFrom:
			kind = KindSender
		case mail.ErrSMTPRcptTo:
			kind = KindRecipients
		}
	case errors.As(cause, &response) && dialing:
		// AUTH reads arbitrary SMTP response codes before the auth implementation
		// rejects them, so the response callback may not run. RFC 4954 defines
		// these authentication-specific status codes independently of that hook.
		if response.Code == 432 || response.Code == 454 || (response.Code >= 534 && response.Code <= 538) ||
			(observed != nil && (observed.command == "auth" || observed.command == "%s")) {
			kind = KindAuthentication
		} else {
			kind = KindConnect
		}
	}
	return &Error{kind, err}
}
