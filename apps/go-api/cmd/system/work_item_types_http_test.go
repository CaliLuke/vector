package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

type typeHTTPRepository struct {
	releaseInput                                    store.ReleaseInput
	itemSourceMode, itemSourceID, itemSourceProject string
	typeRemovalInput                                store.TypeRemovalInput
	editError                                       error
	hierarchyInput                                  store.HierarchyInput
	itemEditInput                                   workitems.ItemEditInput
	formulaInput                                    store.FormulaAuthoringInput
	schemaInput                                     store.ItemSchemaInput
	calls                                           int
	actor, slug                                     string
}

func (r *typeHTTPRepository) Catalog(_ context.Context, actor, slug string) (store.Catalog, error) {
	r.calls++
	r.actor = actor
	r.slug = slug
	return store.Catalog{Types: []store.Type{}}, nil
}
func (r *typeHTTPRepository) Mutate(_ context.Context, actor, slug, _, _ string, _ store.Input) (store.Catalog, error) {
	r.calls++
	r.actor = actor
	r.slug = slug
	return store.Catalog{Enabled: true, Types: []store.Type{}}, nil
}

func (r *typeHTTPRepository) ProjectCatalog(ctx context.Context, actor, slug, project string) (store.ProjectCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.ProjectCatalog{Enabled: true, CanManage: true, Revision: 7, ProjectName: project, Types: []store.ProjectType{}}, err
}
func (r *typeHTTPRepository) MutateProject(ctx context.Context, actor, slug, project string, input store.ProjectInput) (store.ProjectCatalog, error) {
	return r.ProjectCatalog(ctx, actor, slug, project)
}

type typeHTTPSessions struct{}

