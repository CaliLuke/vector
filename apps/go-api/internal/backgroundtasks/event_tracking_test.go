package backgroundtasks

import (
	"context"
	"errors"
	"testing"

	"github.com/posthog/posthog-go"
)

func TestEventTrackingUsesConfiguredPostHogCapture(t *testing.T) {
	t.Setenv("SKIP_ENV_VAR", "0")
	t.Setenv("POSTHOG_API_KEY", "fixture-key")
	t.Setenv("POSTHOG_HOST", "https://posthog.example.test")
	client := &eventTrackingClientRecorder{}
	store := &eventTrackingStore{newClient: func(apiKey, host string) (eventTrackingClient, error) {
		if apiKey != "fixture-key" || host != "https://posthog.example.test" {
			t.Fatalf("PostHog config key=%q host=%q", apiKey, host)
		}
		return client, nil
	}}
	input := EventTrackingInput{UserID: "3b3c8ce6-c6aa-4373-9d0b-8df90aafeb1b", EventName: "workspace_created", Slug: "acme", EventProperties: map[string]any{"workspace_name": "Acme"}}
	if err := store.Track(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if client.enqueued.Event != input.EventName || client.enqueued.DistinctId != input.UserID || client.enqueued.Groups["workspace"] != input.Slug {
		t.Fatalf("captured PostHog event=%+v", client.enqueued)
	}
	if client.enqueued.Properties["workspace_name"] != "Acme" || client.flushCalls != 1 || client.closeCalls != 1 {
		t.Fatalf("capture delivery state=%+v flush=%d close=%d", client.enqueued.Properties, client.flushCalls, client.closeCalls)
	}
}

func TestEventTrackingUnavailableConfigurationIsNoOp(t *testing.T) {
	t.Setenv("SKIP_ENV_VAR", "false")
	t.Setenv("POSTHOG_API_KEY", "")
	t.Setenv("POSTHOG_HOST", "")
	store := &eventTrackingStore{}
	if err := store.Track(context.Background(), EventTrackingInput{UserID: "3b3c8ce6-c6aa-4373-9d0b-8df90aafeb1b", EventName: "workspace_created", Slug: "acme"}); err != nil {
		t.Fatalf("unconfigured tracking should follow source no-op behavior: %v", err)
	}
}

type eventTrackingClientRecorder struct {
	enqueued   posthog.Capture
	flushCalls int
	closeCalls int
}

func (recorder *eventTrackingClientRecorder) Enqueue(message posthog.Message) error {
	capture, ok := message.(posthog.Capture)
	if !ok {
		return errors.New("unexpected PostHog message type")
	}
	recorder.enqueued = capture
	return nil
}

func (recorder *eventTrackingClientRecorder) FlushWithContext(context.Context) error {
	recorder.flushCalls++
	return nil
}

func (recorder *eventTrackingClientRecorder) Close() error {
	recorder.closeCalls++
	return nil
}
