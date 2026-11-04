//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPReleaseLabelReplayRechecksSelectedLabels(t *testing.T) {
	runCatalogueCredentials(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range []string{"attach", "detach"} {
			input := releaseWriteInput(t, f, call, releaseMutationCase{"release_label", action})
			accepted := milestoneObject(t, call("release_label", input))
			require.Equal(t, false, accepted["replayed"])
			_, err := f.pool.Exec(t.Context(), `UPDATE release_labels SET deleted_at=now() WHERE id=$1`, input["label_ids"])
			require.NoError(t, err)
			before := releaseSnapshot(t, f)
			milestoneFailure(t, call("release_label", input), "not_found")
			require.Equal(t, before, releaseSnapshot(t, f))
			_, err = f.pool.Exec(t.Context(), `UPDATE release_labels SET deleted_at=NULL WHERE id=$1`, input["label_ids"])
			require.NoError(t, err)
			accepted["replayed"] = true
			require.Equal(t, accepted, milestoneObject(t, call("release_label", input)))
		}
	})
}
