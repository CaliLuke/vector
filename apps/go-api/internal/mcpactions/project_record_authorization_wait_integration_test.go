//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPProjectRecordWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, action := range []string{"update", "delete"} {
		t.Run(action, func(t *testing.T) {
			runProjectMutationAuthorizationWait(t, projectMutationWaitScenario{
				tool: "project", action: action, fixture: newMilestoneFixture,
				createFields: map[string]any{"name": "Project wait acceptance", "identifier": "WAIT"},
				snapshot:     projectRecordAuthorizationSnapshot,
			})
		})
	}
}

func projectRecordAuthorizationSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(projectMutationStorageSnapshot(t, f), &data))
	// The test changes membership to revoke access; domain data and effects must
	// remain unchanged after that intentional authorization change.
	delete(data, "members")
	snapshot, err := json.Marshal(data)
	require.NoError(t, err)
	return snapshot
}
