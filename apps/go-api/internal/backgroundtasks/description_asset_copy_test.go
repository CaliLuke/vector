package backgroundtasks

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDescriptionAssetCopyRewritesOnlyImageComponentSources(t *testing.T) {
	input := `<p><image-component src="old-asset" alt="report"></image-component><a href="old-asset">link</a><image-component src="leave-me"></image-component></p>`
	got, err := replaceDescriptionImageIDs(input, map[string]string{"old-asset": "new-asset"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `src="new-asset"`) || !strings.Contains(got, `href="old-asset"`) || !strings.Contains(got, `src="leave-me"`) {
		t.Fatalf("rewritten HTML = %q", got)
	}
}

func TestDescriptionAssetCopyFilenameAndEntityMappingMatchSource(t *testing.T) {
	for _, test := range []struct {
		attributes string
		want       string
	}{
		{`{"name":"photo.png"}`, "photo.png"},
		{`{}`, "None"},
		{`{"name":null}`, "None"},
	} {
		got, err := descriptionAssetFilename(json.RawMessage(test.attributes))
		if err != nil || got != test.want {
			t.Fatalf("descriptionAssetFilename(%s) = %q, %v; want %q", test.attributes, got, err, test.want)
		}
	}
	for _, test := range []struct{ entity, column string }{
		{"PAGE_DESCRIPTION", "page_id"},
		{"ISSUE_DESCRIPTION", "issue_id"},
		{"WORKSPACE_LOGO", "workspace_id"},
		{"DRAFT_ISSUE_ATTACHMENT", ""},
		{"future", ""},
	} {
		got, err := descriptionAssetEntityColumn(test.entity)
		if err != nil || got != test.column {
			t.Fatalf("descriptionAssetEntityColumn(%q) = %q, %v; want %q", test.entity, got, err, test.column)
		}
	}
}
