package pagereads

import (
	"math"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPageCommandFieldsRejectInvalidTypesAndValues(t *testing.T) {
	for _, test := range []struct {
		name, tool, action, field string
		value                     any
	}{
		{"page access rejects member-only level", "page", "create", "access", 2},
		{"access rejects numeric coercion", "page", "create", "access", float64(1)},
		{"member access rejects negative level", "collection", "add_member", "member_access", -1},
		{"boolean rejects string coercion", "page", "archive", "archive", "true"},
		{"sort order rejects integer coercion", "collection", "update", "sort_order", 1},
		{"sort order rejects NaN", "collection", "update", "sort_order", math.NaN()},
		{"sort order rejects infinity", "collection", "update", "sort_order", math.Inf(1)},
		{"name rejects null", "page", "update", "name", nil},
		{"name rejects whitespace", "page", "update", "name", " \t "},
		{"text rejects nul", "page", "update", "name", "a\x00b"},
		{"name limit counts runes", "page", "update", "name", strings.Repeat("界", 256)},
		{"update rejects creation-only field", "page", "update", "access", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			in := PageAPICommand{
				PageAPIQuery: PageAPIQuery{Tool: test.tool, Action: test.action, PageID: uuid.New(), CollectionID: uuid.New(), UserID: uuid.New()},
				Fields:       map[string]any{test.field: test.value},
			}
			if test.action == "create" {
				in.Fields["name"] = "Page"
				in.Fields["description_html"] = "<p>Body</p>"
			}
			require.ErrorIs(t, normalizePageAPI(&in), projectmutations.ErrInvalid)
		})
	}
}

func TestPageCommandSanitizesOwnedFieldsWithoutChangingCallerInput(t *testing.T) {
	name := strings.Repeat("界", 255)
	body := "<p onclick=\"alert(1)\">" + strings.Repeat("body ", 60) + "</p>"
	fields := map[string]any{"name": name, "description_html": body}
	in := PageAPICommand{PageAPIQuery: PageAPIQuery{Tool: "page", Action: "update", PageID: uuid.New()}, Fields: fields}
	require.NoError(t, normalizePageAPI(&in))
	require.Equal(t, name, in.Fields["name"])
	require.Equal(t, "<p>"+strings.Repeat("body ", 60)+"</p>", in.Fields["description_html"])
	require.Equal(t, body, fields["description_html"])
	in.Fields["name"] = "Changed"
	require.Equal(t, name, fields["name"])
}
