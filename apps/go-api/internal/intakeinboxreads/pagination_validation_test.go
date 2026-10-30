package intakeinboxreads

import (
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

func TestMalformedPaginationReturnsClientErrors(t *testing.T) {
	for _, test := range []struct {
		name, limit, cursor string
		want                error
	}{
		{"zero limit", "0", "", ErrInvalidPerPage},
		{"negative limit", "-1", "", ErrInvalidPerPage},
		{"huge negative limit", "-999999999999999999999", "", ErrInvalidPerPage},
		{"unrepresentable page", "1000", "1000:999999999999999999999:0", ErrInvalidCursor},
		{"overflowing offset", "1000", "1000:9223372036854776:0", ErrInvalidCursor},
		{"overflowing page end", "1", "1:9223372036854775807:0", ErrInvalidCursor},
		{"mismatched previous limit", "1000", "10:2:1", ErrInvalidCursor},
		{"nonfinite cursor", "1000", "1.e999:0:0", ErrInvalidCursor},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseOffsetPagination(test.limit, true, test.cursor, test.cursor != "")
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
		})
	}
}

func TestStatusFiltersRejectValuesOutsideDatabaseIntegerRange(t *testing.T) {
	for _, value := range []string{"invalid", "2147483648", "-2147483649", "999999999999999999999"} {
		for _, filter := range []string{"status", "intake_status", "inbox_status"} {
			t.Run(filter+"/"+value, func(t *testing.T) {
				builder := newWhereBuilder()
				var err error
				if filter == "status" {
					err = addIntakeStatuses(builder, value, true)
				} else {
					err = addIssueFilters(builder, "issue", map[string]string{filter: value}, nil, false, "UTC")
				}
				if !errors.Is(err, ErrBadRequest) {
					t.Fatalf("error=%v want=%v", err, ErrBadRequest)
				}
			})
		}
	}
}

func TestMalformedListParametersStopAfterAuthorization(t *testing.T) {
	for _, draft := range []bool{false, true} {
		name := "intake"
		if draft {
			name = "draft"
		}
		for _, test := range []struct {
			name    string
			options ListOptions
			want    error
		}{
			{"limit", ListOptions{PerPage: "0", PerPageSet: true}, ErrInvalidPerPage},
			{"cursor", ListOptions{Cursor: "1000:999999999999999999999:0", CursorSet: true}, ErrInvalidCursor},
		} {
			t.Run(name+"/"+test.name, func(t *testing.T) {
				database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true)}}
				queries := 1
				if !draft {
					database.rows = append(database.rows, repositoryIntakeContextRow(uuid.New(), false))
					queries = 2
				}
				repository, err := NewRepository(database)
				if err != nil {
					t.Fatal(err)
				}
				if draft {
					_, err = repository.DraftIssueList(t.Context(), uuid.New(), "demo", test.options)
				} else {
					_, err = repository.IntakeIssueList(t.Context(), uuid.New(), "demo", uuid.New(), test.options)
				}
				if !errors.Is(err, test.want) {
					t.Fatalf("error=%v want=%v", err, test.want)
				}
				if len(database.queries) != queries {
					t.Fatalf("data query ran for invalid pagination: %d queries", len(database.queries))
				}
			})
		}
	}
}
