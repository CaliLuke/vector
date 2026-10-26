//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	"github.com/CaliLuke/go-plane/internal/workitems"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func attachValueProperties(t *testing.T, f projectTypeFixture, definitions ...property.Definition) {
	t.Helper()
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, def := range definitions {
		catalog, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: def})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, def.ID)
	}
	_, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: f.task, PropertyIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDraftCreationDefaultsReadBackAndFormPublication(t *testing.T) {
	f := newProjectTypeFixture(t)
	number := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(7)}})
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	result, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_draft_issues", Method: "POST", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(fmt.Sprintf(`{"name":"Draft","project_id":"%s","type_id":"%s"}`, f.project, f.task)), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	reader, err := intakeinboxreads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := uuid.Parse(f.member)
	draft, _ := uuid.Parse(id)
	raw, err := reader.DraftIssueDetail(f.ctx, owner, "types", draft)
	if err != nil {
		t.Fatal(err)
	}
	var read struct {
		Values map[string]any `json:"custom_properties"`
	}
	if err = json.Unmarshal(raw, &read); err != nil || read.Values[number] != float64(7) {
		t.Fatalf("read lost defaults: %s %v", raw, err)
	}
	admin, _ := uuid.Parse(f.owner)
	if _, err = reader.DraftIssueDetail(f.ctx, admin, "types", draft); !errors.Is(err, intakeinboxreads.ErrForbidden) {
		t.Fatalf("administrator read another member's private draft: %v", err)
	}
	form := url.Values{"custom_properties": {fmt.Sprintf(`{"%s":0}`, number)}}.Encode()
	result, err = repo.Execute(f.ctx, workitems.Request{Operation: draftPublishOperation, Method: "POST", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{"draft_id": id}}, strings.NewReader(form), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	if values := draftValues(t, f, "issues", resultIssueID(t, result)); values[number] != float64(0) {
		t.Fatalf("form publication lost explicit zero: %#v", values)
	}
}

func draftValues(t *testing.T, f projectTypeFixture, table, id string) map[string]any {
	t.Helper()
	var raw []byte
	if err := f.db.QueryRow(f.ctx, `SELECT custom_properties FROM `+table+` WHERE id=$1`, id).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	return values
}

func editValueDraft(f projectTypeFixture, repo *workitems.Repository, id, body string) error {
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_draft_issues_by_pk", Method: "PATCH", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{"pk": id}}, strings.NewReader(body), "application/json")
	return err
}

func TestDraftValuesSaveIncompleteAndValidateOnPublication(t *testing.T) {
	f := newProjectTypeFixture(t)
	number, required, member, inactive, boolean := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(7)}},
		property.Definition{ID: required, Title: "Acceptance", Kind: property.Text, Configuration: property.Configuration{Active: true, Required: true, Format: "Single line"}},
		property.Definition{ID: member, Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: inactive, Title: "Historical value", Kind: property.Number, Configuration: property.Configuration{Active: false}},
		property.Definition{ID: boolean, Title: "Verified", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
	)
	repo, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,42) WHERE id=$1`, draft, inactive)
	err := editValueDraft(f, repo, draft, fmt.Sprintf(`{"custom_properties":{"%s":null,"%s":"%s","%s":false}}`, number, member, f.member, boolean))
	if err != nil {
		t.Fatal(err)
	}
	stored := draftValues(t, f, "draft_issues", draft)
	if _, ok := stored[number]; !ok || stored[number] != nil || stored[inactive] != float64(42) || stored[boolean] != false {
		t.Fatalf("draft clear/inactive/false lost: %#v", stored)
	}
	var subscribers int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM issue_subscribers`).Scan(&subscribers); err != nil || subscribers != 0 {
		t.Fatalf("private draft subscribed users: %d %v", subscribers, err)
	}
	_, err = publishDraftForTest(f, repo, draft, f.member, `{}`)
	var validation *djangoapi.HTTPError
	if !errors.As(err, &validation) || validation.Status != 400 {
		t.Fatalf("missing required accepted: %v", err)
	}
	var consumed bool
	if err = f.db.QueryRow(f.ctx, `SELECT deleted_at IS NOT NULL FROM draft_issues WHERE id=$1`, draft).Scan(&consumed); err != nil || consumed {
		t.Fatalf("failed publication consumed draft: %v %v", consumed, err)
	}
	result, err := publishDraftForTest(f, repo, draft, f.member, fmt.Sprintf(`{"custom_properties":{"%s":"Review complete"}}`, required))
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, result)
	values := draftValues(t, f, "issues", issue)
	if values[required] != "Review complete" || values[inactive] != float64(42) || values[member] != f.member || values[boolean] != false {
		t.Fatalf("publication lost merged values: %#v", values)
	}
	if value, ok := values[number]; !ok || value != nil {
		t.Fatalf("explicit clear replaced by default: %#v", values)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM issue_subscribers WHERE issue_id=$1 AND subscriber_id=$2 AND deleted_at IS NULL`, issue, f.member).Scan(&subscribers); err != nil || subscribers != 1 {
		t.Fatalf("publication subscription: %d %v", subscribers, err)
	}
}

func TestDraftValuesRejectInvalidShapesUnknownAndReadonlyEdits(t *testing.T) {
	f := newProjectTypeFixture(t)
	readonly := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: readonly, Title: "Instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Follow the checklist"}})
	repo, draft, _ := seedPublicationDraft(t, f)
	for _, payload := range []string{`null`, `[]`, `"text"`, `1`, fmt.Sprintf(`{"%s":1}`, uuid.New().String()), fmt.Sprintf(`{"%s":"replace"}`, readonly)} {
		err := editValueDraft(f, repo, draft, `{"custom_properties":`+payload+`}`)
		var validation *djangoapi.HTTPError
		if !errors.As(err, &validation) || validation.Status != 400 {
			t.Fatalf("accepted %s: %v", payload, err)
		}
		if values := draftValues(t, f, "draft_issues", draft); len(values) != 0 {
			t.Fatalf("invalid save changed values: %#v", values)
		}
	}
}

func TestDraftTypeClearRequiresPreservation(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	repo, draft, _ := seedPublicationDraft(t, f)
	if err := editValueDraft(f, repo, draft, fmt.Sprintf(`{"custom_properties":{"%s":0}}`, id)); err != nil {
		t.Fatal(err)
	}
	err := editValueDraft(f, repo, draft, `{"project":null,"type_id":null}`)
	var conflict *djangoapi.HTTPError
	if !errors.As(err, &conflict) || conflict.Status != 409 {
		t.Fatalf("type clear discarded value meaning: %v", err)
	}
	values := draftValues(t, f, "draft_issues", draft)
	if values[id] != float64(0) {
		t.Fatalf("failed clear changed draft: %#v", values)
	}
}