func (typeHTTPSessions) Validate(_ context.Context, credential string) (djangosession.Principal, error) {
	if credential != "types-session" {
		return djangosession.Principal{}, errors.New("invalid session")
	}
	return djangosession.Principal{UserID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}, nil
}
func TestWorkItemTypesHTTPProtection(t *testing.T) {
	const releaseBody = `{"revision":4,"action":"create","id":"eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee","values":{"name":"October","description":"Notes","target_date":"2026-10-31"}}`
	const removalBody = `{"revision":4,"type_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","replacement_id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","review_id":"dddddddd-dddd-4ddd-8ddd-dddddddddddd"}`
	const editBody = `{"expected_source_version":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","revision":4,"mode":"draft","project_id":"","item_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","changes":{"custom_properties":{"zero":0,"flag":false,"clear":null,"large":9007199254740993}},"review_id":"dddddddd-dddd-4ddd-8ddd-dddddddddddd"}`
	const restoreBody = `{"expected_source_version":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","revision":4,"mode":"edit","project_id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","item_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","changes":{},"restore":{"property_id":"eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee","version_id":"ffffffff-ffff-4fff-8fff-ffffffffffff"},"review_id":"dddddddd-dddd-4ddd-8ddd-dddddddddddd"}`

	const hierarchyBody = `{"revision":4,"default_level":0,"placements":[{"type_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","level":1}],"review_id":"dddddddd-dddd-4ddd-8ddd-dddddddddddd","confirm":"confirm"}`
	const formulaBody = `{"revision":4,"type_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","definition":{"id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","title":"Calculation","description":"","icon":"","name":"calculation","kind":"Formula","configuration":{"active":true,"required":false,"multiple":false,"format":"","content":"","expression":"1-1","result":"Number","options":[],"levels":[]},"contexts":[]}}`
	for _, test := range []struct {
		name, method, path, body, session, csrf, origin, project, resource string
		editError                                                          error
		status                                                             int
		called                                                             bool
		typePresent, parentPresent                                         bool
		selectedType, selectedParent                                       string
	}{
		{name: "anonymous release catalog", method: "GET", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", status: 401},
		{name: "release catalog", method: "GET", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", session: "types-session", status: 200, called: true},
		{name: "anonymous release mutation", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", body: releaseBody, csrf: strings.Repeat("a", 32), status: 401},
		{name: "release mutation requires csrf", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", body: releaseBody, session: "types-session", status: 403},
		{name: "release mutation foreign origin", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", body: releaseBody, session: "types-session", csrf: strings.Repeat("a", 32), origin: "http://evil.example", status: 403},

		{name: "anonymous property identities", method: "GET", resource: "work-item-property-identities", status: 401},
		{name: "workspace property identities", method: "GET", resource: "work-item-property-identities", session: "types-session", status: 200, called: true},
		{name: "project property identities", method: "GET", resource: "work-item-property-identities", path: "?project_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", session: "types-session", status: 200, called: true},
		{name: "invalid property identity project", method: "GET", resource: "work-item-property-identities", path: "?project_id=bad", session: "types-session", status: 400},
		{name: "anonymous type identities", method: "GET", resource: "work-item-type-identities", status: 401},
		{name: "workspace type identities", method: "GET", resource: "work-item-type-identities", session: "types-session", status: 200, called: true},
		{name: "project type identities", method: "GET", resource: "work-item-type-identities", path: "?project_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", session: "types-session", status: 200, called: true},
		{name: "invalid type identity project", method: "GET", resource: "work-item-type-identities", path: "?project_id=bad", session: "types-session", status: 400},
		{name: "release mutation", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", body: releaseBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "release invalid identifier", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", body: strings.Replace(releaseBody, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", "bad", 1), session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "release invalid action", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", body: strings.Replace(releaseBody, "create", "ship", 1), session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "release missing values", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", resource: "work-item-releases", body: `{"revision":4,"action":"create","id":"eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "anonymous type removal", method: "POST", path: "removal/preview/", body: removalBody, csrf: strings.Repeat("a", 32), status: 401},
		{name: "type removal needs csrf", method: "POST", path: "removal/commit/", body: removalBody, session: "types-session", status: 403},
		{name: "type removal foreign origin", method: "POST", path: "removal/preview/", body: removalBody, session: "types-session", csrf: strings.Repeat("a", 32), origin: "http://evil.example", status: 403},
		{name: "type removal preview", method: "POST", path: "removal/preview/", body: removalBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "type removal commit", method: "POST", path: "removal/commit/", body: removalBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "type removal retry marker", method: "POST", path: "removal/commit/", body: removalBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 409, called: true, editError: &store.Error{Status: 409, Message: "Review changed", ReviewAgain: true}},
		{name: "type removal private failure", method: "POST", path: "removal/commit/", body: removalBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 500, called: true, editError: errors.New("private database detail")},

		{name: "anonymous edit source", method: "GET", resource: "work-item-edits", path: "source/?mode=edit&item_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb&project_id=cccccccc-cccc-4ccc-8ccc-cccccccccccc", status: 401},
		{name: "canonical edit source", method: "GET", resource: "work-item-edits", path: "source/?mode=edit&item_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb&project_id=cccccccc-cccc-4ccc-8ccc-cccccccccccc", session: "types-session", status: 200, called: true},
		{name: "canonical draft source", method: "GET", resource: "work-item-edits", path: "source/?mode=draft&item_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", session: "types-session", status: 200, called: true},
		{name: "invalid source mode", method: "GET", resource: "work-item-edits", path: "source/?mode=other&item_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", session: "types-session", status: 400},
		{name: "anonymous item edit", method: "POST", resource: "work-item-edits", path: "preview/", body: editBody, csrf: strings.Repeat("a", 32), status: 401},
		{name: "item edit needs csrf", method: "POST", resource: "work-item-edits", path: "commit/", body: editBody, session: "types-session", status: 403},
		{name: "item edit foreign origin", method: "POST", resource: "work-item-edits", path: "preview/", body: editBody, session: "types-session", csrf: strings.Repeat("a", 32), origin: "http://evil.example", status: 403},
		{name: "item edit preview", method: "POST", resource: "work-item-edits", path: "preview/", body: editBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "item edit commit", method: "POST", resource: "work-item-edits", path: "commit/", body: editBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "property restore preview", method: "POST", resource: "work-item-edits", path: "preview/", body: restoreBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "property restore commit", method: "POST", resource: "work-item-edits", path: "commit/", body: restoreBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "property restore invalid version", method: "POST", resource: "work-item-edits", path: "preview/", body: strings.Replace(restoreBody, "ffffffff-ffff-4fff-8fff-ffffffffffff", "bad", 1), session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "invalid item edit source", method: "POST", resource: "work-item-edits", path: "preview/", body: strings.Replace(editBody, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "bad", 1), session: "types-session", csrf: strings.Repeat("a", 32), status: 400},

		{name: "item edit field errors", method: "POST", resource: "work-item-edits", path: "commit/", body: editBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 400, called: true, editError: &djangoapi.HTTPError{Status: 400, Body: map[string]any{"custom_properties": map[string]any{"effort": "Enter a number."}}}},
		{name: "item edit database privacy", method: "POST", resource: "work-item-edits", path: "preview/", body: editBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 500, called: true, editError: errors.New("private database detail")},

		{name: "anonymous hierarchy", method: "GET", resource: "work-item-hierarchy", status: 401},
		{name: "hierarchy catalog", method: "GET", resource: "work-item-hierarchy", session: "types-session", status: 200, called: true},
		{name: "hierarchy review needs session", method: "POST", resource: "work-item-hierarchy", path: "preview/", body: hierarchyBody, csrf: strings.Repeat("a", 32), status: 401},
		{name: "hierarchy review needs csrf", method: "POST", resource: "work-item-hierarchy", path: "preview/", body: hierarchyBody, session: "types-session", status: 403},
		{name: "hierarchy review", method: "POST", resource: "work-item-hierarchy", path: "preview/", body: hierarchyBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "hierarchy commit", method: "POST", resource: "work-item-hierarchy", body: hierarchyBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "hierarchy commit needs csrf", method: "POST", resource: "work-item-hierarchy", body: hierarchyBody, session: "types-session", status: 403},
		{name: "hierarchy foreign origin", method: "POST", resource: "work-item-hierarchy", body: hierarchyBody, session: "types-session", csrf: strings.Repeat("a", 32), origin: "http://evil.example", status: 403},
		{name: "hierarchy missing placements", method: "POST", resource: "work-item-hierarchy", body: `{"revision":4,"default_level":0}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "hierarchy invalid level", method: "POST", resource: "work-item-hierarchy", path: "preview/", body: strings.Replace(hierarchyBody, `"level":1`, `"level":21`, 1), session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "hierarchy invalid receipt", method: "POST", resource: "work-item-hierarchy", body: strings.Replace(hierarchyBody, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "bad", 1), session: "types-session", csrf: strings.Repeat("a", 32), status: 400},

		{name: "anonymous formula preview", method: "POST", resource: "work-item-properties", path: "formula-preview/", body: formulaBody, csrf: strings.Repeat("a", 32), status: 401},
		{name: "formula preview needs csrf", method: "POST", resource: "work-item-properties", path: "formula-preview/", body: formulaBody, session: "types-session", status: 403},
		{name: "formula preview foreign origin", method: "POST", resource: "work-item-properties", path: "formula-preview/", body: formulaBody, session: "types-session", csrf: strings.Repeat("a", 32), origin: "http://evil.example", status: 403},
		{name: "formula preview", method: "POST", resource: "work-item-properties", path: "formula-preview/", body: formulaBody, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "formula preview requires definition", method: "POST", resource: "work-item-properties", path: "formula-preview/", body: `{"revision":4,"type_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "anonymous schema preview", method: "POST", resource: "work-item-schema", path: "preview/?mode=draft", body: `{}`, csrf: strings.Repeat("a", 32), status: 401},
		{name: "schema preview needs csrf", method: "POST", resource: "work-item-schema", path: "preview/?mode=draft", body: `{}`, session: "types-session", status: 403},
		{name: "schema preview retains scalar values", method: "POST", resource: "work-item-schema", path: "preview/?mode=draft", body: `{"custom_properties":{"zero":0,"flag":false,"empty":null}}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "anonymous item schema", method: "GET", resource: "work-item-schema", path: "?mode=draft", status: 401},
		{name: "schema omitted type", method: "GET", resource: "work-item-schema", path: "?mode=draft", session: "types-session", status: 200, called: true},
		{name: "schema cleared type", method: "GET", resource: "work-item-schema", path: "?mode=draft&type_id=", session: "types-session", status: 200, called: true, typePresent: true},
		{name: "schema selected type", method: "GET", resource: "work-item-schema", path: "?mode=draft&type_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", session: "types-session", status: 200, called: true, typePresent: true, selectedType: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"},
		{name: "schema cleared parent", method: "GET", resource: "work-item-schema", path: "?mode=draft&parent_id=", session: "types-session", status: 200, called: true, parentPresent: true},
		{name: "schema selected parent", method: "GET", resource: "work-item-schema", path: "?mode=draft&parent_id=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", session: "types-session", status: 200, called: true, parentPresent: true, selectedParent: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"},
		{name: "schema invalid parent", method: "GET", resource: "work-item-schema", path: "?mode=draft&parent_id=bad", session: "types-session", status: 400},
		{name: "schema invalid type", method: "GET", resource: "work-item-schema", path: "?mode=draft&type_id=bad", session: "types-session", status: 400},
		{name: "schema invalid mode", method: "GET", resource: "work-item-schema", path: "?mode=bad", session: "types-session", status: 400},
		{name: "schema missing mode", method: "GET", resource: "work-item-schema", session: "types-session", status: 400},
		{name: "anonymous removal review", method: "POST", resource: "work-item-properties", path: "removal-preview/", body: `{"revision":4,"action":"delete","id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}`, csrf: strings.Repeat("a", 32), status: 401},
		{name: "removal review needs csrf", method: "POST", resource: "work-item-properties", path: "removal-preview/", body: `{"revision":4,"action":"delete","id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}`, session: "types-session", status: 403},
		{name: "removal review", method: "POST", resource: "work-item-properties", path: "removal-preview/", body: `{"revision":4,"action":"delete","id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "invalid removal review receipt", method: "POST", resource: "work-item-properties", body: `{"revision":4,"action":"delete","id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","review_id":"invalid"}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "anonymous property catalog", method: "GET", resource: "work-item-properties", status: 401},
		{name: "property catalog", method: "GET", resource: "work-item-properties", session: "types-session", status: 200, called: true},
		{name: "property mutation needs csrf", method: "POST", resource: "work-item-properties", body: `{"revision":4,"action":"attach","type_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","property_ids":[]}`, session: "types-session", status: 403},
		{name: "property attachment", method: "POST", resource: "work-item-properties", body: `{"revision":4,"action":"attach","type_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","property_ids":[]}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "incomplete property command", method: "POST", resource: "work-item-properties", body: `{"revision":4,"action":"create"}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "project catalog", method: "GET", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", session: "types-session", status: 200, called: true},
		{name: "anonymous project catalog", method: "GET", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", status: 401},
		{name: "project mutation needs csrf", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", body: `{"revision":7,"action":"import","type_ids":["cccccccc-cccc-4ccc-8ccc-cccccccccccc"]}`, session: "types-session", status: 403},
		{name: "project import", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", body: `{"revision":7,"action":"import","type_ids":["cccccccc-cccc-4ccc-8ccc-cccccccccccc"]}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "invalid project id", method: "GET", project: "not-a-uuid", session: "types-session", status: 400},
		{name: "invalid imported id", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", body: `{"revision":7,"action":"import","type_ids":["bad"]}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "automatic project default", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", body: `{"revision":7,"action":"set_default","type_ids":[]}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "empty import", method: "POST", project: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", body: `{"revision":7,"action":"import","type_ids":[]}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "anonymous catalog", method: "GET", status: 401},
		{name: "authenticated catalog", method: "GET", session: "types-session", status: 200, called: true},
		{name: "missing csrf", method: "POST", path: "enable/", body: `{"revision":0}`, session: "types-session", status: 403},
		{name: "foreign origin", method: "POST", path: "enable/", body: `{"revision":0}`, session: "types-session", csrf: strings.Repeat("a", 32), origin: "http://evil.example", status: 403},
		{name: "validated enable", method: "POST", path: "enable/", body: `{"revision":0}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "catalog icon", method: "POST", body: `{"revision":0,"name":"Infrastructure","description":"","icon":"Database","color":"#02b55c","active":false,"mandatory":false}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 200, called: true},
		{name: "invalid icon identifier", method: "POST", body: `{"revision":0,"name":"Infrastructure","description":"","icon":"<svg>","color":"#02b55c","active":false,"mandatory":false}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
		{name: "invalid definition", method: "POST", body: `{"revision":0,"name":"Bug"}`, session: "types-session", csrf: strings.Repeat("a", 32), status: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &typeHTTPRepository{editError: test.editError}
			service, err := systemapi.NewWorkItemTypes(repo, typeHTTPSessions{}, repo)
			if err != nil {
				t.Fatal(err)
			}
			handler := newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{workItemTypes: api.NewEndpoints(service), activityReadsBodyLimit: 1 << 20})
			base := "http://example.test/api/workspaces/acme/"
			if test.project != "" {
				base += "projects/" + test.project + "/"
			}
			resource := test.resource
			if resource == "" {
				resource = "work-item-types"
			}
			request := httptest.NewRequest(test.method, base+resource+"/"+test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			if test.session != "" {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: test.session})
			}
			if test.csrf != "" {
				request.AddCookie(&http.Cookie{Name: "csrftoken", Value: test.csrf})
				request.Header.Set("X-CSRFToken", test.csrf)
			}
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.status, response.Body.String())
			}
			if test.resource == "work-item-releases" && test.called && test.method == "POST" {
				input := repo.releaseInput
				if input.Revision != 4 || input.Action != "create" || input.ID != "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee" || input.Values.Name != "October" || input.Values.Description != "Notes" || input.Values.TargetDate != "2026-10-31" {
					t.Fatalf("release mutation contract lost: %+v", input)
				}
			}
			if strings.HasPrefix(test.name, "property restore") && test.called {
				input := repo.itemEditInput
				if input.Restore == nil || input.Restore.PropertyID != "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee" || input.Restore.VersionID != "ffffffff-ffff-4fff-8fff-ffffffffffff" || len(input.Changes) != 0 || input.ExpectedSourceVersion != strings.Repeat("a", 64) || input.Mode != "edit" {
					t.Fatalf("restore contract lost: %+v", input)
				}
			}
			if test.resource == "work-item-edits" && test.called && test.method == "POST" && !strings.HasPrefix(test.name, "property restore") {
				input := repo.itemEditInput
				values := input.Changes["custom_properties"].(map[string]any)
				if input.ExpectedSourceVersion != strings.Repeat("a", 64) || input.Mode != "draft" || input.ProjectID != "" || input.Revision != 4 || values["zero"] != json.Number("0") || values["flag"] != false || values["large"] != json.Number("9007199254740993") {
					t.Fatalf("edit contract lost values: %+v", input)
				}
				if value, present := values["clear"]; !present || value != nil {
					t.Fatal("explicit clear lost")
				}
			}
			if strings.HasPrefix(test.name, "canonical") {
				if repo.itemSourceID != "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" || !strings.Contains(response.Body.String(), `"source_version":"saved-version"`) {
					t.Fatalf("source contract lost: %s", response.Body.String())
				}
				if test.name == "canonical edit source" && (repo.itemSourceMode != "edit" || repo.itemSourceProject != "cccccccc-cccc-4ccc-8ccc-cccccccccccc") {
					t.Fatal("edit source query params lost")
				}
				if test.name == "canonical draft source" && (repo.itemSourceMode != "draft" || repo.itemSourceProject != "") {
					t.Fatal("draft source query params lost")
				}
			}
			if strings.Contains(response.Body.String(), "private database detail") {
				t.Fatal("private error leaked")
			}
			if strings.HasPrefix(test.path, "removal/") && test.called {
				input := repo.typeRemovalInput
				if input.Revision != 4 || input.TypeID != "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" || input.ReplacementID != "cccccccc-cccc-4ccc-8ccc-cccccccccccc" || input.ReviewID != "dddddddd-dddd-4ddd-8ddd-dddddddddddd" {
					t.Fatalf("replacement command lost: %+v", input)
				}
				if test.name == "type removal retry marker" && !strings.Contains(response.Body.String(), `"review_again":true`) {
					t.Fatalf("safe retry marker missing: %s", response.Body.String())
				}
			}
			if test.name == "item edit field errors" && !strings.Contains(response.Body.String(), `"effort":"Enter a number."`) {
				t.Fatalf("structured field error lost: %s", response.Body.String())
			}
			if test.resource == "work-item-schema" && test.called {
				var body struct {
					StoredValues  map[string]any `json:"stored_values"`
					ItemUpdatedAt string         `json:"item_updated_at"`
					Computed      map[string]struct {
						Value any `json:"value"`
					} `json:"computed"`
				}
				if err = json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Computed["zero"].Value != float64(0) || body.Computed["flag"].Value != false {
					t.Fatalf("computed scalar values lost: %s %v", response.Body.String(), err)
				}
				if body.StoredValues["zero"] != float64(0) || body.StoredValues["flag"] != false || body.ItemUpdatedAt == "" {
					t.Fatalf("saved snapshot lost: %s", response.Body.String())
				}
				if test.name == "schema preview retains scalar values" {
					values := repo.schemaInput.Preview.Values
					if value, exists := values["empty"]; !exists || value != nil || values["zero"] != float64(0) || values["flag"] != false {
						t.Fatalf("preview input values lost: %#v", values)
					}
				}

				if (repo.schemaInput.ParentID != nil) != test.parentPresent || repo.schemaInput.ParentID != nil && *repo.schemaInput.ParentID != test.selectedParent {
					t.Fatalf("parent presence/value lost: %+v", repo.schemaInput)
				}
				if (repo.schemaInput.TypeID != nil) != test.typePresent || repo.schemaInput.TypeID != nil && *repo.schemaInput.TypeID != test.selectedType {
					t.Fatalf("type presence/value lost: %+v", repo.schemaInput)
				}
			}

			if test.name == "hierarchy review" || test.name == "hierarchy commit" {
				input := repo.hierarchyInput
				if input.Revision != 4 || input.DefaultLevel != 0 || input.ReviewID != "dddddddd-dddd-4ddd-8ddd-dddddddddddd" || input.Confirm != "confirm" || len(input.Placements) != 1 || input.Placements[0].TypeID != "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" || input.Placements[0].Level != 1 {
					t.Fatalf("hierarchy command changed: %+v", input)
				}
				if test.name == "hierarchy review" {
					var result store.HierarchyReview
					if err = json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Links != 1 || result.RestrictedLinks != 1 || result.Pairs == nil {
						t.Fatalf("hierarchy impact serialization: %s %v", response.Body.String(), err)
					}
				}
			}
			if test.name == "formula preview" {
				if repo.formulaInput.TypeID != "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" || repo.formulaInput.Revision != 4 || repo.formulaInput.Definition.Name != "calculation" {
					t.Fatalf("formula authoring input lost: %+v", repo.formulaInput)
				}
				var body struct {
					Valid  bool `json:"valid"`
					Result struct {
						Value any `json:"value"`
					} `json:"result"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || !body.Valid || body.Result.Value != float64(0) {
					t.Fatalf("formula preview result lost: %s %v", response.Body.String(), err)
				}
			}
			if (repo.calls > 0) != test.called {
				t.Fatalf("protected operation dispatch=%d", repo.calls)
			}
			if test.called && (repo.actor != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" || repo.slug != "acme") {
				t.Fatalf("wrong request scope: %s %s", repo.actor, repo.slug)
			}
		})
	}
}

func (r *typeHTTPRepository) PropertyCatalog(ctx context.Context, actor, slug string) (store.PropertyCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.PropertyCatalog{Revision: 4, CanManage: true, Properties: []store.PropertyRecord{}, Attachments: []store.PropertyAttachment{}, Projects: []store.PropertyChoice{}, Members: []store.PropertyChoice{}}, err
}
func (r *typeHTTPRepository) MutateProperty(ctx context.Context, actor, slug string, input store.PropertyInput) (store.PropertyCatalog, error) {
	return r.PropertyCatalog(ctx, actor, slug)
}
func (r *typeHTTPRepository) PreviewPropertyRemoval(ctx context.Context, actor, slug string, input store.PropertyInput) (store.PropertyReview, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.PropertyReview{ID: "review", Items: 2, Types: 1, Projects: 1}, err
}

func (r *typeHTTPRepository) ItemSchema(ctx context.Context, actor, slug string, input store.ItemSchemaInput) (store.ItemSchema, error) {
	_, err := r.Catalog(ctx, actor, slug)
	r.schemaInput = input
	return store.ItemSchema{StoredValues: map[string]any{"zero": float64(0), "flag": false}, ItemUpdatedAt: new(time.Date(2026, 10, 25, 12, 34, 56, 0, time.UTC)), Computed: map[string]store.FormulaResult{"zero": {Kind: "Number", Status: "value", Value: float64(0)}, "flag": {Kind: "Boolean", Status: "value", Value: false}}, Enabled: true, Revision: 7, Properties: []store.ResolvedProperty{}, Members: []store.ItemReferenceChoice{}, Releases: []store.ItemReferenceChoice{}}, err
}

func (r *typeHTTPRepository) PreviewFormula(ctx context.Context, actor, slug string, input store.FormulaAuthoringInput) (store.FormulaAuthoringResult, error) {
	_, err := r.Catalog(ctx, actor, slug)
	r.formulaInput = input
	return store.FormulaAuthoringResult{Valid: true, Active: true, Contexts: []store.PropertyChoice{}, Result: &store.FormulaResult{Kind: "Number", Status: "value", Value: float64(0)}}, err
}

func (r *typeHTTPRepository) HierarchyCatalog(ctx context.Context, actor, slug string) (store.HierarchyCatalog, error) {
	catalog, err := r.Catalog(ctx, actor, slug)
	return store.HierarchyCatalog{Catalog: catalog, HierarchyEnabled: true, DefaultLevel: 1}, err
}
func (r *typeHTTPRepository) PreviewHierarchy(ctx context.Context, actor, slug string, input store.HierarchyInput) (store.HierarchyReview, error) {
	r.hierarchyInput = input
	_, err := r.Catalog(ctx, actor, slug)
	return store.HierarchyReview{ID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", Links: 1, RestrictedLinks: 1, Pairs: []store.HierarchyPair{}}, err
}
func (r *typeHTTPRepository) CommitHierarchy(ctx context.Context, actor, slug string, input store.HierarchyInput) (store.HierarchyCatalog, error) {
	r.hierarchyInput = input
	return r.HierarchyCatalog(ctx, actor, slug)
}

func (r *typeHTTPRepository) PreviewItemChange(_ context.Context, q workitems.Request, input workitems.ItemEditInput) (workitems.Result, error) {
	r.calls++
	r.actor, r.slug, r.itemEditInput = q.ActorID, q.Slug, input
	return workitems.Result{Status: 200, Body: workitems.ItemEditReview{ID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", PreservedProperties: 0, Description: "<p>Edited</p>"}}, r.editError
}
func (r *typeHTTPRepository) CommitItemChange(ctx context.Context, q workitems.Request, input workitems.ItemEditInput) (workitems.Result, error) {
	return r.PreviewItemChange(ctx, q, input)
}

func (r *typeHTTPRepository) PreviewTypeRemoval(ctx context.Context, actor, slug string, input store.TypeRemovalInput) (store.TypeRemovalReview, error) {
	r.typeRemovalInput = input
	_, err := r.Catalog(ctx, actor, slug)
	return store.TypeRemovalReview{ID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", Items: 2, Drafts: 1, Preserved: 1}, err
}
func (r *typeHTTPRepository) CommitTypeRemoval(ctx context.Context, actor, slug string, input store.TypeRemovalInput) (store.Catalog, error) {
	r.typeRemovalInput = input
	out, err := r.Catalog(ctx, actor, slug)
	if r.editError != nil {
		return out, r.editError
	}
	return out, err
}

func (r *typeHTTPRepository) ReadItemEditSource(_ context.Context, q workitems.Request, mode, item, project string) (workitems.Result, error) {
	r.calls++
	r.actor = q.ActorID
	r.slug = q.Slug
	r.itemSourceMode = mode
	r.itemSourceID = item
	r.itemSourceProject = project
	return workitems.Result{Status: 200, Body: workitems.ItemEditSource{Item: map[string]any{"id": item}, SourceVersion: "saved-version"}}, r.editError
}

func (r *typeHTTPRepository) TypeIdentities(ctx context.Context, actor, slug, project string) (store.TypeIdentityCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.TypeIdentityCatalog{Enabled: true, Revision: 7, Types: []store.TypeIdentity{{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Name: "Retained", Icon: "task", Color: "#123456", Active: false}}}, err
}

func (r *typeHTTPRepository) DiscoveryProperties(ctx context.Context, actor, slug, project string) (store.PropertyDiscoveryCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.PropertyDiscoveryCatalog{Enabled: true, Revision: 7, Fields: []store.DiscoveryProperty{}}, err
}
