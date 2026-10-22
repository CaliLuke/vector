package workitems

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

type nativeRecentVisitRecorder struct {
	args  []string
	err   error
	calls int
}

func (recorder *nativeRecentVisitRecorder) AdmitRecentVisit(_ context.Context, eventID, slug, entity, entityID, userID, projectID string) error {
	recorder.args = []string{eventID, slug, entity, entityID, userID, projectID}
	recorder.calls++
	return recorder.err
}

func TestRecentVisitUsesNativeAdmissionWithFreshOperationID(t *testing.T) {
	legacy := &versionTaskRecorder{}
	native := &nativeRecentVisitRecorder{}
	repository := &Repository{publisher: legacy, recentVisits: native}
	request := Request{Slug: "workspace", ProjectID: "project", ActorID: "user"}
	if err := repository.recentVisit(context.Background(), request, "issue", "issue-id"); err != nil {
		t.Fatal(err)
	}
	if native.calls != 1 || uuid.Validate(native.args[0]) != nil || !reflect.DeepEqual(native.args[1:], []string{"workspace", "issue", "issue-id", "user", "project"}) {
		t.Fatalf("native recent visit arguments = %#v", native.args)
	}
	if len(legacy.names) != 0 {
		t.Fatalf("native recent visit also published legacy task: %#v", legacy.names)
	}
}

func TestRecentVisitLegacyFallbackPreservesTaskPayload(t *testing.T) {
	legacy := &versionTaskRecorder{}
	repository := &Repository{publisher: legacy}
	request := Request{Slug: "workspace", ProjectID: "project", ActorID: "user"}
	if err := repository.recentVisit(context.Background(), request, "issue", "issue-id"); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"slug": "workspace", "entity_name": "issue", "entity_identifier": "issue-id", "user_id": "user", "project_id": "project"}
	if !reflect.DeepEqual(legacy.kwargs, []map[string]any{want}) || !reflect.DeepEqual(legacy.names, []string{"plane.bgtasks.recent_visited_task.recent_visited_task"}) {
		t.Fatalf("legacy recent visit task = %#v %#v", legacy.names, legacy.kwargs)
	}
}

func TestRecentVisitAdmissionErrorPropagatesWithoutFallback(t *testing.T) {
	wantErr := errors.New("admission unavailable")
	legacy := &versionTaskRecorder{}
	native := &nativeRecentVisitRecorder{err: wantErr}
	repository := &Repository{publisher: legacy, recentVisits: native}
	err := repository.recentVisit(context.Background(), Request{Slug: "workspace", ActorID: "user"}, "issue", "issue-id")
	if !errors.Is(err, wantErr) {
		t.Fatalf("recent visit admission error = %v, want %v", err, wantErr)
	}
	if len(legacy.names) != 0 {
		t.Fatalf("failed native admission unexpectedly published legacy task: %#v", legacy.names)
	}
}
