package backgroundtasks

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/selvedge"
)

type fileAssetTaskFallback struct {
	taskCalls int
	err       error
}

func (fallback *fileAssetTaskFallback) Task(context.Context, string, []any, map[string]any) error {
	fallback.taskCalls++
	return fallback.err
}
func (*fileAssetTaskFallback) SoftDelete(context.Context, string, string, string) error { return nil }

func TestFileAssetMetadataTaskIDAcceptsBothSourcePayloadForms(t *testing.T) {
	for _, test := range []struct {
		name   string
		args   []any
		kwargs map[string]any
	}{
		{name: "positional", args: []any{"asset-id"}},
		{name: "keyword", kwargs: map[string]any{"asset_id": "asset-id"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := fileAssetMetadataID(test.args, test.kwargs)
			if err != nil || got != "asset-id" {
				t.Fatalf("fileAssetMetadataID() = %q, %v", got, err)
			}
		})
	}
}

func TestFileAssetMetadataTaskIDRejectsAmbiguousPayload(t *testing.T) {
	for _, test := range []struct {
		args   []any
		kwargs map[string]any
	}{
		{args: []any{"id", "extra"}},
		{kwargs: map[string]any{"asset_id": "id", "other": true}},
		{kwargs: map[string]any{"wrong": "id"}},
		{kwargs: map[string]any{"asset_id": 12}},
	} {
		if _, err := fileAssetMetadataID(test.args, test.kwargs); err == nil {
			t.Fatalf("fileAssetMetadataID(%#v, %#v) unexpectedly succeeded", test.args, test.kwargs)
		}
	}
}

func TestFileAssetMetadataTaskPublisherPreservesFallbackAndErrors(t *testing.T) {
	fallback := &fileAssetTaskFallback{}
	app := selvedge.NewApp()
	workflow := selvedge.DefineWorkflow(assets.StorageMetadataWorkflowName, func(*selvedge.WorkflowContext, assets.StorageMetadataWorkflowInput) (struct{}, error) {
		return struct{}{}, nil
	})
	dispatcher, err := NewFileAssetMetadataDispatcher(app, workflow)
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := NewFileAssetMetadataTaskPublisher(fallback, dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Task(context.Background(), "unrelated.task", nil, nil); err != nil {
		t.Fatalf("route unrelated task: %v", err)
	}
	if fallback.taskCalls != 1 {
		t.Fatalf("fallback task calls = %d, want 1", fallback.taskCalls)
	}
	fallback.err = errors.New("publisher failure")
	if err := publisher.Task(context.Background(), "unrelated.task", nil, nil); !errors.Is(err, fallback.err) {
		t.Fatalf("fallback error = %v, want %v", err, fallback.err)
	}
}
