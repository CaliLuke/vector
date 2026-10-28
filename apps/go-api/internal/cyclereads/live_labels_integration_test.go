//go:build integration

package cyclereads

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const liveLabelSeed = `
INSERT INTO issues(id,name,workspace_id,project_id,priority,created_at,updated_at)
 SELECT ('00000000-0000-4000-8000-'||lpad(n::text,12,'0'))::uuid,'case-'||n,
 '00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000002','low',now()-n*interval '1 second',now()
 FROM generate_series(7,11) n;
UPDATE issues SET priority='low';
INSERT INTO cycle_issues(issue_id,cycle_id,created_at)
 SELECT id,'00000000-0000-4000-8000-000000000003',now() FROM issues WHERE name LIKE 'case-%';
INSERT INTO labels(id,project_id,workspace_id,name,deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000021','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004','Zulu',NULL),
 ('00000000-0000-4000-8000-000000000022','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004','Alpha',now()),
 ('00000000-0000-4000-8000-000000000023','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004','Beta',NULL),
 ('00000000-0000-4000-8000-000000000024','00000000-0000-4000-8000-000000000099','00000000-0000-4000-8000-000000000004','Foreign',NULL),
 ('00000000-0000-4000-8000-000000000025','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000099','Other workspace',NULL);
INSERT INTO issue_labels(issue_id,label_id,deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000021',NULL),
 ('00000000-0000-4000-8000-000000000006','00000000-0000-4000-8000-000000000022',NULL),
 ('00000000-0000-4000-8000-000000000007','00000000-0000-4000-8000-000000000023',now()),
 ('00000000-0000-4000-8000-000000000009','00000000-0000-4000-8000-000000000021',NULL),
 ('00000000-0000-4000-8000-000000000009','00000000-0000-4000-8000-000000000022',NULL),
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000024',NULL),
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000025',NULL),
 ('00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000023',NULL);
`

func cycleLabelFixture(t *testing.T) *Repository {
	t.Helper()
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, cycleIssueQuerySchema+liveLabelSeed)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) })
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func cycleLabelIDs(rows []map[string]any) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row["id"].(string))
	}
	sort.Strings(ids)
	return ids
}
func cycleCaseID(suffix string) string { return "00000000-0000-4000-8000-0000000000" + suffix }

type cycleLabelBucket struct {
	Results json.RawMessage `json:"results"`
	Total   int             `json:"total_results"`
}
type cycleLabelEnvelope struct {
	Results map[string]cycleLabelBucket `json:"results"`
	Total   int                         `json:"total_count"`
	Count   int                         `json:"count"`
	Pages   int                         `json:"total_pages"`
	Next    bool                        `json:"next_page_results"`
}

func cycleLabelGroups(t *testing.T, raw []byte, subgroup bool) (cycleLabelEnvelope, map[string][]string) {
	t.Helper()
	var envelope cycleLabelEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	buckets := envelope.Results
	if subgroup {
		rawBuckets := buckets["low"].Results
		buckets = nil
		if err := json.Unmarshal(rawBuckets, &buckets); err != nil {
			t.Fatal(err)
		}
	}
	groups := map[string][]string{}
	for key, bucket := range buckets {
		var rows []map[string]any
		if err := json.Unmarshal(bucket.Results, &rows); err != nil {
			t.Fatal(err)
		}
		groups[key] = cycleLabelIDs(rows)
	}
	return envelope, groups
}

