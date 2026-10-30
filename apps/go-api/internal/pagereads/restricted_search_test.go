package pagereads

import (
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

func TestRestrictedPageSearchDoesNotReadOrRevealItsName(t *testing.T) {
	for _, sample := range []struct {
		name, search string
		matches      bool
	}{
		{name: "empty"},
		{name: "matching", search: "Secret", matches: true},
		{name: "nonmatching", search: "missing"},
		{name: "multiple terms", search: "Secret title", matches: true},
		{name: "literal wildcards", search: `%_`},
	} {
		t.Run(sample.name, func(t *testing.T) {
			db := &databaseStub{rows: []pgx.Row{stringRow("restricted_guest"), boolRow(sample.matches)}}
			visits := &recentVisitCapture{}
			repository, err := NewRepository(db, visits)
			if err != nil {
				t.Fatal(err)
			}
			data, err := repository.Detail(t.Context(), uuid.New(), "demo", uuid.New(), uuid.New(), sample.search, "-created_at", "UTC")
			if !errors.Is(err, ErrRestrictedGuest) || data != nil {
				t.Fatalf("denied search: data=%s err=%v", data, err)
			}
			if len(db.queries) != 1 {
				t.Fatalf("denied page performed %d queries, want authorization only", len(db.queries))
			}
			if len(visits.args) != 0 {
				t.Fatal("denied search recorded a recent visit")
			}
		})
	}
}
