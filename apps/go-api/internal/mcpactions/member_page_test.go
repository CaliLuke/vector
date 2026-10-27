package mcpactions

import (
	"testing"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemberPageProjectionRejectsUnserializableOrIncompatibleData(t *testing.T) {
	// A projection must fail closed if its application and result types diverge.
	// Neither the supplied row nor a codec error may escape as tool content.
	_, err := encodeMemberPage[api.MemberDirectoryPage]([]chan string{make(chan string)}, 1, 20, 0)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "chan")
	_, err = encodeMemberPage[api.MemberDirectoryPage]([]string{"private row"}, 1, 20, 0)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "private row")
}
