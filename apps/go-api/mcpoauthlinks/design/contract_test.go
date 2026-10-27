package design

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestAccountLinkingRequiresSessionExceptProviderCallback(t *testing.T) {
	require.NoError(t, expr.RegisterDefaultRoots())
	require.NoError(t, eval.RunDSL())
	service := expr.Root.Service("mcp_oauth_links")
	require.NotNil(t, service)
	require.Len(t, service.Methods, 5)
	for _, name := range []string{"Status", "Start", "Confirm", "Disconnect"} {
		method := service.Method(name)
		require.NotNil(t, method, name)
		require.Len(t, method.SessionAuths, 1, name)
		require.NotEmpty(t, method.EffectiveRequirements(), name)
	}
	callback := service.Method("Callback")
	require.Empty(t, callback.EffectiveRequirements())
	require.Equal(t, []string{"state"}, callback.Payload.Validation.Required)
	require.Equal(t, expr.String, callback.Payload.Find("code").Type)
	require.Equal(t, expr.String, callback.Payload.Find("state").Type)
	require.EqualValues(t, expr.FormatUUID, service.Method("Confirm").Payload.Find("id").Validation.Format)
	require.True(t, status.Attribute().Find("subject").Nullable)
	require.True(t, status.Attribute().Find("pending").Nullable)
}
