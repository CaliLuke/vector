package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/notionsync"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
)

const exampleID = "00000000-0000-4000-8000-000000000001"

type sampleSource struct{ failure error }

func (s sampleSource) Page(context.Context, string) (notionsync.Page, error) {
	return notionsync.Page{ID: exampleID, Title: "Example", Relations: map[string][]string{"🏗️ Projects": {}}}, s.failure
}

func TestSnapshotImportDoesNotRequireNotionTokenAndCannotPoll(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "")
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.WriteFile(path, []byte(`[{"id":"`+exampleID+`","title":"Example"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := sourceFor(options{snapshot: path})
	if err != nil {
		t.Fatal(err)
	}
	page, err := source.Page(context.Background(), exampleID)
	if err != nil || page.Title != "Example" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	if _, err := sourceFor(options{snapshot: path, interval: 5 * time.Minute}); err == nil {
		t.Fatal("stale snapshot polling accepted")
	}
	if _, err := sourceFor(options{snapshot: filepath.Join(t.TempDir(), "missing.json")}); err == nil {
		t.Fatal("missing snapshot accepted")
	}
}

func TestTeamPreviewUsesItsOwnScopeWithoutWriting(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "")
	t.Setenv("VECTOR_API_KEY", "test-key")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/example/projects/" {
			t.Errorf("unexpected preview request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	const roadmapID = "00000000-0000-4000-8000-000000000002"
	data, err := json.Marshal([]notionsync.Page{
		{ID: exampleID, Title: "Platform App"},
		{ID: roadmapID, Title: "Roadmap", Relations: map[string][]string{"👷 Team": {exampleID}, "🏗️ Projects": {}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pages.json")
	if err = os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	o := options{team: exampleID, snapshot: path, workspace: "example", origin: server.URL,
		state: filepath.Join(t.TempDir(), "state.json"), requestInterval: time.Millisecond}
	if err = execute(context.Background(), o); err != nil || requests != 1 {
		t.Fatalf("team preview: requests=%d err=%v", requests, err)
	}
	journal, err := notionsync.OpenJournal(o.state, server.URL+"|example|team:"+exampleID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Close(t, journal) }()
	if len(journal.Entries) != 0 {
		t.Fatal("preview recorded writes")
	}
}

func TestPreviewAndSourceFailureNeverWrite(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Errorf("preview attempted %s", r.Method)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	vector, err := notionsync.NewVector(server.URL, "example", "test-key")
	if err != nil {
		t.Fatal(err)
	}
	o := options{roadmap: exampleID, state: filepath.Join(t.TempDir(), "state.json")}
	if err = syncOnce(context.Background(), o, sampleSource{}, vector, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("preview requests: %d", requests)
	}
	o.apply = true
	if err = syncOnce(context.Background(), o, sampleSource{errors.New("source inaccessible")}, vector, nil); err == nil || requests != 1 {
		t.Fatal("source failure did not stop import")
	}
}

func TestInvalidConfigurationFailsBeforeNetworkAccess(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "")
	t.Setenv("VECTOR_API_KEY", "")
	cases := []options{
		{},
		{team: exampleID},
		{team: exampleID, roadmap: exampleID, snapshot: "snapshot.json"},
		{team: exampleID, snapshot: "snapshot.json", parents: true},
		{team: exampleID, snapshot: "snapshot.json", roadmapOnly: true},
		{roadmap: exampleID, interval: time.Minute, apply: true},
		{roadmap: exampleID, interval: 5 * time.Minute},
		{roadmap: exampleID},
	}
	for _, o := range cases {
		if execute(context.Background(), o) == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	t.Setenv("NOTION_TOKEN", "test-token")
	if execute(context.Background(), options{roadmap: exampleID, origin: "http://remote.example", workspace: "example"}) == nil {
		t.Fatal("insecure origin accepted")
	}
	t.Setenv("VECTOR_API_KEY", "test-key")
	o := options{roadmap: exampleID, origin: "http://localhost:8000", workspace: "example", users: filepath.Join(t.TempDir(), "missing.json")}
	if execute(context.Background(), o) == nil {
		t.Fatal("missing mapping accepted")
	}
}

func TestCanceledRoadmapImportStopsBeforeNetworkAccess(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")
	t.Setenv("VECTOR_API_KEY", "test-key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	o := options{roadmap: exampleID, origin: "http://localhost:8000", workspace: "example",
		state: filepath.Join(t.TempDir(), "state.json"), roadmapOnly: true, apply: true}
	if err := execute(ctx, o); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled import: %v", err)
	}
}
