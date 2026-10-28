package releases

import (
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
)

func TestMutateRejectsInvalidReleaseCommandsBeforeDatabaseAccess(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	for _, test := range []struct {
		name  string
		input Command
	}{
		{"missing update identity", Command{Query: Query{Tool: "release", Action: "update"}, Fields: map[string]any{"name": "Release"}}},
		{"unknown field", Command{Query: Query{Tool: "release", Action: "create"}, Fields: map[string]any{"name": "Release", "admin": true}}},
		{"invalid date", Command{Query: Query{Tool: "release", Action: "create"}, Fields: map[string]any{"name": "Release", "target_date": "2026-02-30"}}},
		{"blank tag version", Command{Query: Query{Tool: "release_tag", Action: "create"}, Fields: map[string]any{"version": " \t"}}},
		{"nil tag", Command{Query: Query{Tool: "release", Action: "create"}, Fields: map[string]any{"name": "Release", "tag_id": uuid.Nil.String()}}},
		{"invalid label color", Command{Query: Query{Tool: "release_label", Action: "create"}, Fields: map[string]any{"name": "Release", "color": "red"}}},
		{"empty changelog", Command{Query: Query{Tool: "release", Action: "update_changelog", ID: id}}},
		{"attachment without release", Command{Query: Query{Tool: "release_label", Action: "attach"}, Add: []uuid.UUID{id}}},
		{"nil association", Command{Query: Query{Tool: "release", Action: "manage_workitems", ID: id}, Add: []uuid.UUID{uuid.Nil}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewRepository(nil).Mutate(t.Context(), test.input)
			if !errors.Is(err, projectmutations.ErrInvalid) {
				t.Fatalf("Mutate error = %v, want invalid command before receipt/database access", err)
			}
		})
	}
}
