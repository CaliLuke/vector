package notionsync

import (
	"context"
	"testing"
)

func TestCompleteImportPreservesHierarchyAndRepeatsWithoutWrites(t *testing.T) {
	runner, api := fixtureRunner(t)
	ctx := context.Background()
	graph, err := Discover(ctx, exampleSource(), rootID, true, false)
	if err != nil {
		t.Fatal(err)
	}
	graph.Tasks[0].Priority = "P1"
	graph.Tasks[0].Start = "2026-01-01"
	graph.Tasks[0].End = "2026-01-31"
	graph.Tasks[0].People = []string{"notion-person"}
	graph.Tasks[0].HTML = "<p>Acceptance criteria</p>"
	runner.Users = map[string]string{"notion-person": "vector-person"}
	report, err := runner.Run(ctx, graph)
	if err != nil || len(report.Conflicts) != 0 {
		t.Fatalf("import: %+v %v", report, err)
	}
	project := report.ProjectID
	task := runner.Journal.Entries["issues:"+taskID].ID
	parent := runner.Journal.Entries["issues:"+parentID].ID
	if api.rows[runner.Vector.collection("issues", project)+task+"/"]["parent"] != parent {
		t.Fatal("parent relationship missing")
	}
	row := api.rows[runner.Vector.collection("issues", project)+task+"/"]
	if row["priority"] != "high" || row["start_date"] != "2026-01-01" || row["target_date"] != "2026-01-31" || !equivalent(row["assignees"], []string{"vector-person"}) {
		t.Fatalf("task fields did not survive import: %+v", row)
	}
	for _, source := range []string{moduleA, moduleB} {
		module := runner.Journal.Entries["modules:"+source].ID
		members := api.members[runner.Vector.collection("modules", project)+module+"/module-issues/"]
		if len(members) != 1 || members[0] != task {
			t.Fatalf("shared task membership: %v", members)
		}
	}
	creates, patches := api.creates, api.patches
	report, err = runner.Run(ctx, graph)
	if err != nil || len(report.Conflicts) != 0 || api.creates != creates || api.patches != patches {
		t.Fatalf("repeat changed destination: %+v %v", report, err)
	}
	graph.Tasks[0].Archived = true
	report, err = runner.Run(ctx, graph)
	if err != nil || len(report.Conflicts) != 0 {
		t.Fatalf("archived task: %+v %v", report, err)
	}
	for _, members := range api.members {
		if len(members) != 1 || members[0] != task {
			t.Fatal("archiving removed membership")
		}
	}
}

func TestMissingStatusRejectedBeforeProjectCreation(t *testing.T) {
	runner, api := fixtureRunner(t)
	graph := Graph{Roadmap: Page{ID: rootID, Title: "Roadmap"}, Tasks: []Page{{ID: taskID, Title: "Task"}}}
	if _, err := runner.Run(context.Background(), graph); err == nil || api.creates != 0 {
		t.Fatal("invalid task caused partial import")
	}
}
