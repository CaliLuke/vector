package notionsync

import (
	"context"
	"fmt"
	"testing"
)

const rootID = "00000000-0000-4000-8000-000000000001"
const moduleA = "00000000-0000-4000-8000-000000000002"
const moduleB = "00000000-0000-4000-8000-000000000003"
const taskID = "00000000-0000-4000-8000-000000000004"
const parentID = "00000000-0000-4000-8000-000000000005"

type sourceFixture map[string]Page

func (s sourceFixture) Page(_ context.Context, id string) (Page, error) {
	p, ok := s[id]
	if !ok {
		return Page{}, fmt.Errorf("missing page %s", id)
	}
	return p, nil
}
func exampleSource() sourceFixture {
	return sourceFixture{
		rootID:   {ID: rootID, Title: "Roadmap", Relations: map[string][]string{"🏗️ Projects": {moduleA, moduleB}}},
		moduleA:  {ID: moduleA, Title: "A", Relations: map[string][]string{"🧑‍💻 Tasks": {taskID}}},
		moduleB:  {ID: moduleB, Title: "B", Relations: map[string][]string{"🧑‍💻 Tasks": {taskID}}},
		taskID:   {ID: taskID, Title: "Task", Status: "Inbox", Relations: map[string][]string{"Parent": {parentID}}},
		parentID: {ID: parentID, Title: "Parent", Status: "In progress", Relations: map[string][]string{"Child Tasks": {"unrelated-task"}}},
	}
}
func TestDiscoveryDeduplicatesSharedTaskAndFlagsExternalParent(t *testing.T) {
	graph, err := Discover(context.Background(), exampleSource(), rootID, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Modules) != 2 || len(graph.Tasks) != 1 || len(graph.Members[taskID]) != 2 || len(graph.Warnings) != 1 {
		t.Fatalf("unexpected selected graph: %+v", graph)
	}
}
func TestIncludeParentsDoesNotPullInSiblings(t *testing.T) {
	graph, err := Discover(context.Background(), exampleSource(), rootID, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Tasks) != 2 || graph.Tasks[1].ID != parentID {
		t.Fatal("parent import crossed into siblings")
	}
}
func TestMissingSourceAbortsBeforeAnyVectorWrites(t *testing.T) {
	source := exampleSource()
	delete(source, taskID)
	if _, err := Discover(context.Background(), source, rootID, false, false); err == nil {
		t.Fatal("missing task silently ignored")
	}
}
func TestParentCycleRejectedBeforeWrites(t *testing.T) {
	source := exampleSource()
	p := source[parentID]
	p.Relations["Parent"] = []string{taskID}
	source[parentID] = p
	graph, err := Discover(context.Background(), source, rootID, true, false)
	if err != nil {
		t.Fatal(err)
	}
	runner, api := fixtureRunner(t)
	if _, err = runner.Run(context.Background(), graph); err == nil || api.creates != 0 {
		t.Fatal("parent cycle was not rejected before mutation")
	}
}
func TestPageIDAcceptsNamedNotionURL(t *testing.T) {
	id, err := PageID("https://app.notion.com/p/Example-00000000000040008000000000000001?source=copy_link")
	if err != nil || id != rootID {
		t.Fatalf("id=%s err=%v", id, err)
	}
}
