//go:build integration

package mcpactions_test

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerReceiptsRetainAcceptedTimezone(t *testing.T) {
	for _, scenario := range []customerMutationScenario{
		{"customer", "create"}, {"customer", "upsert"}, {"customer", "update"},
		{"customer_request", "create"}, {"customer_request", "update"},
		{"customer_property", "create"}, {"customer_property", "update"},
	} {
		t.Run(scenario.tool+"/"+scenario.action, func(t *testing.T) {
			runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
				input := customerMutationWriteInput(t, f, call, scenario)
				_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata' WHERE id=$1`, testsupport.LabelActor)
				require.NoError(t, err)
				accepted := milestoneObject(t, call(scenario.tool, input))
				require.Equal(t, false, accepted["replayed"])
				assertCustomerRecordTimezone(t, accepted["result"].(map[string]any))
				before := customerStorageSnapshot(t, f)
				_, err = f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='UTC' WHERE id=$1`, testsupport.LabelActor)
				require.NoError(t, err)
				accepted["replayed"] = true
				require.Equal(t, accepted, milestoneObject(t, call(scenario.tool, input)), "receipt replay must retain the accepted JSON representation after a timezone preference change")
				require.Equal(t, before, customerStorageSnapshot(t, f))
			})
		})
	}
}

func assertCustomerRecordTimezone(t *testing.T, record map[string]any) {
	t.Helper()
	require.NotNil(t, record["created_at"])
	for _, field := range []string{"created_at", "updated_at"} {
		if raw, ok := record[field].(string); ok {
			stamp, err := time.Parse(time.RFC3339Nano, raw)
			require.NoError(t, err)
			_, offset := stamp.Zone()
			require.Equal(t, 19800, offset, field)
		}
	}
	if options, ok := record["options"].([]any); ok {
		for _, option := range options {
			assertCustomerRecordTimezone(t, option.(map[string]any))
		}
	}
}
