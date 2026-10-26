package notionsync

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// apiFixture implements the public collection/detail contract with persisted
// records, so retries and reconciliation are tested through actual HTTP calls.
type apiFixture struct {
	rows          map[string]Object
	creates       int
	patches       int
	failCreate    bool
	normalizeHTML bool
	members       map[string][]string
}

func (a *apiFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Header.Get("X-Api-Key") != "test-key" {
		w.WriteHeader(401)
		return
	}
	if strings.Contains(r.URL.Path, "/module-issues/") {
		if a.members == nil {
			a.members = map[string][]string{}
		}
		switch r.Method {
		case http.MethodGet:
			values := []Object{}
			for _, id := range a.members[r.URL.Path] {
				values = append(values, Object{"id": id})
			}
			_ = json.NewEncoder(w).Encode(values)
		case http.MethodPost:
			var body struct {
				Issues []string `json:"issues"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(400)
				return
			}
			a.members[r.URL.Path] = append(a.members[r.URL.Path], body.Issues...)
			w.WriteHeader(201)
		default:
			w.WriteHeader(405)
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		if object, ok := a.rows[r.URL.Path]; ok {
			_ = json.NewEncoder(w).Encode(object)
			return
		}
		values := []Object{}
		for path, object := range a.rows {
			if strings.TrimSuffix(path, "/") == r.URL.Path+fmt.Sprint(object["id"]) {
				values = append(values, object)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": values, "next_page_results": false})
	case http.MethodPost:
		var body Object
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(400)
			return
		}
		body["id"] = uuid.NewString()
		a.normalizeDescription(body)
		a.creates++
		a.rows[r.URL.Path+fmt.Sprint(body["id"])+"/"] = body
		if a.failCreate {
			a.failCreate = false
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(body)
	case http.MethodPatch:
		object, ok := a.rows[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			return
		}
		var patch Object
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			w.WriteHeader(400)
			return
		}
		a.normalizeDescription(patch)
		maps.Copy(object, patch)
		a.patches++
		_ = json.NewEncoder(w).Encode(object)
	default:
		w.WriteHeader(405)
	}
}

func (a *apiFixture) normalizeDescription(object Object) {
	if value, ok := object["description_html"].(string); a.normalizeHTML && ok {
		object["description_html"] = "<div>" + value + "</div>"
	}
}

func TestServerNormalizedDescriptionIsUnchangedAfterRestart(t *testing.T) {
	runner, api := fixtureRunner(t)
	api.normalizeHTML = true
	ctx := context.Background()
	desired := map[string]any{"name": "Task", "description_html": "<p>Original</p>"}
	id, err := runner.upsert(ctx, "issues", "project", taskID, desired)
	if err != nil {
		t.Fatal(err)
	}
	path := runner.Journal.path
	if err = runner.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	runner.Journal, err = OpenJournal(path, "test-scope")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runner.upsert(ctx, "issues", "project", taskID, desired); err != nil {
		t.Fatal(err)
	}
	if api.patches != 0 || len(runner.report.Conflicts) != 0 {
		t.Fatal("unchanged normalized HTML caused a write or conflict")
	}
	desired["description_html"] = "<p>Source edit</p>"
	if _, err = runner.upsert(ctx, "issues", "project", taskID, desired); err != nil {
		t.Fatal(err)
	}
	if api.patches != 1 {
		t.Fatal("source edit was not applied")
	}
	row := api.rows[runner.Vector.collection("issues", "project")+id+"/"]
	row["description_html"] = "<p>Local edit</p>"
	if _, err = runner.upsert(ctx, "issues", "project", taskID, desired); err != nil {
		t.Fatal(err)
	}
	if api.patches != 1 || len(runner.report.Conflicts) != 1 || row["description_html"] != "<p>Local edit</p>" {
		t.Fatal("local edit was not protected")
	}
}

func fixtureRunner(t *testing.T) (*Runner, *apiFixture) {
	t.Helper()
	api := &apiFixture{rows: map[string]Object{}}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)
	vector, err := NewVector(server.URL, "example", "test-key")
	if err != nil {
		t.Fatal(err)
	}
	vector.api.interval = 0
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "state.json"), "test-scope")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.Close() })
	return &Runner{Vector: vector, Journal: journal, Apply: true}, api
}

func TestRepeatImportUpdatesExistingObjectWithoutDuplicates(t *testing.T) {
	runner, api := fixtureRunner(t)
	ctx := context.Background()
	id, err := runner.upsert(ctx, "project", "", "source", map[string]any{"name": "Original"})
	if err != nil {
		t.Fatal(err)
	}
	again, err := runner.upsert(ctx, "project", "", "source", map[string]any{"name": "Original"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := runner.upsert(ctx, "project", "", "source", map[string]any{"name": "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if id != again || id != updated || api.creates != 1 || api.patches != 1 {
		t.Fatalf("identity/counts: %s %s %s %d %d", id, again, updated, api.creates, api.patches)
	}
	if api.rows[runner.Vector.prefix+id+"/"]["name"] != "Updated" {
		t.Fatal("source edit was not applied")
	}
}

func TestLocalEditIsPreservedAsConflict(t *testing.T) {
	runner, api := fixtureRunner(t)
	id, err := runner.upsert(context.Background(), "project", "", "source", map[string]any{"name": "Original"})
	if err != nil {
		t.Fatal(err)
	}
	api.rows[runner.Vector.prefix+id+"/"]["name"] = "Local edit"
	_, err = runner.upsert(context.Background(), "project", "", "source", map[string]any{"name": "Notion edit"})
	if err != nil {
		t.Fatal(err)
	}
	if api.patches != 0 || api.rows[runner.Vector.prefix+id+"/"]["name"] != "Local edit" || len(runner.report.Conflicts) != 1 {
		t.Fatal("local edit was not preserved and reported")
	}
}

func TestCommittedCreateWithLostResponseRecoversAfterRestart(t *testing.T) {
	runner, api := fixtureRunner(t)
	api.failCreate = true
	_, err := runner.upsert(context.Background(), "project", "", "source", map[string]any{"name": "Original"})
	if err == nil {
		t.Fatal("expected uncertain create error")
	}
	path := runner.Journal.path
	if err = runner.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	runner.Journal, err = OpenJournal(path, "test-scope")
	if err != nil {
		t.Fatal(err)
	}
	id, err := runner.upsert(context.Background(), "project", "", "source", map[string]any{"name": "Original"})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || api.creates != 1 || len(runner.report.Conflicts) != 0 {
		t.Fatal("uncertain create was duplicated or not recovered")
	}
	if runner.Journal.Entries["project:source"].Pending != nil {
		t.Fatal("recovered intent remains pending")
	}
}

func TestPreviewDoesNotWriteDestinationOrJournal(t *testing.T) {
	runner, api := fixtureRunner(t)
	runner.Apply = false
	_, err := runner.upsert(context.Background(), "project", "", "source", map[string]any{"name": "Preview"})
	if err != nil {
		t.Fatal(err)
	}
	if api.creates != 0 || api.patches != 0 || len(runner.Journal.Entries) != 0 {
		t.Fatal("preview changed state")
	}
}

func TestJournalRefusesConcurrentRunnerAndWrongDestination(t *testing.T) {
	runner, _ := fixtureRunner(t)
	if _, err := OpenJournal(runner.Journal.path, "test-scope"); err == nil {
		t.Fatal("second runner acquired the lock")
	}
	if err := runner.Journal.Save(); err != nil {
		t.Fatal(err)
	}
	if err := runner.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenJournal(runner.Journal.path, "different-workspace"); err == nil {
		t.Fatal("journal accepted another destination")
	}
}

func TestAssigneeOrderingDoesNotCauseConflict(t *testing.T) {
	changes, conflicts := fieldChanges(Object{"assignees": []any{"b", "a"}}, map[string]any{"assignees": []string{"a", "b"}}, &Entry{})
	if len(changes) != 0 || len(conflicts) != 0 {
		t.Fatal("set ordering caused a change")
	}
}
