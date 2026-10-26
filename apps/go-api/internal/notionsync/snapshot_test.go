package notionsync

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func snapshotJSON(t *testing.T, pages []Page) string {
	t.Helper()
	data, err := json.Marshal(pages)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSnapshotImportReusesRoadmapAndDoesNotDuplicateTasks(t *testing.T) {
	fixture := exampleSource()
	pages := []Page{fixture[rootID], fixture[moduleA], fixture[moduleB], fixture[taskID]}
	pages[0].Relations["🏗️ Projects"][0] = "https://www.notion.so/" + moduleA
	source, err := ReadSnapshot(strings.NewReader(snapshotJSON(t, pages)))
	if err != nil {
		t.Fatal(err)
	}
	runner, api := fixtureRunner(t)
	roadmap, err := Discover(context.Background(), source, rootID, false, true)
	if err != nil {
		t.Fatal(err)
	}
	first, err := runner.Run(context.Background(), roadmap)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := Discover(context.Background(), source, rootID, false, false)
	if err != nil {
		t.Fatal(err)
	}
	imported, err := runner.Run(context.Background(), graph)
	if err != nil || imported.ProjectID != first.ProjectID || len(graph.Tasks) != 1 {
		t.Fatalf("import=%+v err=%v", imported, err)
	}
	created := api.creates
	repeated, err := runner.Run(context.Background(), graph)
	if err != nil || api.creates != created || len(repeated.Conflicts) != 0 {
		t.Fatalf("repeat created duplicates or conflicts: %+v, %v", repeated, err)
	}
}

func TestSnapshotRejectsIncompleteOrAmbiguousExports(t *testing.T) {
	fixture := exampleSource()
	cases := []string{
		`[{"id":"invalid"}]`,
		`[{"id":"` + rootID + `","relations":{"Parent":["invalid"]}}]`,
		`[{"id":"` + rootID + `","typo":true}]`,
		snapshotJSON(t, []Page{fixture[rootID], fixture[rootID]}),
		`[] []`,
	}
	for _, input := range cases {
		if _, err := ReadSnapshot(strings.NewReader(input)); err == nil {
			t.Fatal("invalid export accepted")
		}
	}
	source, err := ReadSnapshot(strings.NewReader(snapshotJSON(t, []Page{fixture[rootID]})))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(context.Background(), source, rootID, false, false); err == nil {
		t.Fatal("incomplete export accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.Page(ctx, rootID); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}
