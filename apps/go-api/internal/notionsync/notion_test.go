package notionsync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotionReaderPaginatesRelationsAndEscapesContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Notion-Version") != "2026-03-11" {
			t.Error("missing source authorization or version")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/pages/" + rootID:
			_, _ = w.Write([]byte(`{"id":"` + rootID + `","parent":{"data_source_id":"root-source"},"properties":{"Name":{"type":"title","title":[{"plain_text":"Roadmap"}]},"🏗️ Projects":{"id":"a%3Ab","type":"relation"}}}`))
		case "/data_sources/root-source":
			_, _ = w.Write([]byte(`{"properties":{"🏗️ Projects":{"relation":{"data_source_id":"project-source"}}}}`))
		case "/data_sources/project-source":
			_, _ = w.Write([]byte(`{}`))
		case "/pages/" + rootID + "/properties/a:b":
			if r.URL.Query().Get("start_cursor") == "" {
				if r.URL.Query().Has("start_cursor") {
					t.Error("empty start_cursor must be omitted")
				}
				_, _ = w.Write([]byte(`{"results":[{"relation":{"id":"` + moduleA + `"}}],"has_more":true,"next_cursor":"second"}`))
			} else {
				_, _ = w.Write([]byte(`{"results":[{"relation":{"id":"` + moduleB + `"}}],"has_more":false}`))
			}
		case "/blocks/" + rootID + "/children":
			_, _ = w.Write([]byte(`{"results":[{"id":"block","type":"paragraph","paragraph":{"rich_text":[{"plain_text":"<script>alert(1)</script>","text":{"link":{"url":"javascript:alert(1)"}}}]}},{"id":"image-block","type":"image","image":{}}],"has_more":false}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	notion, err := NewNotion("test-token", true)
	if err != nil {
		t.Fatal(err)
	}
	notion.api.base = server.URL
	notion.api.interval = 0
	page, err := notion.Page(context.Background(), rootID)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Relations["🏗️ Projects"]) != 2 {
		t.Fatal("relation pagination lost rows")
	}
	if strings.Contains(page.HTML, "<script>") || strings.Contains(page.HTML, "javascript:") || !strings.Contains(page.HTML, "&lt;script&gt;") {
		t.Fatal("unsafe source markup rendered")
	}
	if len(page.Warnings) != 1 {
		t.Fatal("unsupported block not reported")
	}
}

func TestNotionReaderRejectsInaccessibleRelationDatabase(t *testing.T) {
	databaseReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/data_sources/") {
			databaseReads++
		}
		switch r.URL.Path {
		case "/pages/" + rootID:
			_, _ = w.Write([]byte(`{"id":"` + rootID + `","parent":{"data_source_id":"root-source"},"properties":{"🏗️ Projects":{"id":"r","type":"relation"}}}`))
		case "/data_sources/root-source":
			_, _ = w.Write([]byte(`{"properties":{"🏗️ Projects":{"relation":{"data_source_id":"private-source"}}}}`))
		case "/blocks/" + rootID + "/children":
			_, _ = w.Write([]byte(`{"results":[],"has_more":false}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	notion, err := NewNotion("test-token", true)
	if err != nil {
		t.Fatal(err)
	}
	notion.api.base = server.URL
	notion.api.interval = 0
	if _, err = notion.Page(context.Background(), rootID); err == nil {
		t.Fatal("missing database access was treated as an empty selection")
	}
	reads := databaseReads
	notion.followRelations = false
	graph, err := Discover(context.Background(), notion, rootID, false, true)
	if err != nil || graph.Roadmap.ID != rootID || len(graph.Modules) != 0 || len(graph.Tasks) != 0 {
		t.Fatalf("roadmap-only import: %+v %v", graph, err)
	}
	if databaseReads != reads {
		t.Fatal("roadmap-only import read a related database")
	}
}

func TestVectorListFollowsCursorAndAcceptsArrayEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/array/" {
			_, _ = w.Write([]byte(`[{"id":"array-row"}]`))
			return
		}
		if r.URL.Query().Get("cursor") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []Object{{"id": "first"}}, "next_page_results": true, "next_cursor": "100:1:0"})
		} else {
			_, _ = w.Write([]byte(`{"results":[{"id":"second"}],"next_page_results":false}`))
		}
	}))
	defer server.Close()
	v, err := NewVector(server.URL, "example", "key")
	if err != nil {
		t.Fatal(err)
	}
	v.api.interval = 0
	rows, err := v.list(context.Background(), "/pages/")
	if err != nil || len(rows) != 2 {
		t.Fatalf("pagination: rows=%v error=%v", rows, err)
	}
	rows, err = v.list(context.Background(), "/array/")
	if err != nil || len(rows) != 1 {
		t.Fatalf("array: rows=%v error=%v", rows, err)
	}
}
