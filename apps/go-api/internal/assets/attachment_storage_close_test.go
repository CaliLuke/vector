package assets

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type attachmentTransportFunc func(*http.Request) (*http.Response, error)

func (f attachmentTransportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type attachmentCloseBody struct {
	io.Reader
	closeErr error
	closed   int
}

func (b *attachmentCloseBody) Close() error { b.closed++; return b.closeErr }

func TestAttachmentStorageReadPreservesReadLimitAndCloseErrors(t *testing.T) {
	closeFailure := errors.New("injected attachment close failure")
	for _, tc := range []struct {
		name     string
		content  string
		declared int64
		readErr  error
		closeErr error
		limited  bool
	}{
		{name: "success", content: "body", declared: 4},
		{name: "successful read failed close", content: "body", declared: 4, closeErr: closeFailure},
		{name: "read and close failures", content: "part", declared: -1, readErr: io.ErrUnexpectedEOF, closeErr: closeFailure},
		{name: "read failure", content: "part", declared: -1, readErr: io.ErrUnexpectedEOF},
		{name: "declared size and close failure", content: "too large", declared: 9, closeErr: closeFailure, limited: true},
		{name: "streamed size and close failure", content: "too large", declared: -1, closeErr: closeFailure, limited: true},
		{name: "size limit", content: "too large", declared: 9, limited: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reader io.Reader = strings.NewReader(tc.content)
			if tc.readErr != nil {
				reader = io.MultiReader(reader, iotest.ErrReader(tc.readErr))
			}
			body := &attachmentCloseBody{Reader: reader, closeErr: tc.closeErr}
			transport := attachmentTransportFunc(func(request *http.Request) (*http.Response, error) {
				headers := http.Header{"Content-Type": {"text/plain"}}
				if tc.declared >= 0 {
					headers.Set("Content-Length", strconv.FormatInt(tc.declared, 10))
				}
				return &http.Response{StatusCode: http.StatusOK, Header: headers, Body: body, ContentLength: tc.declared, Request: request}, nil
			})
			client := s3.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("fixture", "fixture", ""), HTTPClient: &http.Client{Transport: transport}}, func(options *s3.Options) {
				options.BaseEndpoint = aws.String("http://storage.example.test")
				options.UsePathStyle = true
				options.RetryMaxAttempts = 1
			})
			storage := &Storage{client: client, bucket: "attachments"}
			content, err := storage.readAttachment(t.Context(), "example.txt", 5)
			require.Equal(t, 1, body.closed)
			if tc.closeErr != nil {
				require.ErrorIs(t, err, tc.closeErr)
			}
			if tc.readErr != nil {
				require.ErrorIs(t, err, tc.readErr)
			}
			if tc.limited {
				var rejected *djangoapi.HTTPError
				require.ErrorAs(t, err, &rejected)
				require.Equal(t, http.StatusBadRequest, rejected.Status)
				require.Nil(t, content)
			}
			if tc.closeErr == nil && tc.readErr == nil && !tc.limited {
				require.NoError(t, err)
				require.Equal(t, tc.content, string(content))
			}
		})
	}
}
