package workitems

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed testdata/model.json
var modelSourceFixtures []byte

func TestIssueModelSourceParity(t *testing.T) {
	var fixture struct {
		StripTags []struct {
			Name   string `json:"name"`
			Input  string `json:"input"`
			Output string `json:"output"`
		} `json:"strip_tags"`
		ProjectLockKeys []struct {
			UUID string `json:"uuid"`
			Key  int64  `json:"key"`
		} `json:"project_lock_keys"`
	}
	if err := json.Unmarshal(modelSourceFixtures, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range fixture.StripTags {
		t.Run("strip_tags/"+testCase.Name, func(t *testing.T) {
			var want any = testCase.Output
			if testCase.Input == "" {
				// Issue.save stores NULL for an empty description; its helper
				// strip_tags itself returns the empty string.
				want = nil
			}
			if got := stripTags(testCase.Input); got != want {
				t.Errorf("stripTags(%q) = %#v, want %#v", testCase.Input, got, want)
			}
		})
	}
	for _, testCase := range fixture.ProjectLockKeys {
		t.Run("project_lock_key/"+testCase.UUID, func(t *testing.T) {
			if got := projectLockKey(testCase.UUID); got != testCase.Key {
				t.Errorf("projectLockKey(%q) = %d, want %d", testCase.UUID, got, testCase.Key)
			}
		})
	}
}
