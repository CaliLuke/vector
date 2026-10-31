package notionsync

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestJournalLoadFailureReleasesLockForRecovery(t *testing.T) {
	for _, contents := range []string{`{`, `{"version":2,"scope":"scope","entries":{}}`, `{"version":1,"scope":"other","entries":{}}`} {
		t.Run(contents, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
			failed, err := OpenJournal(path, "scope")
			require.Error(t, err)
			require.Nil(t, failed)
			require.NoError(t, os.WriteFile(path, []byte(`{"version":1,"scope":"scope","entries":{}}`), 0600))
			recovered, err := OpenJournal(path, "scope")
			require.NoError(t, err, "failed load must release its lock")
			testsupport.Close(t, recovered)
		})
	}
}

func TestJournalFailedReplacementRemovesTemporaryFileAndRetainsPriorState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	journal, err := OpenJournal(path, "scope")
	require.NoError(t, err)
	t.Cleanup(func() { testsupport.Close(t, journal) })
	journal.Entries["project:one"] = &Entry{ID: "destination", Base: map[string]any{"name": "Accepted"}}
	require.NoError(t, journal.Save())
	original, err := os.ReadFile(path)
	require.NoError(t, err)

	// Keep the original journal, but make its target a directory to force rename
	// failure after the temporary file has been written, synced and closed.
	saved := path + ".saved"
	require.NoError(t, os.Rename(path, saved))
	require.NoError(t, os.Mkdir(path, 0700))
	journal.Entries["project:one"].Base["name"] = "Next"
	err = journal.Save()
	require.Error(t, err)
	var pathErr *os.LinkError
	require.ErrorAs(t, err, &pathErr)
	require.Equal(t, "rename", pathErr.Op)
	retained, err := os.ReadFile(saved)
	require.NoError(t, err)
	require.Equal(t, original, retained)
	temps, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".notion-sync-*"))
	require.NoError(t, err)
	require.Empty(t, temps)
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Rename(saved, path))
	require.NoError(t, journal.Save(), "the same journal remains usable after filesystem recovery")
	accepted, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotEqual(t, original, accepted)
}

func TestJournalReadFailurePreservesCauseAndReleasesLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.Mkdir(path, 0700))
	_, err := OpenJournal(path, "scope")
	var failure *os.PathError
	require.ErrorAs(t, err, &failure)
	require.True(t, errors.Is(err, failure.Err))
	require.NoError(t, os.Remove(path))
	recovered, err := OpenJournal(path, "scope")
	require.NoError(t, err)
	testsupport.Close(t, recovered)
}
