//go:build integration

package instancemanagement

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type canceledConfigurationCipher struct {
	cancel  context.CancelFunc
	failure error
}

func (c canceledConfigurationCipher) Encrypt(string) (string, error) {
	c.cancel()
	return "", c.failure
}
func (c canceledConfigurationCipher) Decrypt(value string) (string, error) { return value, nil }

type rollbackContextTracer struct {
	cleanup           context.Context
	canceled, bounded bool
	failure           error
}

func (tr *rollbackContextTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if data.SQL == "rollback" {
		tr.cleanup = ctx
		tr.canceled = ctx.Err() != nil
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		tr.bounded = ok && remaining > 0 && remaining <= 5*time.Second
	}
	return ctx
}
func (tr *rollbackContextTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if ctx == tr.cleanup {
		tr.failure = data.Err
	}
}

func TestConfigurationFailureRollsBackAfterCallerCancellation(t *testing.T) {
	pair, err := postgresfixture.StartUncloned(t.Context(), func(ctx context.Context, url string) error {
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			return err
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, `CREATE TABLE instance_configurations(id uuid PRIMARY KEY,key text,value text,category text,is_encrypted boolean,deleted_at timestamptz); INSERT INTO instance_configurations VALUES('00000000-0000-4000-8000-000000000001','SECRET','original','email',true,NULL)`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, pair)
	config, err := pgxpool.ParseConfig(pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	tracer := &rollbackContextTracer{}
	config.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	failure := errors.New("encryption failed")
	repo := &Repository{db: pool, cipher: canceledConfigurationCipher{cancel: cancel, failure: failure}}
	_, err = repo.PatchConfigurations(ctx, map[string]json.RawMessage{"SECRET": json.RawMessage(`"replacement"`)})
	if !errors.Is(err, failure) {
		t.Fatalf("lost encryption failure: %v", err)
	}
	if tracer.cleanup == nil || tracer.canceled || !tracer.bounded || tracer.failure != nil {
		t.Fatalf("rollback called=%t canceled=%t bounded=%t failure=%v", tracer.cleanup != nil, tracer.canceled, tracer.bounded, tracer.failure)
	}
	if !errors.Is(tracer.cleanup.Err(), context.Canceled) {
		t.Fatalf("cleanup context was not released: %v", tracer.cleanup.Err())
	}
	var value string
	if err := pool.QueryRow(t.Context(), `SELECT value FROM instance_configurations WHERE key='SECRET'`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "original" {
		t.Fatalf("failed mutation persisted %q", value)
	}
}
