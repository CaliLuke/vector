//go:build integration

package issuequery

import (
	"context"
	"reflect"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestTypeFiltersMatchRowsAndCountsInPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		db, connectErr := pgx.Connect(ctx, url)
		if connectErr != nil {
			return connectErr
		}
		defer func() { _ = db.Close(context.WithoutCancel(ctx)) }()
		_, seedErr := db.Exec(ctx, `CREATE TABLE issues(row_number integer PRIMARY KEY,type_id uuid,priority text);
INSERT INTO issues VALUES(1,'`+selectedTypeA+`','low'),(2,'`+selectedTypeB+`','high'),(3,'00000000-0000-4000-8000-000000000003','low'),(4,NULL,'low')`)
		return seedErr
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pair.Close() }()
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close(context.WithoutCancel(ctx)) }()
	catalog := map[string]bool{selectedTypeA: true, selectedTypeB: true}
	for _, test := range []struct {
		name, filter string
		want         []int32
	}{
		{"one", `{"type_id__exact":"` + selectedTypeA + `"}`, []int32{1}},
		{"all selected", `{"type_id__in":["` + selectedTypeA + `","` + selectedTypeB + `"]}`, []int32{1, 2}},
		{"excludes and retains untyped", `{"nOt":{"type_id__in":"` + selectedTypeA + `,` + selectedTypeB + `"}}`, []int32{3, 4}},
		{"nested standard and type", `{"AND":[{"type_id__in":"` + selectedTypeA + `,` + selectedTypeB + `"},{"priority__exact":"high"}]}`, []int32{2}},
		{"untyped", `{"type_id__exact":null}`, []int32{4}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := []any{}
			predicate, buildErr := BuildComplexFilterWithTypes(test.filter, &args, catalog)
			if buildErr != nil {
				t.Fatal(buildErr)
			}
			var rows []int32
			var count int
			if queryErr := db.QueryRow(ctx, `SELECT array_agg(issue.row_number ORDER BY issue.row_number),count(*) FROM issues issue WHERE `+predicate, args...).Scan(&rows, &count); queryErr != nil {
				t.Fatal(queryErr)
			}
			if !reflect.DeepEqual(rows, test.want) || count != len(test.want) {
				t.Fatalf("rows/count=%v/%d want%v", rows, count, test.want)
			}
		})
	}
}