func TestCycleLiveLabelRelationContract(t *testing.T) {
	repository := cycleLabelFixture(t)
	ctx := context.Background()
	user := uuid.MustParse(cycleCaseID("01"))
	project := uuid.MustParse(cycleCaseID("02"))
	cycle := uuid.MustParse(cycleCaseID("03"))
	query := func(testingScope *testing.T, options IssueListOptions) issueQueryResult {
		testingScope.Helper()
		result, err := repository.queryIssues(ctx, user, "demo", project, cycle, options, 30, 0)
		if err != nil {
			testingScope.Fatal(err)
		}
		return result
	}
	none := []string{cycleCaseID("06"), cycleCaseID("07"), cycleCaseID("08"), cycleCaseID("10")}
	live := []string{cycleCaseID("05"), cycleCaseID("09")}
	t.Run("projection", func(t *testing.T) {
		result := query(t, IssueListOptions{})
		expected := map[string][]string{cycleCaseID("05"): {cycleCaseID("21")}, cycleCaseID("09"): {cycleCaseID("21")}, cycleCaseID("11"): {cycleCaseID("23")}}
		for _, row := range result.rows {
			raw, _ := json.Marshal(row["label_ids"])
			var labels []string
			if err := json.Unmarshal(raw, &labels); err != nil {
				t.Fatal(err)
			}
			want := expected[row["id"].(string)]
			if want == nil {
				want = []string{}
			}
			if !reflect.DeepEqual(labels, want) {
				t.Errorf("%s labels=%v want=%v", row["id"], labels, want)
			}
		}
	})
	t.Run("None filter", func(t *testing.T) {
		if got := cycleLabelIDs(query(t, IssueListOptions{Filters: map[string]string{"labels": "None"}}).rows); !reflect.DeepEqual(got, none) {
			t.Fatalf("None IDs=%v want=%v", got, none)
		}
	})
	t.Run("selected filter", func(t *testing.T) {
		if got := cycleLabelIDs(query(t, IssueListOptions{Filters: map[string]string{"labels": cycleCaseID("21")}}).rows); !reflect.DeepEqual(got, live) {
			t.Fatalf("selected IDs=%v want=%v", got, live)
		}
	})
	for _, id := range []string{cycleCaseID("22"), cycleCaseID("24"), cycleCaseID("25")} {
		t.Run("invalid label filter "+id, func(t *testing.T) {
			for _, options := range []IssueListOptions{{Filters: map[string]string{"labels": id}}, {ComplexFilter: `{"label_id":"` + id + `"}`}} {
				if result := query(t, options); len(result.rows) != 0 || result.total != 0 {
					t.Fatalf("invalid label %s matched %v", id, cycleLabelIDs(result.rows))
				}
			}
		})
	}

	t.Run("standalone subgroup", func(t *testing.T) {
		result := query(t, IssueListOptions{SubGroupBy: "labels__id"})
		if got := cycleLabelIDs(result.rows); !reflect.DeepEqual(got, []string{cycleCaseID("05"), cycleCaseID("06"), cycleCaseID("07"), cycleCaseID("08"), cycleCaseID("09"), cycleCaseID("10"), cycleCaseID("11")}) || result.total != 7 {
			t.Fatalf("standalone subgroup rows=%d total=%d", len(result.rows), result.total)
		}
	})
	t.Run("ordering", func(t *testing.T) {
		result := query(t, IssueListOptions{OrderBy: "labels__name"})
		if result.rows[0]["id"] != cycleCaseID("11") {
			t.Fatalf("first label-ordered row=%v", result.rows[0])
		}
		if got := cycleLabelIDs(result.rows[1:3]); !reflect.DeepEqual(got, live) {
			t.Fatalf("live sorted labels=%v want=%v", got, live)
		}
	})
	expected := map[string][]string{cycleCaseID("21"): live, cycleCaseID("23"): {cycleCaseID("11")}, "None": none}
	for _, subgroup := range []bool{false, true} {
		t.Run(map[bool]string{false: "grouped", true: "subgrouped"}[subgroup], func(t *testing.T) {
			options := IssueListOptions{GroupBy: "labels__id"}
			if subgroup {
				options.GroupBy = "priority"
				options.SubGroupBy = "labels__id"
			}
			raw, err := repository.groupedIssues(ctx, user, "demo", project, cycle, options, 30, 30, 30, 0, false)
			if err != nil {
				t.Fatal(err)
			}
			envelope, groups := cycleLabelGroups(t, raw, subgroup)
			if !reflect.DeepEqual(groups, expected) || envelope.Total != 7 || envelope.Count != 7 {
				t.Fatalf("groups=%v total=%d count=%d want=%v", groups, envelope.Total, envelope.Count, expected)
			}
		})
	}
	t.Run("label universe", func(t *testing.T) {
		var raw []byte
		if err := repository.database.QueryRow(ctx, "SELECT "+strings.ReplaceAll(issueGroupUniverseSQL("labels__id"), "$3", "$1"), project).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var ids []string
		if err := json.Unmarshal(raw, &ids); err != nil {
			t.Fatal(err)
		}
		sort.Strings(ids)
		want := []string{cycleCaseID("21"), cycleCaseID("23"), "None"}
		sort.Strings(want)
		if !reflect.DeepEqual(ids, want) {
			t.Fatalf("universe=%v want=%v", ids, want)
		}
	})
	for field, key := range map[string]string{"priority": "low", "state_id": "None", "state__group": "None", "assignees__id": "None", "issue_module__module_id": "None", "project_id": project.String(), "cycle_id": cycle.String()} {
		t.Run("supported subgroup "+field, func(t *testing.T) {
			raw, err := repository.groupedIssues(ctx, user, "demo", project, cycle, IssueListOptions{GroupBy: "labels__id", SubGroupBy: field}, 30, 30, 30, 0, false)
			if err != nil {
				t.Fatal(err)
			}
			var envelope cycleLabelEnvelope
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Fatal(err)
			}
			for label, want := range expected {
				var buckets map[string]cycleLabelBucket
				if err := json.Unmarshal(envelope.Results[label].Results, &buckets); err != nil {
					t.Fatal(err)
				}
				if len(buckets) != 1 {
					t.Fatalf("subgroup %s buckets=%v", field, buckets)
				}
				bucket, exists := buckets[key]
				if !exists {
					t.Fatalf("subgroup %s missing key %s; buckets=%v", field, key, buckets)
				}
				var rows []map[string]any
				if err := json.Unmarshal(bucket.Results, &rows); err != nil {
					t.Fatal(err)
				}
				if got := cycleLabelIDs(rows); !reflect.DeepEqual(got, want) {
					t.Fatalf("subgroup %s label %s ids=%v want=%v", field, label, got, want)
				}
			}
			if envelope.Total != 7 || envelope.Count != 7 {
				t.Fatalf("subgroup %s total=%d count=%d", field, envelope.Total, envelope.Count)
			}
		})
	}

	t.Run("group pagination", func(t *testing.T) {
		raw, err := repository.groupedIssues(ctx, user, "demo", project, cycle, IssueListOptions{GroupBy: "labels__id"}, 1, 1, 1, 0, false)
		if err != nil {
			t.Fatal(err)
		}
		envelope, groups := cycleLabelGroups(t, raw, false)
		if envelope.Total != 7 || envelope.Pages != 4 || !envelope.Next || envelope.Count != 3 {
			t.Fatalf("page=%+v groups=%v", envelope, groups)
		}
	})
}
