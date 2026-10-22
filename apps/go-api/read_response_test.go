package systemapi

import (
	"io"
	"testing"
)

type capturedRawResponse struct {
	metadata any
	body     io.ReadCloser
	err      error
}

func captureRawResponse(metadata any, body io.ReadCloser, err error) capturedRawResponse {
	return capturedRawResponse{metadata: metadata, body: body, err: err}
}

func readRawResponse(t *testing.T, response capturedRawResponse) string {
	t.Helper()
	if response.metadata == nil {
		t.Fatal("raw response metadata is nil")
	}
	return readRawBody(t, response.body, response.err)
}

func readRawBody(t *testing.T, body io.ReadCloser, err error) string {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if body == nil {
		t.Fatal("raw response body is nil")
	}
	data, readErr := io.ReadAll(body)
	closeErr := body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return string(data)
}

func TestRawJSONReadCloserEscapesJavaScriptLineSeparators(t *testing.T) {
	body := readRawBody(t, rawJSONReadCloser([]byte("{\"value\":\"before\u2028middle\u2029after\"}")), nil)
	want := `{"value":"before\u2028middle\u2029after"}`
	if body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}
