package notionsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestMembershipSyncRemovesOnlyPreviouslyImportedLinks(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		t.Run(fmt.Sprint(interrupted), func(t *testing.T) {
			runner, _ := fixtureRunner(t)
			current := []string{"old-import", "manual"}
			if interrupted {
				current = append(current, "partial")
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPost:
					var body struct {
						Issues []string `json:"issues"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					current = append(current, body.Issues...)
				case http.MethodDelete:
					id := strings.Trim(r.URL.Path, "/")
					current = slices.DeleteFunc(current, func(value string) bool { return value == id })
				default:
					t.Error("unexpected HTTP method")
				}
				w.WriteHeader(204)
			}))
			defer server.Close()
			runner.Vector.api.base = server.URL
			runner.Journal.Entries["member:source"] = &Entry{Base: map[string]any{"issues": []string{"old-import"}}}
			if interrupted {
				runner.Journal.Entries["member:source"].Pending = map[string]any{"issues": []string{"partial"}}
			}
			if err := runner.syncMembers(context.Background(), "source", "/", current, []string{"new-import"}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(current, []string{"manual", "new-import"}) {
				t.Fatalf("membership results: %v", current)
			}

		})
	}
}

func TestMembershipSyncPreservesLocallyRemovedLink(t *testing.T) {
	runner, _ := fixtureRunner(t)
	runner.Journal.Entries["member:source"] = &Entry{Base: map[string]any{"issues": []string{"local-removal"}}}
	err := runner.syncMembers(context.Background(), "source", "/", nil, []string{"local-removal"})
	if err != nil || len(runner.report.Conflicts) != 1 {
		t.Fatalf("expected membership conflict: %v", err)
	}
}

func TestMissingUserMappingDoesNotClearExistingAssignees(t *testing.T) {
	runner, _ := fixtureRunner(t)
	fields, err := runner.taskFields(Page{ID: taskID, Title: "Task", People: []string{"unknown"}}, "state")
	if err != nil {
		t.Fatal(err)
	}
	if _, present := fields["assignees"]; present || len(runner.report.Warnings) != 1 {
		t.Fatal("unmapped identity silently assigned or cleared")
	}
}
