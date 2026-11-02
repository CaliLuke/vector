package projectmembershipreads

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectMemberInputUsesCanonicalIdentityAndRoleChoices(t *testing.T) {
	const id = "10000000-0000-4000-8000-00000000000a"
	command := LifecycleCommand{Operation: "legacy-create-members", Method: http.MethodPost, ContentType: "application/json", Body: strings.NewReader(`{"members":[{"member_id":"` + strings.ToUpper(id) + `","role":20.0}]}`)}
	inputs, err := parseProjectMembers(command)
	require.NoError(t, err)
	require.Len(t, inputs, 1)
	require.Equal(t, id, inputs[0].member)
	require.Equal(t, 20, *inputs[0].role)
	for _, body := range []string{
		`null`, `[]`, `{"members":[]}`,
		`{"members":[{"member_id":true}]}`,
		`{"members":[{"member_id":"` + id + `","role":true}]}`,
		`{"members":[{"member_id":"` + id + `","role":15.5}]}`,
		`{"members":[{"member_id":"` + id + `","role":null}]}`,
		`{"members":[{"member_id":"` + id + `","created_by":"` + id + `"}]}`,
		`{"members":[{"member_id":"` + id + `"},{"member_id":"` + strings.ToUpper(id) + `"}]}`,
	} {
		command.Body = strings.NewReader(body)
		_, err = parseProjectMembers(command)
		require.Error(t, err, body)
	}
}
