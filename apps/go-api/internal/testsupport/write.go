package testsupport

import (
	"encoding/json"
	"io"
)

// Write reports failed or incomplete fixture writes without calling Fatal from
// an HTTP handler goroutine. The caller owns the writer and its lifetime.
func Write(t errorReporter, writer io.Writer, body []byte) {
	t.Helper()
	written, err := writer.Write(body)
	if err == nil && written != len(body) {
		err = io.ErrShortWrite
	}
	if err != nil {
		t.Errorf("write fixture data: %v", err)
	}
}

// WriteJSON writes a JSON fixture response with the encoder's trailing newline.
// Encoding and write failures fail the owning test.
func WriteJSON(t errorReporter, writer io.Writer, value any) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Errorf("encode fixture JSON: %v", err)
		return
	}
	Write(t, writer, append(body, '\n'))
}
