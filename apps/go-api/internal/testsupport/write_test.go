package testsupport_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

type fixtureWriteFailure struct{ err error }

func (writer fixtureWriteFailure) Write([]byte) (int, error) { return 0, writer.err }

func TestWritePreservesFixtureBytes(t *testing.T) {
	reporter := &cleanupReporter{}
	var output bytes.Buffer
	testsupport.Write(reporter, &output, []byte("fixture\x00body"))
	require.Empty(t, reporter.messages)
	require.Equal(t, "fixture\x00body", output.String())
}

func TestWriteReportsBrokenAndShortWrites(t *testing.T) {
	for _, writeErr := range []error{errors.New("connection reset"), nil} {
		reporter := &cleanupReporter{}
		testsupport.Write(reporter, fixtureWriteFailure{err: writeErr}, []byte("body"))
		require.Len(t, reporter.messages, 1)
		require.Contains(t, reporter.messages[0], "write fixture data")
	}
}

func TestWriteJSONPreservesEncodingAndNewline(t *testing.T) {
	reporter := &cleanupReporter{}
	var output bytes.Buffer
	testsupport.WriteJSON(reporter, &output, map[string]string{"value": "<fixture>"})
	require.Empty(t, reporter.messages)
	require.Equal(t, "{\"value\":\"\\u003cfixture\\u003e\"}\n", output.String())
}

func TestWriteJSONRejectsUnencodableFixtureBeforeWriting(t *testing.T) {
	reporter := &cleanupReporter{}
	var output bytes.Buffer
	testsupport.WriteJSON(reporter, &output, make(chan int))
	require.Len(t, reporter.messages, 1)
	require.Contains(t, reporter.messages[0], "encode fixture JSON")
	require.Empty(t, output.String())
}

func TestWriteJSONReportsResponseFailure(t *testing.T) {
	reporter := &cleanupReporter{}
	testsupport.WriteJSON(reporter, fixtureWriteFailure{err: errors.New("connection reset")}, true)
	require.Len(t, reporter.messages, 1)
	require.Contains(t, reporter.messages[0], "connection reset")
}
