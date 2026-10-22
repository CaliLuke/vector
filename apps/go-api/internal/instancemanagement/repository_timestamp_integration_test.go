//go:build integration

package instancemanagement

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type timestampTestCipher struct{}

func (timestampTestCipher) Encrypt(value string) (string, error) { return value, nil }
func (timestampTestCipher) Decrypt(value string) (string, error) { return value, nil }

func TestConfigurationBulkWritesPreserveUpdatedAt(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, `
CREATE TABLE instance_configurations (
  id uuid PRIMARY KEY,
  key text NOT NULL,
  value text,
  category text NOT NULL,
  is_encrypted boolean NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  deleted_at timestamptz,
  created_by_id uuid,
  updated_by_id uuid
);
INSERT INTO instance_configurations (id,key,value,category,is_encrypted,created_at,updated_at) VALUES
('00000000-0000-4000-8000-000000000001','EMAIL_HOST','smtp.example.test','email',false,'2001-01-01T00:00:00Z','2001-01-01T00:00:00Z'),
('00000000-0000-4000-8000-000000000002','EMAIL_HOST_USER','user','email',false,'2001-01-01T00:00:00Z','2001-01-01T00:00:00Z'),
('00000000-0000-4000-8000-000000000003','EMAIL_HOST_PASSWORD','secret','email',false,'2001-01-01T00:00:00Z','2001-01-01T00:00:00Z'),
('00000000-0000-4000-8000-000000000004','ENABLE_SMTP','1','email',false,'2001-01-01T00:00:00Z','2001-01-01T00:00:00Z'),
('00000000-0000-4000-8000-000000000005','EMAIL_PORT','587','email',false,'2001-01-01T00:00:00Z','2001-01-01T00:00:00Z'),
('00000000-0000-4000-8000-000000000006','EMAIL_FROM','Plane <noreply@example.test>','email',false,'2001-01-01T00:00:00Z','2001-01-01T00:00:00Z');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()

	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repository, err := NewRepository(pool, timestampTestCipher{})
	if err != nil {
		t.Fatal(err)
	}
	wantUpdatedAt := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)

	if _, err := repository.PatchConfigurations(ctx, map[string]json.RawMessage{"EMAIL_HOST": json.RawMessage(`"mail.example.test"`)}); err != nil {
		t.Fatalf("PatchConfigurations: %v", err)
	}
	var value string
	var updatedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT value,updated_at FROM instance_configurations WHERE key='EMAIL_HOST'`).Scan(&value, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if value != "mail.example.test" || !updatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("patch persisted value=%q updated_at=%s; want value mail.example.test and unchanged updated_at=%s", value, updatedAt, wantUpdatedAt)
	}

	if err := repository.DisableEmail(ctx); err != nil {
		t.Fatalf("DisableEmail: %v", err)
	}
	rows, err := pool.Query(ctx, `SELECT key,value,updated_at FROM instance_configurations ORDER BY key`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, actualValue string
		var actualUpdatedAt time.Time
		if err := rows.Scan(&key, &actualValue, &actualUpdatedAt); err != nil {
			t.Fatal(err)
		}
		wantValue := ""
		if key == "ENABLE_SMTP" {
			wantValue = "0"
		}
		if actualValue != wantValue || !actualUpdatedAt.Equal(wantUpdatedAt) {
			t.Errorf("disable email persisted %s value=%q updated_at=%s; want value=%q and unchanged updated_at=%s", key, actualValue, actualUpdatedAt, wantValue, wantUpdatedAt)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
