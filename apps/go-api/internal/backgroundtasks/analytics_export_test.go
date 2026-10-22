package backgroundtasks

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestAnalyticsCSVRowUsesQuoteAllAndCRLF(t *testing.T) {
	var output strings.Builder
	row := []string{"comma,value", `quote "value"`, "line one\nline two", "Zoë\r\n東京"}
	if err := writeAnalyticsCSVRow(&output, row); err != nil {
		t.Fatalf("write analytics CSV row: %v", err)
	}
	want := "\"comma,value\",\"quote \"\"value\"\"\",\"line one\nline two\",\"Zoë\r\n東京\"\r\n"
	if got := output.String(); got != want {
		t.Fatalf("analytics CSV row = %q, want %q", got, want)
	}
}

func TestAnalyticsCSVRowPropagatesWriterErrors(t *testing.T) {
	writeErr := errors.New("attachment writer failed")
	err := writeAnalyticsCSVRow(failingAnalyticsCSVWriter{err: writeErr}, []string{"value"})
	if !errors.Is(err, writeErr) {
		t.Fatalf("write error = %v, want wrapped %v", err, writeErr)
	}
}

type failingAnalyticsCSVWriter struct{ err error }

func (writer failingAnalyticsCSVWriter) Write([]byte) (int, error) { return 0, writer.err }

var _ io.Writer = failingAnalyticsCSVWriter{}
