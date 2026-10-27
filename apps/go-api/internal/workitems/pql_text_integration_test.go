//go:build integration

package workitems

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIPQLTextSearchUsesTitleAndMainDescriptionOnly(t *testing.T) {
	f := newPQLIdentityFixture(t)
	f.examples(t, map[string][]int{
		`text ~ "body NEEDLE"`: {1}, `title ~ "body needle"`: {},
		`text ~ "Only body"`: {2}, `text = "Only body"`: {2}, `text != "Only body"`: {1, 3},
		`text ~ "Ordinary"`: {3}, `text ~ "private body"`: {},
		`text ~ "50%_\\"`: {1}, `text ~ "%"`: {1},
		`text IN ("Only body","Ordinary")`: {2, 3}, `text NOT IN ("Only body","Ordinary")`: {1},
		`text IS NULL`: {4}, `text IS NOT NULL`: {1, 2, 3}, `text IS EMPTY`: {}, `text IS NOT EMPTY`: {1, 2, 3},
		`text = "Literal 50%_\\\\ title\nBody needle 50%_\\\\"`: {1},
	})
	_, err := f.db.Exec(f.ctx, `UPDATE issues SET description_stripped=NULL WHERE sequence_id=2; UPDATE issues SET description_stripped='Body-only' WHERE sequence_id=4`)
	require.NoError(t, err)
	f.examples(t, map[string][]int{`text IS EMPTY`: {2}, `text = "Body-only"`: {4}, `text IS NULL`: {}, `text IS NOT EMPTY`: {1, 3, 4}})
}
