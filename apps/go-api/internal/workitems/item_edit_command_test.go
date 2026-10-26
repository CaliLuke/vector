package workitems

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestItemEditNormalizationDoesNotMutateProposedFields(t *testing.T) {
	id := "61ED80B5-E9B2-43E1-91A6-EAE636146FED"
	changes := map[string]any{"assignee_ids": []string{id}, "custom_properties": map[string]any{"value": json.Number("9007199254740993")}}
	before, _ := json.Marshal(changes)
	command, _, err := normalizedItemEdit(ItemEditInput{ExpectedSourceVersion: strings.Repeat("a", 64), Mode: "draft", ItemID: id, ProjectID: id, Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(changes)
	if string(before) != string(after) {
		t.Fatalf("normalization changed caller: %s -> %s", before, after)
	}
	if command.Changes["project_id"] != strings.ToLower(id) || command.Changes["custom_properties"].(map[string]any)["value"] != json.Number("9007199254740993") {
		t.Fatalf("normalized values lost precision or destination: %#v", command.Changes)
	}
}

func TestPropertyRestoreNormalizationBindsIDsWithoutMutatingCaller(t *testing.T) {
	id := "61ED80B5-E9B2-43E1-91A6-EAE636146FED"
	input := ItemEditInput{Mode: "edit", ItemID: id, ProjectID: id, ExpectedSourceVersion: strings.Repeat("a", 64), Changes: map[string]any{}, Restore: &ItemPropertyRestore{PropertyID: id, VersionID: id}}
	command, raw, err := normalizedItemEdit(input)
	if err != nil {
		t.Fatal(err)
	}
	if input.Restore.PropertyID != id || input.Restore.VersionID != id || command.Restore.PropertyID != strings.ToLower(id) || command.Restore.VersionID != strings.ToLower(id) || len(command.Changes) != 0 {
		t.Fatal("restore normalization mutated input or lost identity")
	}
	if !strings.Contains(string(raw), `"restore":`) {
		t.Fatal("receipt omitted restore identity")
	}
}

func TestPropertyRestoreNormalizationRejectsAmbiguousCommands(t *testing.T) {
	id := "61ed80b5-e9b2-43e1-91a6-eae636146fed"
	for _, change := range []string{"draft", "publish", "changes", "source", "field", "version"} {
		t.Run(change, func(t *testing.T) {
			input := ItemEditInput{Mode: "edit", ItemID: id, ProjectID: id, ReviewID: id, ExpectedSourceVersion: strings.Repeat("a", 64), Changes: map[string]any{}, Restore: &ItemPropertyRestore{PropertyID: id, VersionID: id}}
			switch change {
			case "draft", "publish":
				input.Mode = change
			case "changes":
				input.Changes["name"] = "Unrelated change"
			case "source":
				input.ExpectedSourceVersion = ""
			case "field":
				input.Restore.PropertyID = "bad"
			case "version":
				input.Restore.VersionID = "bad"
			}
			if _, _, err := normalizedItemEdit(input); err == nil {
				t.Fatal("ambiguous restore accepted")
			}
		})
	}
}
