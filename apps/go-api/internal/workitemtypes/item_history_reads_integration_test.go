//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	history "github.com/CaliLuke/go-plane/internal/workitemhistory"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
)

func historyReader(t *testing.T, f projectTypeFixture, state types.ItemHistoryState, field string) (*history.Repository, history.Request) {
	t.Helper()
	repo, err := history.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	return repo, history.Request{Slug: "types", ProjectID: uuid.MustParse(f.project), ResourceID: uuid.MustParse(state.ItemID), PropertyID: uuid.MustParse(field), UserID: uuid.MustParse(f.member), Timezone: "UTC"}
}

func TestPropertyHistoryReadsRetainedDefinitionsAndRejectsScopeMismatch(t *testing.T) {
	f, state, field := newHistoryRecorderFixture(t)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "delete", ID: field, Revision: catalog.Revision}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	command.ReviewID = review.ID
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	repo, request := historyReader(t, f, state, field)
	request.Operation = "property_history"
	raw, err := repo.Read(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var fields []struct {
		PropertyID string `json:"property_id"`
		Title      string `json:"title"`
		Count      int    `json:"version_count"`
	}
	if err = json.Unmarshal(raw, &fields); err != nil || len(fields) != 1 || fields[0].PropertyID != field || fields[0].Title != "Acceptance notes" || fields[0].Count != 1 {
		t.Fatalf("catalog=%s error=%v", raw, err)
	}
	request.Operation = "property_versions"
	raw, err = repo.Read(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var page struct {
		Results []struct {
			ID     string `json:"id"`
			Source string `json:"source"`
		}
	}
	if err = json.Unmarshal(raw, &page); err != nil || len(page.Results) != 1 || page.Results[0].Source != "preservation" || strings.Contains(string(raw), "<p>Before</p>") {
		t.Fatalf("metadata=%s error=%v", raw, err)
	}
	request.Operation = "property_version_detail"
	request.VersionID = uuid.MustParse(page.Results[0].ID)
	raw, err = repo.Read(f.ctx, request)
	if err != nil || !strings.Contains(string(raw), "Before") {
		t.Fatalf("retained value unavailable: %s %v", raw, err)
	}
	for _, scope := range []string{"workspace", "project", "item", "field", "version"} {
		t.Run(scope, func(t *testing.T) {
			bad := request
			switch scope {
			case "workspace":
				bad.Slug = "elsewhere"
			case "project":
				bad.ProjectID = uuid.New()
			case "item":
				bad.ResourceID = uuid.New()
			case "field":
				bad.PropertyID = uuid.New()
			case "version":
				bad.VersionID = uuid.New()
			}
			body, readErr := repo.Read(f.ctx, bad)
			if readErr == nil || len(body) != 0 {
				t.Fatalf("foreign scope exposed history: %s %v", body, readErr)
			}
		})
	}
	for _, mutation := range []string{
		`UPDATE issues SET deleted_at=now()`,
		`UPDATE issues SET is_draft=true,created_by_id=NULL`,
		`UPDATE workspace_members SET is_active=false`,
		`UPDATE projects SET deleted_at=now()`,
		`UPDATE workspaces SET deleted_at=now()`,
		`UPDATE project_members SET role=5; UPDATE issues SET created_by_id=NULL; UPDATE projects SET guest_view_all_features=false`,
	} {
		t.Run(mutation, func(t *testing.T) {
			tx, beginErr := f.db.Begin(f.ctx)
			if beginErr != nil {
				t.Fatal(beginErr)
			}
			defer func() { testsupport.Rollback(t, tx, f.ctx) }()
			if _, execErr := tx.Exec(f.ctx, mutation); execErr != nil {
				t.Fatal(execErr)
			}
			guarded, createErr := history.NewRepository(tx)
			if createErr != nil {
				t.Fatal(createErr)
			}
			for _, operation := range []string{"property_history", "property_versions", "property_version_detail"} {
				denied := request
				denied.Operation = operation
				body, readErr := guarded.Read(f.ctx, denied)
				if readErr == nil || len(body) != 0 {
					t.Fatalf("%s disclosed history: %s %v", operation, body, readErr)
				}
			}
		})
	}
}

func TestPropertyVersionCursorRemainsStableAcrossAppends(t *testing.T) {
	f, state, field := newHistoryRecorderFixture(t)
	f.exec(`INSERT INTO work_item_property_versions(id,workspace_id,issue_id,property_id,value,search_text,actor_id,created_at,property_title,source) SELECT gen_random_uuid(),$1,$2,$3,'"<p>Revision</p>"','Revision',$4,now()-n*interval '1 minute','Acceptance notes','change' FROM generate_series(1,52)n`, f.workspace, state.ItemID, field, f.member)
	repo, request := historyReader(t, f, state, field)
	request.Operation = "property_versions"
	raw, err := repo.Read(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var first, second struct {
		Results []struct{ ID string }
		Next    *string `json:"next_cursor"`
	}
	if err = json.Unmarshal(raw, &first); err != nil || len(first.Results) != 50 || first.Next == nil {
		t.Fatalf("first page=%s %v", raw, err)
	}
	f.exec(`INSERT INTO work_item_property_versions(id,workspace_id,issue_id,property_id,value,search_text,actor_id) VALUES(gen_random_uuid(),$1,$2,$3,'"<p>New</p>"','New',$4)`, f.workspace, state.ItemID, field, f.member)
	cursor := uuid.MustParse(*first.Next)
	request.BeforeID = &cursor
	raw, err = repo.Read(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &second); err != nil || len(second.Results) != 2 || second.Next != nil {
		t.Fatalf("second page=%s %v", raw, err)
	}
	for _, old := range first.Results {
		for _, next := range second.Results {
			if old.ID == next.ID {
				t.Fatal("append duplicated a revision")
			}
		}
	}
	wrong := uuid.New()
	request.BeforeID = &wrong
	if _, err = repo.Read(f.ctx, request); !errors.Is(err, history.ErrNotFound) {
		t.Fatalf("foreign cursor accepted: %v", err)
	}
}

func TestPropertyVersionReadSanitizesLegacyRichText(t *testing.T) {
	f, state, field := newHistoryRecorderFixture(t)
	version := uuid.New()
	f.exec(`INSERT INTO work_item_property_versions(id,workspace_id,issue_id,property_id,value,search_text) VALUES($1,$2,$3,$4,to_jsonb($5::text),'Unsafe')`, version, f.workspace, state.ItemID, field, `<p>Safe</p><script>alert(1)</script><img src="javascript:alert(2)" onerror="alert(3)">`)
	repo, request := historyReader(t, f, state, field)
	request.Operation = "property_version_detail"
	request.VersionID = version
	raw, err := repo.Read(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var detail struct{ Value string }
	if err = json.Unmarshal(raw, &detail); err != nil || !strings.Contains(detail.Value, "Safe") || strings.Contains(detail.Value, "script") || strings.Contains(detail.Value, "onerror") {
		t.Fatalf("unsafe legacy value: %s %v", raw, err)
	}
}
