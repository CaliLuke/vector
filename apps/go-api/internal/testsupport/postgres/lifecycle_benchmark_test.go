//go:build integration

package postgres

import (
	"context"
	"testing"
)

// BenchmarkDatabaseLifecycle measures per-test isolation cost without container startup.
func BenchmarkDatabaseLifecycle(b *testing.B) {
	pair, err := StartUncloned(b.Context(), func(context.Context, string) error { return nil })
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := pair.Close(); err != nil {
			b.Error(err)
		}
	})
	b.ResetTimer()
	for b.Loop() {
		database, err := pair.CloneDatabase(b.Context())
		if err != nil {
			b.Fatal(err)
		}
		if err := database.Close(b.Context()); err != nil {
			b.Fatal(err)
		}
	}
}
