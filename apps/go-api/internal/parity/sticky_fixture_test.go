package parity

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

type stickyFixture struct {
	Response CapturedResponse `json:"response"`
	Database struct {
		Before                map[string][]map[string]any `json:"before"`
		After                 map[string][]map[string]any `json:"after"`
		ExpectedChangedTables []string                    `json:"expected_changed_tables"`
	} `json:"database"`
}

func TestRecordedStickyCreateTransition(t *testing.T) {
	fixture := loadStickyFixture(t)
	before, after := fixtureSnapshots(t, fixture)
	option := DeclaredNormalizationsOption(Normalizations{})
	source := fixtureTransition(t, before, after, fixture.Response, fixture.Database.ExpectedChangedTables)
	target := fixtureTransition(t, before, after, fixture.Response, fixture.Database.ExpectedChangedTables)
	if diff, err := CompareTransitions(source, target, option); err != nil || diff != "" {
		t.Fatalf("CompareTransitions() diff = %q, error = %v", diff, err)
	}

	divergentAfter := cloneSnapshot(after)
	sticky := tableByName(t, &divergentAfter, "stickies")
	if len(sticky.Rows) == 0 {
		t.Fatal("recorded Sticky create fixture has no created row")
	}
	var row map[string]any
	if err := json.Unmarshal(sticky.Rows[0], &row); err != nil {
		t.Fatalf("decode divergent Sticky row: %v", err)
	}
	row["name"] = "divergent target secret"
	sticky.Rows[0] = mustMarshal(t, row)
	divergentTarget := fixtureTransition(t, before, divergentAfter, fixture.Response, fixture.Database.ExpectedChangedTables)
	diff, err := CompareTransitions(source, divergentTarget, option)
	if err == nil || !strings.Contains(diff, "stickies") || strings.Contains(diff, "divergent target secret") {
		t.Fatalf("CompareTransitions() diff = %q, error = %v; want safe Sticky divergence", diff, err)
	}
}

func TestRecordedStickyCreateChecksAllTablesForUndeclaredWrites(t *testing.T) {
	fixture := loadStickyFixture(t)
	before, after := fixtureSnapshots(t, fixture)
	divergentAfter := cloneSnapshot(after)
	accounts := tableByName(t, &divergentAfter, "accounts")
	accounts.Rows = append(accounts.Rows, json.RawMessage(`{"id":"ffffffff-ffff-4fff-8fff-ffffffffffff"}`))
	_, err := NewRuntimeTransition(before, divergentAfter, fixture.Response, fixture.Database.ExpectedChangedTables)
	if err == nil || !strings.Contains(err.Error(), "accounts") {
		t.Fatalf("NewRuntimeTransition() error = %v, want undeclared accounts write", err)
	}
}

func TestRecordedStickyCreateComparesSafeResponse(t *testing.T) {
	fixture := loadStickyFixture(t)
	before, after := fixtureSnapshots(t, fixture)
	source := fixtureTransition(t, before, after, fixture.Response, fixture.Database.ExpectedChangedTables)
	divergentResponse := fixture.Response
	divergentResponse.Status = 202
	divergentResponse.Body = json.RawMessage(`{"token":"target response secret"}`)
	target := fixtureTransition(t, before, after, divergentResponse, fixture.Database.ExpectedChangedTables)
	diff, err := CompareTransitions(source, target, DeclaredNormalizationsOption(Normalizations{}))
	if err == nil || !strings.Contains(diff, "Response") || !strings.Contains(diff, "Status") {
		t.Fatalf("CompareTransitions() diff = %q, error = %v; want response divergence", diff, err)
	}
	if strings.Contains(diff, "target response secret") {
		t.Fatalf("CompareTransitions() leaked response body in %q", diff)
	}
}

func loadStickyFixture(t *testing.T) stickyFixture {
	t.Helper()
	fixtureBytes, err := os.ReadFile("testdata/fixtures/stickies/create.json")
	if err != nil {
		t.Fatalf("read Sticky fixture: %v", err)
	}
	var fixture stickyFixture
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatalf("decode Sticky fixture: %v", err)
	}
	return fixture
}

func fixtureSnapshots(t *testing.T, fixture stickyFixture) (Snapshot, Snapshot) {
	t.Helper()
	tableNames, err := PlaneTables()
	if err != nil {
		t.Fatalf("PlaneTables() error = %v", err)
	}
	if len(fixture.Database.Before) != len(tableNames) || len(fixture.Database.After) != len(tableNames) {
		t.Fatalf("fixture table counts before=%d after=%d, want %d", len(fixture.Database.Before), len(fixture.Database.After), len(tableNames))
	}
	before := Snapshot{Tables: make([]TableDump, 0, len(tableNames))}
	after := Snapshot{Tables: make([]TableDump, 0, len(tableNames))}
	for _, tableName := range tableNames {
		beforeRows, beforeExists := fixture.Database.Before[tableName]
		afterRows, afterExists := fixture.Database.After[tableName]
		if !beforeExists || !afterExists {
			t.Fatalf("fixture missing complete dump for table %q", tableName)
		}
		columns := fixtureColumns(beforeRows, afterRows)
		before.Tables = append(before.Tables, fixtureTableDump(t, tableName, columns, beforeRows))
		after.Tables = append(after.Tables, fixtureTableDump(t, tableName, columns, afterRows))
	}
	return before, after
}

func fixtureColumns(rowSets ...[]map[string]any) []string {
	seen := make(map[string]struct{})
	for _, rows := range rowSets {
		for _, row := range rows {
			for column := range row {
				seen[column] = struct{}{}
			}
		}
	}
	columns := make([]string, 0, len(seen))
	for column := range seen {
		columns = append(columns, column)
	}
	slices.Sort(columns)
	return columns
}

func fixtureTableDump(t *testing.T, tableName string, columns []string, rows []map[string]any) TableDump {
	t.Helper()
	dump := TableDump{Name: tableName, PrimaryKey: []string{"id"}}
	for _, column := range columns {
		dump.Columns = append(dump.Columns, Column{Name: column})
	}
	for _, row := range rows {
		dump.Rows = append(dump.Rows, mustMarshal(t, row))
	}
	return dump
}

func fixtureTransition(t *testing.T, before Snapshot, after Snapshot, response CapturedResponse, expected []string) RuntimeTransition {
	t.Helper()
	transition, err := NewRuntimeTransition(before, after, response, expected)
	if err != nil {
		t.Fatalf("NewRuntimeTransition() error = %v", err)
	}
	return transition
}

func tableByName(t *testing.T, snapshot *Snapshot, name string) *TableDump {
	t.Helper()
	for index := range snapshot.Tables {
		if snapshot.Tables[index].Name == name {
			return &snapshot.Tables[index]
		}
	}
	t.Fatalf("snapshot missing table %q", name)
	return nil
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	clone := Snapshot{Tables: make([]TableDump, len(snapshot.Tables))}
	for index, table := range snapshot.Tables {
		clone.Tables[index] = TableDump{
			Name:       table.Name,
			Columns:    slices.Clone(table.Columns),
			PrimaryKey: slices.Clone(table.PrimaryKey),
			Rows:       slices.Clone(table.Rows),
		}
	}
	return clone
}

func mustMarshal(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode fixture value: %v", err)
	}
	return encoded
}
