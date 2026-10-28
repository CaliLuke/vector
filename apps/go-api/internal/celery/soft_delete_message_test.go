package celery

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestSoftDeleteMessageUsesKombuUUIDAndDurableCeleryEnvelope(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000009020"
	message, err := softDeleteMessage("db", "notification", id)
	if err != nil {
		t.Fatal(err)
	}
	if message.ContentType != "application/json" || message.ContentEncoding != "utf-8" || message.DeliveryMode != 2 {
		t.Fatalf("task must use persistent UTF-8 JSON: %#v", message)
	}
	if message.Headers["task"] != "plane.bgtasks.deletion_task.soft_delete_related_objects" || message.Headers["lang"] != "py" {
		t.Fatalf("wrong Celery task headers: %#v", message.Headers)
	}
	if _, parseErr := uuid.Parse(message.CorrelationId); parseErr != nil {
		t.Fatalf("invalid task ID: %v", parseErr)
	}
	if message.Headers["id"] != message.CorrelationId || message.Headers["root_id"] != message.CorrelationId {
		t.Fatalf("task identity differs between correlation and Celery headers: %#v", message.Headers)
	}
	// Captured from kombu.utils.json.dumps in the pinned Django test image.
	// A plain UUID string loses Kombu's Python UUID type on worker delivery.
	wantJSON := `[["db", "notification", {"__type__": "uuid", "__value__": {"hex": "00000000000040008000000000009020"}}], {"using": null}, {"callbacks": null, "errbacks": null, "chain": null, "chord": null}]`
	var got, want any
	if unmarshalErr := json.Unmarshal(message.Body, &got); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if unmarshalErr := json.Unmarshal([]byte(wantJSON), &want); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Celery envelope differs from Kombu: got %s; want %s", message.Body, wantJSON)
	}
	next, err := softDeleteMessage("db", "notification", id)
	if err != nil {
		t.Fatal(err)
	}
	if next.CorrelationId == message.CorrelationId || !bytes.Equal(next.Body, message.Body) {
		t.Fatal("separate publications must have distinct task IDs and equivalent payloads")
	}
}

func TestSoftDeleteMessageRejectsInvalidUUID(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "00000000-0000-4000-8000-00000000902x"} {
		if _, err := softDeleteMessage("db", "notification", id); err == nil {
			t.Errorf("softDeleteMessage accepted invalid UUID %q", id)
		}
	}
}
