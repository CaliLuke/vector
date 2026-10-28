//go:build integration

package issuequery

import (
	"context"
	"reflect"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const propertyItem1 = "10000000-0000-4000-8000-000000000001"
const propertyItem2 = "10000000-0000-4000-8000-000000000002"
const propertyItem5 = "10000000-0000-4000-8000-000000000005"
const selectedTypeDeleted = "10000000-0000-4000-8000-000000000099"

func propertyFilterDatabase(t *testing.T) (context.Context, *pgx.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		db, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, db, context.WithoutCancel(ctx)) }()
		_, err = db.Exec(ctx, `CREATE TABLE issues(id uuid PRIMARY KEY,row_number integer,priority text,project_id uuid,workspace_id uuid);
 CREATE TABLE labels(id uuid PRIMARY KEY,name text,project_id uuid,workspace_id uuid,deleted_at timestamptz);
 CREATE TABLE issue_labels(issue_id uuid,label_id uuid,deleted_at timestamptz);
 INSERT INTO issues SELECT ('10000000-0000-4000-8000-00000000000'||n)::uuid,n,CASE n WHEN 1 THEN 'high' WHEN 4 THEN 'high' WHEN 5 THEN NULL ELSE 'low' END,'10000000-0000-4000-8000-000000000010','10000000-0000-4000-8000-000000000020' FROM generate_series(1,5)n;
 INSERT INTO labels VALUES
  ('`+selectedTypeA+`','Zebra','10000000-0000-4000-8000-000000000010','10000000-0000-4000-8000-000000000020',NULL),
  ('`+selectedTypeB+`','Apple','10000000-0000-4000-8000-000000000010','10000000-0000-4000-8000-000000000020',NULL),
  ('`+selectedTypeDeleted+`','Deleted','10000000-0000-4000-8000-000000000010','10000000-0000-4000-8000-000000000020',now());
 INSERT INTO issue_labels SELECT id,'`+selectedTypeA+`'::uuid,CASE WHEN row_number=5 THEN now() END FROM issues WHERE row_number IN(1,3,5);
 INSERT INTO issue_labels SELECT id,'`+selectedTypeB+`'::uuid,NULL FROM issues WHERE row_number IN(1,2,3);
 INSERT INTO issue_labels SELECT id,'`+selectedTypeDeleted+`'::uuid,NULL FROM issues WHERE row_number=4;`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, db, context.WithoutCancel(ctx)) })
	return ctx, db
}

func bindTestPropertyFilter(t *testing.T, raw string) *BoundPropertyFilter {
	t.Helper()
	plan, err := PreparePropertyFilter(raw, nil, acceptTestProperty)
	if err != nil {
		t.Fatal(err)
	}
	matches := map[int]PropertyFilterMatches{}
	for _, leaf := range plan.Leaves() {
		matches[leaf.ID] = PropertyFilterMatches{TrueIDs: []string{propertyItem1}, FalseIDs: []string{propertyItem2, propertyItem5}}
	}
	bound, err := plan.Bind(matches)
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func TestPropertyFilterUnknownsAndLegacyRelationsInPostgres(t *testing.T) {
	ctx, db := propertyFilterDatabase(t)
	leaf := `{"property__effort__exact":4}`
	labelA := `{"label_id__exact":"` + selectedTypeA + `"}`
	labelB := `{"label_id__exact":"` + selectedTypeB + `"}`
	labelDeleted := `{"label_id__exact":"` + selectedTypeDeleted + `"}`
	for _, test := range []struct {
		name, raw string
		want      []int32
	}{
		{"positive", leaf, []int32{1}},
		{"negative excludes unknown", `{"not":` + leaf + `}`, []int32{2, 5}},
		{"double negative", `{"not":{"not":` + leaf + `}}`, []int32{1}},
		{"unknown or matching legacy", `{"or":[` + leaf + `,{"priority":"high"}]}`, []int32{1, 4}},
		{"unknown relation bindings", `{"not":{"and":[` + leaf + `,` + labelB + `]}}`, []int32{2, 4, 5}},
		{"unknown relation or", `{"not":{"or":[` + leaf + `,` + labelB + `]}}`, []int32{5}},
		{"relation or", `{"or":[` + leaf + `,` + labelA + `]}`, []int32{1, 3}},
		{"deleted parent relation is not selected", labelDeleted, nil},
		{"null label filter treats deleted parent and bridge as unlabeled", `{"label_id":null}`, []int32{4, 5}},
		{"positive relation aliases shared", `{"and":[` + leaf + `,` + labelA + `,` + labelB + `]}`, nil},
		{"false decides conjunction", `{"not":{"and":[` + leaf + `,` + labelA + `,` + labelB + `]}}`, []int32{1, 2, 3, 4, 5}},
		{"legacy null under negation", `{"or":[` + leaf + `,{"not":{"priority":"high"}}]}`, []int32{1, 2, 3, 5}},
	} {
		t.Run(test.name, func(t *testing.T) {
			bound := bindTestPropertyFilter(t, test.raw)
			args := []any{}
			predicate, err := bound.SQL(&args)
			if err != nil {
				t.Fatal(err)
			}
			var rows []int32
			var count int
			if err = db.QueryRow(ctx, `SELECT array_agg(issue.row_number ORDER BY issue.row_number),count(*) FROM issues issue WHERE `+predicate, args...).Scan(&rows, &count); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rows, test.want) || count != len(test.want) {
				t.Fatalf("got %v/%d want %v", rows, count, test.want)
			}
		})
	}
}

func TestPropertyFilterConstrainsRelationNameOrdering(t *testing.T) {
	ctx, db := propertyFilterDatabase(t)
	raw := `{"and":[{"label_id":"` + selectedTypeA + `"},{"or":[{"property__effort":4},{"priority":"low"}]}]}`
	bound := bindTestPropertyFilter(t, raw)
	args := []any{}
	predicate, err := bound.SQL(&args)
	if err != nil {
		t.Fatal(err)
	}
	order, err := Order(Options{OrderBy: "labels__name", Filters: raw, PropertyFilter: bound}, "", "", "", &args)
	if err != nil {
		t.Fatal(err)
	}
	// Remove only ORDER BY modifiers to inspect which relation's name controls it.
	expression := order[:len(order)-len(" ASC NULLS LAST")]
	var names []string
	if err = db.QueryRow(ctx, `SELECT array_agg(`+expression+` ORDER BY issue.row_number) FROM issues issue WHERE `+predicate, args...).Scan(&names); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"Zebra", "Zebra"}) {
		t.Fatalf("ordering used an unmatched relation: %v", names)
	}
}

func TestPropertyFilterDisjunctionCannotEscapeSurroundingScope(t *testing.T) {
	ctx, db := propertyFilterDatabase(t)
	bound := bindTestPropertyFilter(t, `{"or":[{"property__effort":4},{"priority":"high"}]}`)
	args := []any{}
	predicate, err := bound.SQL(&args)
	if err != nil {
		t.Fatal(err)
	}
	for _, where := range []string{
		"issue.row_number < 3 AND " + predicate,
		predicate + " AND issue.row_number > 2",
	} {
		var count int
		if err = db.QueryRow(ctx, "SELECT count(*) FROM issues issue WHERE "+where, args...).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("disjunction escaped surrounding scope: count=%d, where=%s", count, where)
		}
	}
}
