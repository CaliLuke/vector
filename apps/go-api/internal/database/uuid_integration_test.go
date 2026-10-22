//go:build integration

package database

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
	"uuid"

	generated "github.com/CaliLuke/go-plane/internal/database/db"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGeneratedUUIDQueriesRoundTrip(t *testing.T) {
	if os.Getenv("PLANE_RUN_UUID_ROUNDTRIP") != "1" {
		t.Skip("set PLANE_RUN_UUID_ROUNDTRIP=1 to run the UUID round-trip proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pair, err := postgresfixture.Start(ctx, applySupportMigrations)
	if err != nil {
		t.Fatalf("start PostgreSQL fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := pair.Close(); err != nil {
			t.Errorf("close PostgreSQL fixture: %v", err)
		}
	})

	config, err := pgxpool.ParseConfig(pair.SourceURL)
	if err != nil {
		t.Fatalf("parse PostgreSQL pool configuration: %v", err)
	}
	if err := ConfigurePool(config); err != nil {
		t.Fatalf("configure PostgreSQL pool: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	queries := generated.New(pool)
	nonnullID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290281")
	nullID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290282")
	nullableID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290283")
	relatedIDs := []uuid.UUID{
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290284"),
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290285"),
	}

	inserted, err := queries.PutUUIDProbe(ctx, generated.PutUUIDProbeParams{
		ID:         nonnullID,
		NullableID: pgtype.UUID{Bytes: [16]byte(nullableID), Valid: true},
		RelatedIDs: relatedIDs,
	})
	if err != nil {
		t.Fatalf("insert non-NULL UUID probe: %v", err)
	}
	assertUUIDProbe(t, inserted, nonnullID, nullableID, relatedIDs)

	if _, err := queries.PutUUIDProbe(ctx, generated.PutUUIDProbeParams{
		ID:         nullID,
		NullableID: pgtype.UUID{},
		RelatedIDs: []uuid.UUID{},
	}); err != nil {
		t.Fatalf("insert NULL UUID probe: %v", err)
	}
	nullableRow, err := queries.GetUUIDProbe(ctx, nullID)
	if err != nil {
		t.Fatalf("get NULL UUID probe: %v", err)
	}
	if nullableRow.ID != nullID || nullableRow.NullableID.Valid {
		t.Fatalf("NULL UUID row = %+v, want ID %s and SQL NULL", nullableRow, nullID)
	}
	if nullableRow.RelatedIDs == nil || len(nullableRow.RelatedIDs) != 0 {
		t.Fatalf("empty UUID array = %#v, want non-nil empty array", nullableRow.RelatedIDs)
	}

	selected, err := queries.GetUUIDProbe(ctx, nonnullID)
	if err != nil {
		t.Fatalf("get non-NULL UUID probe: %v", err)
	}
	assertUUIDProbe(t, selected, nonnullID, nullableID, relatedIDs)
}

func applySupportMigrations(ctx context.Context, databaseURL string) error {
	command := exec.CommandContext(
		ctx,
		"go", "run", "./cmd/support-migrate", "--database-url", databaseURL, "up",
	)
	command.Dir = goAPIRoot()
	command.Env = append(os.Environ(), "GOTOOLCHAIN=go1.27.3")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func assertUUIDProbe(
	t *testing.T,
	row generated.PlaneGoUuidProbe,
	wantID uuid.UUID,
	wantNullable uuid.UUID,
	wantRelated []uuid.UUID,
) {
	t.Helper()
	if row.ID != wantID {
		t.Fatalf("UUID ID = %s, want %s", row.ID, wantID)
	}
	if !row.NullableID.Valid || uuid.UUID(row.NullableID.Bytes) != wantNullable {
		t.Fatalf("nullable UUID = %+v, want %s", row.NullableID, wantNullable)
	}
	if !reflect.DeepEqual(row.RelatedIDs, wantRelated) {
		t.Fatalf("UUID array = %v, want %v", row.RelatedIDs, wantRelated)
	}
}

func goAPIRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}
