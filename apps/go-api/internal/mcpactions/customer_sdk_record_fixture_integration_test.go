//go:build integration

package mcpactions_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Exact field sets come from the verified plane-sdk 0.3.1 source archive.
// Its checksum and parity target are recorded in docs/mcp-parity.md.
var customerSDKRecordFields = map[string][]string{
	"Customer":               {"archived_at", "contract_status", "created_at", "created_by", "customer_request_count", "deleted_at", "description", "description_binary", "description_html", "description_stripped", "domain", "email", "employees", "external_id", "external_source", "id", "logo_asset", "logo_props", "logo_url", "name", "revenue", "stage", "updated_at", "updated_by", "website_url", "workspace"},
	"CustomerPropertyOption": {"created_at", "created_by", "deleted_at", "description", "external_id", "external_source", "id", "is_active", "is_default", "logo_props", "name", "parent", "property", "sort_order", "updated_at", "updated_by", "workspace"},
	"CustomerProperty":       {"created_at", "created_by", "default_value", "deleted_at", "description", "display_name", "external_id", "external_source", "id", "is_active", "is_multi", "is_required", "logo_props", "name", "options", "property_type", "relation_type", "settings", "sort_order", "updated_at", "updated_by", "validation_rules", "workspace"},
}

func assertCustomerSDKMetadata(t *testing.T, kind string, record map[string]any) {
	t.Helper()
	require.Equal(t, customerSDKRecordFields[kind], slices.Sorted(maps.Keys(record)))
	_, err := uuid.Parse(record["id"].(string))
	require.NoError(t, err)
	for _, field := range []string{"created_at", "updated_at"} {
		_, err = time.Parse(time.RFC3339Nano, record[field].(string))
		require.NoError(t, err, field)
	}
	require.Equal(t, testsupport.LabelWorkspace, record["workspace"])
	require.Equal(t, testsupport.LabelActor, record["created_by"])
	require.Equal(t, testsupport.LabelActor, record["updated_by"])
	require.Nil(t, record["deleted_at"])
	require.Equal(t, map[string]any{}, record["logo_props"])
}
