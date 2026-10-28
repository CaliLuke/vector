//go:build integration

package modulereads

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

func TestModuleLabelAxisUsesOnlyLiveLabelRelations(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL);
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE issues (id uuid PRIMARY KEY, project_id uuid NOT NULL, workspace_id uuid NOT NULL);
CREATE TABLE labels (id uuid PRIMARY KEY, project_id uuid NOT NULL, workspace_id uuid NOT NULL, deleted_at timestamptz);
CREATE TABLE issue_labels (issue_id uuid NOT NULL, label_id uuid NOT NULL, deleted_at timestamptz);
INSERT INTO workspaces (id, slug) VALUES ('00000000-0000-4000-8000-000000000002', 'demo');
INSERT INTO projects (id, workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002');
INSERT INTO issues (id, project_id, workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000013', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000014', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000016', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002');
INSERT INTO labels (id, project_id, workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000021', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000022', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000023', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000024', '00000000-0000-4000-8000-000000000004', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000025', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000003');
INSERT INTO issue_labels (issue_id, label_id) VALUES
 ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000021'),
 ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000022'),
 ('00000000-0000-4000-8000-000000000013', '00000000-0000-4000-8000-000000000023'),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000021'),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000022'),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000024'),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000025'),
 ('00000000-0000-4000-8000-000000000016', '00000000-0000-4000-8000-000000000021'),
 ('00000000-0000-4000-8000-000000000016', '00000000-0000-4000-8000-000000000023');
UPDATE labels SET deleted_at = now() WHERE id = '00000000-0000-4000-8000-000000000022';
UPDATE issue_labels SET deleted_at = now()
 WHERE issue_id = '00000000-0000-4000-8000-000000000013';
`)
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

	var rawUniverse []byte
	universeSQL := strings.Replace(moduleIssueGroupUniverseSQL("labels__id"), "$3", "$1", 1)
	universeSQL = strings.ReplaceAll(universeSQL, "$2", "'demo'")
	if err = connection.QueryRow(ctx, `SELECT `+universeSQL, "00000000-0000-4000-8000-000000000001").Scan(&rawUniverse); err != nil {
		t.Fatal(err)
	}
	var universe []string
	if err = json.Unmarshal(rawUniverse, &universe); err != nil {
		t.Fatal(err)
	}
	actualUniverse := map[string]bool{}
	for _, labelID := range universe {
		actualUniverse[labelID] = true
	}
	wantUniverse := map[string]bool{
		"00000000-0000-4000-8000-000000000021": true,
		"00000000-0000-4000-8000-000000000023": true,
		"None":                                 true,
	}
	if len(actualUniverse) != len(wantUniverse) {
		t.Errorf("live label group universe = %#v; want active project/workspace labels plus None", universe)
	}
	for labelID := range wantUniverse {
		if !actualUniverse[labelID] {
			t.Errorf("live label group universe missing %q: %#v", labelID, universe)
		}
	}

	rows, err := connection.Query(ctx, `
SELECT issue.id::text, axis.group_key
FROM issues issue
`+moduleIssueAxisSQL("labels__id", "axis", "group_key")+`
ORDER BY issue.id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	actual := map[string][]string{}
	for rows.Next() {
		var issueID, groupKey string
		if err := rows.Scan(&issueID, &groupKey); err != nil {
			t.Fatal(err)
		}
		actual[issueID] = append(actual[issueID], groupKey)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"00000000-0000-4000-8000-000000000011": {"00000000-0000-4000-8000-000000000021"},
		"00000000-0000-4000-8000-000000000012": {"None"},
		"00000000-0000-4000-8000-000000000013": {"None"},
		"00000000-0000-4000-8000-000000000014": {"None"},
		"00000000-0000-4000-8000-000000000015": {"00000000-0000-4000-8000-000000000021"},
		"00000000-0000-4000-8000-000000000016": {"00000000-0000-4000-8000-000000000021", "00000000-0000-4000-8000-000000000023"},
	}
	if len(actual) != len(want) {
		t.Fatalf("label axis issues = %#v; want one or more buckets for each issue: %#v", actual, want)
	}
	for issueID, wantGroups := range want {
		gotGroups := actual[issueID]
		sort.Strings(gotGroups)
		sort.Strings(wantGroups)
		if !reflect.DeepEqual(gotGroups, wantGroups) {
			t.Errorf("issue %s label axis = %#v; want exactly %#v (all live relation targets only)", issueID, gotGroups, wantGroups)
		}
	}
}

