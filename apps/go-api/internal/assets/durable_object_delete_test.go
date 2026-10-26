package assets

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type objectDeleteGuardFunc func(context.Context, uuid.UUID, string) (bool, error)

func (guard objectDeleteGuardFunc) ObjectKeyInUse(ctx context.Context, assetID uuid.UUID, key string) (bool, error) {
	return guard(ctx, assetID, key)
}

type objectDeleteRecorder struct{ keys []string }

func (recorder *objectDeleteRecorder) Delete(_ context.Context, key string) error {
	recorder.keys = append(recorder.keys, key)
	return nil
}

func TestDeleteStorageObjectUsesCapturedKeyAndSkipsReplacementReference(t *testing.T) {
	assetID, eventID := uuid.New(), uuid.New()
	const capturedKey = "workspace/original.png"
	for _, test := range []struct {
		name       string
		inUse      bool
		wantDelete bool
	}{
		{name: "removed object", wantDelete: true},
		{name: "restored or replacement reference", inUse: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			storage := &objectDeleteRecorder{}
			guard := objectDeleteGuardFunc(func(_ context.Context, gotID uuid.UUID, gotKey string) (bool, error) {
				if gotID != assetID || gotKey != capturedKey {
					t.Fatalf("guard identity = %s/%q, want %s/%q", gotID, gotKey, assetID, capturedKey)
				}
				return test.inUse, nil
			})
			input := StorageObjectDeleteInput{AssetID: assetID.String(), ObjectKey: capturedKey, EventID: eventID.String()}
			if err := deleteStorageObjectIfUnused(context.Background(), input, guard, storage); err != nil {
				t.Fatal(err)
			}
			want := 0
			if test.wantDelete {
				want = 1
			}
			if len(storage.keys) != want || want == 1 && storage.keys[0] != capturedKey {
				t.Fatalf("deleted keys = %v, want %d occurrence(s) of %q", storage.keys, want, capturedKey)
			}
		})
	}
}

func TestDeleteStorageObjectRetriesIdempotentlyAndPreservesGuardFailures(t *testing.T) {
	assetID, eventID := uuid.New(), uuid.New()
	storage := &objectDeleteRecorder{}
	input := StorageObjectDeleteInput{AssetID: assetID.String(), ObjectKey: "workspace/removed.png", EventID: eventID.String()}
	guard := objectDeleteGuardFunc(func(context.Context, uuid.UUID, string) (bool, error) { return false, nil })
	for range 2 {
		if err := deleteStorageObjectIfUnused(context.Background(), input, guard, storage); err != nil {
			t.Fatal(err)
		}
	}
	if len(storage.keys) != 2 {
		t.Fatalf("idempotent retries issued %d deletes, want 2 retries of the exact immutable key", len(storage.keys))
	}
	guardErr := errors.New("visibility store unavailable")
	storage.keys = nil
	err := deleteStorageObjectIfUnused(context.Background(), input, objectDeleteGuardFunc(func(context.Context, uuid.UUID, string) (bool, error) { return false, guardErr }), storage)
	if !errors.Is(err, guardErr) || len(storage.keys) != 0 {
		t.Fatalf("guard failure = %v, deletes=%v; want propagated error without storage call", err, storage.keys)
	}
}
