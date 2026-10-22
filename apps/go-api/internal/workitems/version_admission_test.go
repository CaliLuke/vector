package workitems

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"
)

type versionTaskRecorder struct {
	names  []string
	kwargs []map[string]any
	err    error
}

func (recorder *versionTaskRecorder) Task(_ context.Context, name string, _ []any, kwargs map[string]any) error {
	recorder.names = append(recorder.names, name)
	recorder.kwargs = append(recorder.kwargs, kwargs)
	return recorder.err
}
func (*versionTaskRecorder) SoftDelete(context.Context, string, string, string) error { return nil }

type nativeVersionRecorder struct {
	updated, issue, user string
	eventID              string
	eventIDs             []string
	creating             bool
	calls                int
	err                  error
}

func (recorder *nativeVersionRecorder) AdmitIssueDescriptionVersion(_ context.Context, eventID, updated, issue, user string, creating bool) error {
	recorder.eventID, recorder.updated, recorder.issue, recorder.user, recorder.creating = eventID, updated, issue, user, creating
	recorder.eventIDs = append(recorder.eventIDs, eventID)
	recorder.calls++
	return recorder.err
}

func TestNativeIssueDescriptionVersionAdmissionReplacesLegacyPublish(t *testing.T) {
	legacy := &versionTaskRecorder{}
	native := &nativeVersionRecorder{}
	repository := &Repository{publisher: legacy, issueDescriptionVersions: native}
	if err := repository.admitIssueDescriptionVersion(context.Background(), map[string]any{"name": "A"}, "issue", "actor", true); err != nil {
		t.Fatal(err)
	}
	if native.calls != 1 || uuid.Validate(native.eventID) != nil || native.issue != "issue" || native.user != "actor" || !native.creating || native.updated != `{"name":"A"}` {
		t.Fatalf("native admission = %#v", native)
	}
	if len(legacy.names) != 0 {
		t.Fatalf("native admission also published legacy task: %#v", legacy.names)
	}
}

func TestRepeatedIdenticalEffectsReceiveDistinctAdmissionIDs(t *testing.T) {
	native := &nativeVersionRecorder{}
	repository := &Repository{publisher: &versionTaskRecorder{}, issueDescriptionVersions: native}
	for range 2 {
		if err := repository.admitIssueDescriptionVersion(context.Background(), `{"name":"A"}`, "issue", "actor", false); err != nil {
			t.Fatal(err)
		}
	}
	if native.calls != 2 || len(native.eventIDs) != 2 || native.eventIDs[0] == native.eventIDs[1] || !slices.Contains(native.eventIDs, native.eventID) {
		t.Fatalf("native admission IDs = %#v", native.eventIDs)
	}
}

func TestIssueDescriptionVersionLegacyFallbackPreservesPayload(t *testing.T) {
	legacy := &versionTaskRecorder{}
	repository := &Repository{publisher: legacy}
	if err := repository.admitIssueDescriptionVersion(context.Background(), `{"name":"A"}`, "issue", "actor", false); err != nil {
		t.Fatal(err)
	}
	wantName := "plane.bgtasks.issue_description_version_task.issue_description_version_task"
	wantArgs := map[string]any{"updated_issue": `{"name":"A"}`, "issue_id": "issue", "user_id": "actor"}
	if !reflect.DeepEqual(legacy.names, []string{wantName}) || !reflect.DeepEqual(legacy.kwargs, []map[string]any{wantArgs}) {
		t.Fatalf("legacy tasks = %#v %#v", legacy.names, legacy.kwargs)
	}
}

func TestNativeIssueDescriptionVersionAdmissionFailurePropagatesWithoutFallback(t *testing.T) {
	wantErr := errors.New("durable admission failed")
	legacy := &versionTaskRecorder{}
	native := &nativeVersionRecorder{err: wantErr}
	repository := &Repository{publisher: legacy, issueDescriptionVersions: native}
	if err := repository.admitIssueDescriptionVersion(context.Background(), "{}", "issue", "actor", false); !errors.Is(err, wantErr) {
		t.Fatalf("admission error = %v, want %v", err, wantErr)
	}
	if len(legacy.names) != 0 {
		t.Fatalf("failed native admission unexpectedly fell back to legacy publish: %#v", legacy.names)
	}
}
