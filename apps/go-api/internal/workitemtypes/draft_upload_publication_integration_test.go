//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

func TestDraftPublicationDistinguishesRichTextUploadsFromLiteralText(t *testing.T) {
	f := newProjectTypeFixture(t)
	richText, plainText := uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: richText, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: plainText, Title: "Literal", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}},
	)
	repo, draft, _ := seedPublicationDraft(t, f)
	asset := publicationAsset(f, draft, f.project)
	f.exec(`UPDATE file_assets SET is_uploaded=false WHERE id=$1`, asset)
	markup := fmt.Sprintf(`<image-component src="%s"></image-component>`, asset)
	body, err := json.Marshal(map[string]any{"custom_properties": map[string]any{richText: markup}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = publishDraftForTest(f, repo, draft, f.member, string(body))
	var conflict *djangoapi.HTTPError
	if !errors.As(err, &conflict) || conflict.Status != 409 {
		t.Fatalf("pending rich-text image must block publication: %v", err)
	}
	var consumed bool
	if err = f.db.QueryRow(f.ctx, `SELECT deleted_at IS NOT NULL FROM draft_issues WHERE id=$1`, draft).Scan(&consumed); err != nil || consumed {
		t.Fatalf("blocked publication consumed draft: %v %v", consumed, err)
	}
	body, err = json.Marshal(map[string]any{"custom_properties": map[string]any{plainText: markup}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := publishDraftForTest(f, repo, draft, f.member, string(body))
	if err != nil {
		t.Fatal(err)
	}
	var saved string
	if err = f.db.QueryRow(f.ctx, `SELECT custom_properties->>$2 FROM issues WHERE id=$1`, resultIssueID(t, result), plainText).Scan(&saved); err != nil || saved != markup {
		t.Fatalf("literal text was lost: %q %v", saved, err)
	}
}

func TestDraftPublicationWaitsForUploadsWithoutConsumingTheDraft(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	asset := publicationAsset(f, draft, f.project)
	f.exec(`UPDATE file_assets SET is_uploaded=false WHERE id=$1`, asset)
	body := fmt.Sprintf(`{"description_html":"<image-component src=\"%s\"></image-component>"}`, asset)
	_, err := publishDraftForTest(f, repo, draft, f.member, body)
	var conflict *djangoapi.HTTPError
	if !errors.As(err, &conflict) || conflict.Status != 409 {
		t.Fatalf("pending uploads: want 409, got %v", err)
	}
	var consumed bool
	var count int
	if scanErr := f.db.QueryRow(f.ctx, `SELECT deleted_at IS NOT NULL,(SELECT count(*) FROM issues) FROM draft_issues WHERE id=$1`, draft).Scan(&consumed, &count); scanErr != nil || consumed || count != 0 {
		t.Fatalf("pending upload publication changed draft/items: %v %d %v", consumed, count, scanErr)
	}
	f.exec(`UPDATE file_assets SET is_uploaded=true WHERE id=$1`, asset)
	result, err := publishDraftForTest(f, repo, draft, f.member, body)
	if err != nil {
		t.Fatal(err)
	}
	_ = resultIssueID(t, result)
}

func TestAbandonedDraftUploadDoesNotBlockRetryPublication(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	abandoned := publicationAsset(f, draft, f.project)
	completed := publicationAsset(f, draft, f.project)
	f.exec(`UPDATE file_assets SET is_uploaded=false WHERE id=$1`, abandoned)
	result, err := publishDraftForTest(f, repo, draft, f.member, fmt.Sprintf(`{"description_html":"<image-component src=\"%s\"></image-component>"}`, completed))
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, result)
	var private, transferred bool
	if scanErr := f.db.QueryRow(f.ctx, `SELECT draft_issue_id=$2 AND issue_id IS NULL FROM file_assets WHERE id=$1`, abandoned, draft).Scan(&private); scanErr != nil || !private {
		t.Fatalf("abandoned asset exposed: %v %v", private, scanErr)
	}
	if scanErr := f.db.QueryRow(f.ctx, `SELECT issue_id=$2 AND draft_issue_id IS NULL FROM file_assets WHERE id=$1`, completed, issue).Scan(&transferred); scanErr != nil || !transferred {
		t.Fatalf("completed asset lost: %v %v", transferred, scanErr)
	}
}

func TestDraftPublicationDoesNotTransferInjectedAssets(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	existing, err := writeValueItem(f, repo, "POST", "", `{"name":"Existing"}`)
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, existing)
	workspace, comment, page := uuid.NewString(), uuid.NewString(), uuid.NewString()
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Foreign',$2,'foreign','UTC','',now(),now())`, workspace, f.outsider)
	f.exec(`INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id,created_at,updated_at,comment_stripped,comment_json,comment_html,attachments,access) VALUES($1,$2,$3,$4,$5,now(),now(),'','{}','',ARRAY[]::text[],'INTERNAL')`, comment, f.workspace, f.project, issue, f.member)
	f.exec(`INSERT INTO pages(id,workspace_id,name,description_json,description_html,owned_by_id,access,color,is_locked,view_props,logo_props,is_global,sort_order,created_at,updated_at) VALUES($1,$2,'Page','{}','',$3,0,'',false,'{}','{}',false,0,now(),now())`, page, f.workspace, f.member)
	injected := []string{}
	for column, value := range map[string]string{"workspace_id": workspace, "created_by_id": f.outsider, "issue_id": issue, "comment_id": comment, "page_id": page, "user_id": f.member} {
		asset := publicationAsset(f, draft, f.project)
		f.exec(`UPDATE file_assets SET `+column+`=$2 WHERE id=$1`, asset, value)
		injected = append(injected, asset)
	}
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	_ = resultIssueID(t, result)
	for _, asset := range injected {
		var retained bool
		if scanErr := f.db.QueryRow(f.ctx, `SELECT draft_issue_id=$2 FROM file_assets WHERE id=$1`, asset, draft).Scan(&retained); scanErr != nil || !retained {
			t.Fatalf("injected asset transferred: %v %v", retained, scanErr)
		}
	}
}
