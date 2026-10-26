package notionsync

import (
	"context"
	"strings"
	"testing"
)

const teamID = "00000000-0000-4000-8000-000000000006"
const secondRoadmapID = "00000000-0000-4000-8000-000000000007"
const otherRoadmapID = "00000000-0000-4000-8000-000000000008"
const otherTeamID = "00000000-0000-4000-8000-000000000009"

func teamSnapshot() Snapshot {
	return Snapshot{
		teamID: {ID: teamID, Title: "Platform App", URL: "https://www.notion.so/" + teamID},
		rootID: {ID: rootID, Title: "Roadmap A", Status: "In progress", Relations: map[string][]string{
			"👷 Team": {teamID}, "🏗️ Projects": {moduleA}}},
		secondRoadmapID: {ID: secondRoadmapID, Title: "Roadmap B", Status: "Done", Relations: map[string][]string{
			"👷 Team": {teamID}, "🏗️ Projects": {moduleB}}},
		otherRoadmapID: {ID: otherRoadmapID, Title: "Other team", Relations: map[string][]string{
			"👷 Team": {otherTeamID}, "🏗️ Projects": {"not-in-snapshot"}}},
		moduleA: {ID: moduleA, Title: "Project A", Status: "In progress", URL: "https://www.notion.so/" + moduleA,
			Relations: map[string][]string{"🧑‍💻 Tasks": {taskID}}},
		moduleB: {ID: moduleB, Title: "Project B", Status: "Done", URL: "https://www.notion.so/" + moduleB,
			Relations: map[string][]string{"🧑‍💻 Tasks": {parentID}}},
		taskID: {ID: taskID, Title: "Task A", Status: "Inbox", HTML: "<p>Task details</p>",
			Relations: map[string][]string{"Parent": {"external-parent"}}},
		parentID: {ID: parentID, Title: "Task B", Status: "Done"},
	}
}

func TestTeamImportCreatesOneProjectWithRoadmapModulesAndProjectParents(t *testing.T) {
	runner, api := fixtureRunner(t)
	source := teamSnapshot()
	graph, err := DiscoverTeam(context.Background(), source, teamID)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Modules) != 2 || len(graph.Tasks) != 4 || graph.Roadmap.ID != teamID {
		t.Fatalf("incorrect team scope: %+v", graph)
	}
	report, err := runner.Run(context.Background(), graph)
	if err != nil || len(report.Conflicts) != 0 || report.Created != 7 {
		t.Fatalf("team import: %+v %v", report, err)
	}
	project := report.ProjectID
	parent := runner.Journal.Entries["issues:"+moduleA].ID
	child := runner.Journal.Entries["issues:"+taskID].ID
	if api.rows[runner.Vector.collection("issues", project)+child+"/"]["parent"] != parent {
		t.Fatal("task was not nested under its source project")
	}
	module := runner.Journal.Entries["modules:"+rootID].ID
	if !equivalent(api.members[runner.Vector.collection("modules", project)+module+"/module-issues/"], []string{parent, child}) {
		t.Fatal("roadmap module must contain both project and task")
	}
	if source[taskID].Relations["Parent"][0] != "external-parent" {
		t.Fatal("discovery modified the source snapshot")
	}
	creates, patches := api.creates, api.patches
	repeated, err := DiscoverTeam(context.Background(), source, teamID)
	if err != nil {
		t.Fatal(err)
	}
	report, err = runner.Run(context.Background(), repeated)
	if err != nil || len(report.Conflicts) != 0 || api.creates != creates || api.patches != patches {
		t.Fatalf("repeat changed destination: %+v %v", report, err)
	}
}

func TestTeamSharedTaskKeepsAllModulesWithoutChoosingAParent(t *testing.T) {
	source := teamSnapshot()
	source[moduleB].Relations["🧑‍💻 Tasks"] = []string{taskID}
	graph, err := DiscoverTeam(context.Background(), source, teamID)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Tasks) != 3 || !equivalent(graph.Members[taskID], []string{rootID, secondRoadmapID}) {
		t.Fatalf("shared task duplicated or lost module: %+v", graph)
	}
	task := graph.Tasks[2]
	if task.ID != taskID || len(task.Relations["Parent"]) != 0 || !strings.Contains(task.HTML, source[moduleA].URL) || !strings.Contains(task.HTML, source[moduleB].URL) || len(graph.Warnings) != 1 {
		t.Fatalf("shared task lost source context: %+v", task)
	}
}

func TestTeamMissingTaskAndUnknownTeamFailBeforeImport(t *testing.T) {
	source := teamSnapshot()
	delete(source, taskID)
	if _, err := DiscoverTeam(context.Background(), source, teamID); err == nil {
		t.Fatal("missing task silently omitted")
	}
	if _, err := DiscoverTeam(context.Background(), source, otherTeamID); err == nil {
		t.Fatal("missing team silently accepted")
	}
}
