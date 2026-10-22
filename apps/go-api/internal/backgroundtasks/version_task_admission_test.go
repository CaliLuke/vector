package backgroundtasks

import (
	"context"
	"errors"
	"testing"
	"time"
)

type versionTaskFallback struct {
	name   string
	args   []any
	kwargs map[string]any
}

func (fallback *versionTaskFallback) Task(_ context.Context, name string, args []any, kwargs map[string]any) error {
	fallback.name, fallback.args, fallback.kwargs = name, args, kwargs
	return nil
}

func (*versionTaskFallback) SoftDelete(context.Context, string, string, string) error { return nil }

type issueVersionAdmitRecorder struct {
	eventID string
	input   IssueVersionTaskInput
	err     error
}

func (recorder *issueVersionAdmitRecorder) Admit(_ context.Context, eventID string, input IssueVersionTaskInput) (string, error) {
	recorder.eventID, recorder.input = eventID, input
	return eventID, recorder.err
}

type syncAdmitRecorder struct {
	eventID string
	batch   int
	offset  int
	delay   time.Duration
	err     error
}

func (recorder *syncAdmitRecorder) Admit(_ context.Context, eventID string, batch, offset int, delay time.Duration) (string, error) {
	recorder.eventID, recorder.batch, recorder.offset, recorder.delay = eventID, batch, offset, delay
	return eventID, recorder.err
}

type activityLogAdmitRecorder struct {
	input APIActivityLogInput
	err   error
}

func (recorder *activityLogAdmitRecorder) AdmitInput(_ context.Context, input APIActivityLogInput) (string, error) {
	recorder.input = input
	return input.ID, recorder.err
}

func TestVersionTaskPublisherRoutesDurableTasksAndPreservesFallback(t *testing.T) {
	fallback := &versionTaskFallback{}
	issueVersions := &issueVersionAdmitRecorder{}
	issueSync := &syncAdmitRecorder{}
	descriptionSync := &syncAdmitRecorder{}
	activityLog := &activityLogAdmitRecorder{}
	publisher, err := NewVersionTaskPublisher(fallback, issueVersions, issueSync, descriptionSync, activityLog)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := publisher.Task(ctx, issueTaskCeleryName, []any{`{"name":"new"}`, "issue-id", "user-id"}, nil); err != nil {
		t.Fatal(err)
	}
	if issueVersions.input.UpdatedIssue != `{"name":"new"}` || issueVersions.input.IssueID != "issue-id" || issueVersions.input.UserID != "user-id" || issueVersions.eventID == "" {
		t.Fatalf("issue version admission = %#v, event %q", issueVersions.input, issueVersions.eventID)
	}

	logData := map[string]any{"token_identifier": "token", "path": "/api/v1/issues/", "method": "GET", "response_code": 200, "headers": map[string]any{"accept": "application/json"}}
	if err := publisher.Task(ctx, processLogsCeleryName, nil, map[string]any{"log_data": logData, "mongo_log": "legacy"}); err != nil {
		t.Fatal(err)
	}
	if activityLog.input.ID == "" || activityLog.input.TokenIdentifier != "token" || activityLog.input.Path != "/api/v1/issues/" || activityLog.input.ResponseCode != 200 {
		t.Fatalf("process_logs admission = %#v", activityLog.input)
	}

	if err := publisher.Task(ctx, scheduleIssueDescriptionVersionCelery, nil, map[string]any{"batch_size": 120, "countdown": 45}); err != nil {
		t.Fatal(err)
	}
	if descriptionSync.batch != 120 || descriptionSync.offset != 0 || descriptionSync.delay != 45*time.Second || descriptionSync.eventID == "" {
		t.Fatalf("scheduled description sync = %#v", descriptionSync)
	}
	if err := publisher.Task(ctx, syncIssueDescriptionVersionCelery, nil, map[string]any{"batch_size": 11, "offset": 44, "countdown": 7}); err != nil {
		t.Fatal(err)
	}
	if descriptionSync.batch != 11 || descriptionSync.offset != 44 || descriptionSync.delay != 7*time.Second {
		t.Fatalf("description sync continuation = %#v", descriptionSync)
	}
	if err := publisher.Task(ctx, scheduleIssueVersionCelery, []any{150, 30}, nil); err != nil {
		t.Fatal(err)
	}
	if issueSync.batch != 150 || issueSync.offset != 0 || issueSync.delay != 30*time.Second {
		t.Fatalf("scheduled issue sync = %#v", issueSync)
	}
	if err := publisher.Task(ctx, syncIssueVersionCelery, nil, map[string]any{"batch_size": 5, "offset": 10, "countdown": 9}); err != nil {
		t.Fatal(err)
	}
	if issueSync.batch != 5 || issueSync.offset != 10 || issueSync.delay != 9*time.Second {
		t.Fatalf("issue sync continuation = %#v", issueSync)
	}

	if err := publisher.Task(ctx, "plane.other.task", []any{"arg"}, map[string]any{"key": "value"}); err != nil {
		t.Fatal(err)
	}
	if fallback.name != "plane.other.task" || len(fallback.args) != 1 || fallback.kwargs["key"] != "value" {
		t.Fatalf("fallback task = %q %#v %#v", fallback.name, fallback.args, fallback.kwargs)
	}
}

func TestVersionTaskPublisherPropagatesAdmissionAndPayloadErrors(t *testing.T) {
	fallback := &versionTaskFallback{}
	issueVersions := &issueVersionAdmitRecorder{err: errors.New("admission failed")}
	publisher, err := NewVersionTaskPublisher(fallback, issueVersions, &syncAdmitRecorder{}, &syncAdmitRecorder{}, &activityLogAdmitRecorder{})
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Task(context.Background(), issueTaskCeleryName, []any{`{}`, "issue", "user"}, nil); err == nil || err.Error() != "admission failed" {
		t.Fatalf("admission error = %v", err)
	}
	if err := publisher.Task(context.Background(), processLogsCeleryName, nil, map[string]any{"log_data": "bad"}); err == nil {
		t.Fatal("expected invalid process_logs payload to fail")
	}
	if _, _, _, err := versionSyncOptions(nil, map[string]any{"batch_size": "bad"}, false); err == nil {
		t.Fatal("expected invalid batch size to fail")
	}
}