func TestModuleGroupedPaginationCountsOnlyLiveLabels(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) }()
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, deleted_at timestamptz, archived_at timestamptz, guest_view_all_features boolean DEFAULT false);
CREATE TABLE project_members (project_id uuid, workspace_id uuid, member_id uuid, role integer, is_active boolean, deleted_at timestamptz);
CREATE TABLE workspace_members (workspace_id uuid, member_id uuid, role integer, is_active boolean, deleted_at timestamptz);
CREATE TABLE modules (id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, deleted_at timestamptz, archived_at timestamptz);
CREATE TABLE module_issues (id uuid PRIMARY KEY, issue_id uuid, module_id uuid, project_id uuid, workspace_id uuid, deleted_at timestamptz);
CREATE TABLE issues (id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, type_id uuid, name text, state_id uuid, sort_order double precision, completed_at timestamptz, estimate_point_id uuid, priority text, start_date date, target_date date, sequence_id integer, parent_id uuid, created_at timestamptz, updated_at timestamptz, created_by_id uuid, updated_by_id uuid, archived_at timestamptz, is_draft boolean, deleted_at timestamptz);
CREATE TABLE states (id uuid, project_id uuid, sequence integer, is_triage boolean, deleted_at timestamptz, "group" text);
CREATE TABLE labels (id uuid PRIMARY KEY, project_id uuid NOT NULL, workspace_id uuid NOT NULL, deleted_at timestamptz);
CREATE TABLE issue_labels (issue_id uuid, label_id uuid, deleted_at timestamptz);
CREATE TABLE issue_assignees (issue_id uuid, assignee_id uuid, deleted_at timestamptz);
CREATE TABLE cycle_issues (issue_id uuid, cycle_id uuid, deleted_at timestamptz, created_at timestamptz);
CREATE TABLE file_assets (issue_id uuid, deleted_at timestamptz, entity_type text);
CREATE TABLE issue_links (issue_id uuid, deleted_at timestamptz);
CREATE TABLE intake_issues (issue_id uuid, status integer);
INSERT INTO workspaces (id, slug) VALUES ('00000000-0000-4000-8000-000000000002', 'demo');
INSERT INTO projects (id, workspace_id) VALUES ('00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002');
INSERT INTO modules (id, project_id, workspace_id) VALUES ('00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002');
INSERT INTO project_members (project_id, workspace_id, member_id, role, is_active) VALUES ('00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000031', 20, true);
INSERT INTO issues (id, project_id, workspace_id, name, created_at, is_draft) VALUES
 ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Issue 11', now(), false),
 ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Issue 12', now(), false),
 ('00000000-0000-4000-8000-000000000013', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Issue 13', now(), false),
 ('00000000-0000-4000-8000-000000000014', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Issue 14', now(), false),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Issue 15', now(), false),
 ('00000000-0000-4000-8000-000000000016', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'Issue 16', now(), false);
INSERT INTO module_issues (id, issue_id, module_id, project_id, workspace_id) VALUES
 ('00000000-0000-4000-8000-000000000041', '00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000042', '00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000043', '00000000-0000-4000-8000-000000000013', '00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000044', '00000000-0000-4000-8000-000000000014', '00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000045', '00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002'),
 ('00000000-0000-4000-8000-000000000046', '00000000-0000-4000-8000-000000000016', '00000000-0000-4000-8000-000000000032', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002');
INSERT INTO labels (id, project_id, workspace_id, deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000021', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', NULL),
 ('00000000-0000-4000-8000-000000000022', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', now()),
 ('00000000-0000-4000-8000-000000000023', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', NULL);
INSERT INTO issue_labels (issue_id, label_id, deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000021', NULL),
 ('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000022', NULL),
 ('00000000-0000-4000-8000-000000000013', '00000000-0000-4000-8000-000000000023', now()),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000021', NULL),
 ('00000000-0000-4000-8000-000000000015', '00000000-0000-4000-8000-000000000022', NULL),
 ('00000000-0000-4000-8000-000000000016', '00000000-0000-4000-8000-000000000021', NULL),
 ('00000000-0000-4000-8000-000000000016', '00000000-0000-4000-8000-000000000023', NULL);
`)
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
	page, err := repository.issueResults(ctx,
		uuid.MustParse("00000000-0000-4000-8000-000000000031"), "demo",
		uuid.MustParse("00000000-0000-4000-8000-000000000001"),
		uuid.MustParse("00000000-0000-4000-8000-000000000032"),
		IssueListOptions{GroupBy: "labels__id", SubGroupBy: "state_id", Limit: 2},
		map[string]bool{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 7 {
		t.Fatalf("expanded grouped total = %d; want 7 rows including two groups for the two-live-label issue", page.Total)
	}
	wantGroups := map[string]int{
		"00000000-0000-4000-8000-000000000021": 3,
		"00000000-0000-4000-8000-000000000023": 1,
		"None":                                 3,
	}
	if !reflect.DeepEqual(page.GroupCounts, wantGroups) {
		t.Fatalf("group counts = %#v; want %#v", page.GroupCounts, wantGroups)
	}
	wantSubgroups := map[string]int{
		"00000000-0000-4000-8000-000000000021:::None": 3,
		"00000000-0000-4000-8000-000000000023:::None": 1,
		"None:::None": 3,
	}
	if !reflect.DeepEqual(page.SubGroupCounts, wantSubgroups) {
		t.Fatalf("subgroup counts = %#v; want %#v", page.SubGroupCounts, wantSubgroups)
	}
	if !page.GroupedHasNext {
		t.Fatal("grouped page with more than two rows in the None group must report a next page")
	}
}
