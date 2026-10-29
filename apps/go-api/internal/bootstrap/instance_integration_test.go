package bootstrap

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresBootstrapRegistrationAndConfiguration(t *testing.T) {
	databaseURL := postgresfixture.EmptyDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer adminPool.Close()

	schema := "bootstrap_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, execErr := adminPool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); execErr != nil {
		t.Fatal(execErr)
	}

	_, err = adminPool.Exec(ctx, `CREATE TABLE `+pgx.Identifier{schema}.Sanitize()+`.instances (
		created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, id uuid PRIMARY KEY,
		instance_name varchar(255) NOT NULL, whitelist_emails text, instance_id varchar(255) NOT NULL UNIQUE,
		current_version varchar(255) NOT NULL, last_checked_at timestamptz NOT NULL, namespace varchar(255),
		is_telemetry_enabled boolean NOT NULL, is_support_required boolean NOT NULL,
		is_setup_done boolean NOT NULL, is_signup_screen_visited boolean NOT NULL, is_verified boolean NOT NULL,
		created_by_id uuid, updated_by_id uuid, domain text NOT NULL, latest_version varchar(255),
		edition varchar(255) NOT NULL, deleted_at timestamptz, is_test boolean NOT NULL,
		is_current_version_deprecated boolean NOT NULL
	);
	CREATE TABLE `+pgx.Identifier{schema}.Sanitize()+`.instance_configurations (
		created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, id uuid PRIMARY KEY,
		key varchar(100) NOT NULL UNIQUE, value text, category text NOT NULL, is_encrypted boolean NOT NULL,
		created_by_id uuid, updated_by_id uuid, deleted_at timestamptz
	)`)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const callers = 12
	createdCount := 0
	var metricsAdmissions atomic.Int32
	var mu sync.Mutex
	var wait sync.WaitGroup
	registrationErrors := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			created, registerErr := RegisterInstance(ctx, pool, Registration{
				MachineSignature: "machine-signature", CurrentVersion: "v1.2.3", IsTest: true,
			}, func(context.Context, string) string { return "v9.8.7" }, func(context.Context) error {
				metricsAdmissions.Add(1)
				return nil
			})
			if registerErr != nil {
				registrationErrors <- registerErr
				return
			}
			if created {
				mu.Lock()
				createdCount++
				mu.Unlock()
			}
		}()
	}
	wait.Wait()
	close(registrationErrors)
	for err := range registrationErrors {
		t.Error(err)
	}
	if createdCount != 1 || metricsAdmissions.Load() != callers {
		t.Fatalf("concurrent registration results: created rows=%d (want one), metrics admissions=%d (want %d)", createdCount, metricsAdmissions.Load(), callers)
	}
	if _, registerInstanceErr := RegisterInstance(ctx, pool, Registration{CurrentVersion: "v1.2.3", IsTest: true}, func(context.Context, string) string { return "v9.8.7" }, nil); !errors.Is(registerInstanceErr, ErrMetricsAdmissionUnavailable) {
		t.Fatalf("missing native metrics queue should be an explicit gap, got %v", registerInstanceErr)
	}
	var instanceCount int
	var instanceID string
	var version, latest, edition, instanceName string
	var telemetry, support, testInstance bool
	err = pool.QueryRow(ctx, `SELECT count(*),min(id::text),min(current_version),min(latest_version),min(edition),min(instance_name),bool_and(is_telemetry_enabled),bool_and(is_support_required),bool_and(is_test) FROM instances`).Scan(&instanceCount, &instanceID, &version, &latest, &edition, &instanceName, &telemetry, &support, &testInstance)
	if err != nil {
		t.Fatal(err)
	}
	if instanceCount != 1 || instanceID == "" || version != "v1.2.3" || latest != "v9.8.7" || edition != "PLANE_COMMUNITY" || instanceName != "Vector Community Edition" || !telemetry || !support || !testInstance {
		t.Fatalf("registered instance defaults differ from Django: count=%d id=%q version=%q latest=%q edition=%q name=%q telemetry=%t support=%t test=%t", instanceCount, instanceID, version, latest, edition, instanceName, telemetry, support, testInstance)
	}
	created, err := RegisterInstance(ctx, pool, Registration{CurrentVersion: "v2.0.0"}, func(context.Context, string) string { return "v2.0.0" }, func(context.Context) error {
		metricsAdmissions.Add(1)
		return nil
	})
	if err != nil || created {
		t.Fatalf("existing registration should update without a machine signature: created=%t err=%v", created, err)
	}
	if got, want := metricsAdmissions.Load(), int32(callers+1); got != want {
		t.Fatalf("metrics admissions after repeat registration = %d, want %d", got, want)
	}

	environment := map[string]string{"ENABLE_SIGNUP": "0", "EMAIL_HOST_PASSWORD": "initial-password"}
	lookup := func(key string) (string, bool) { value, ok := environment[key]; return value, ok }
	configure := func() error { return ConfigureInstance(ctx, pool, "bootstrap-test-secret", lookup) }
	var configWait sync.WaitGroup
	configErrors := make(chan error, 4)
	for i := 0; i < cap(configErrors); i++ {
		configWait.Add(1)
		go func() { defer configWait.Done(); configErrors <- configure() }()
	}
	configWait.Wait()
	close(configErrors)
	for err := range configErrors {
		if err != nil {
			t.Error(err)
		}
	}
	if _, execErr := pool.Exec(ctx, `UPDATE instance_configurations SET value='user-selected' WHERE key='ENABLE_SIGNUP'`); execErr != nil {
		t.Fatal(execErr)
	}
	environment["ENABLE_SIGNUP"] = "1"
	environment["EMAIL_HOST_PASSWORD"] = "replacement-password"
	if configureErr := configure(); configureErr != nil {
		t.Fatal(configureErr)
	}
	var savedValue, encryptedPassword string
	var isEncrypted bool
	if scanErr := pool.QueryRow(ctx, `SELECT value FROM instance_configurations WHERE key='ENABLE_SIGNUP'`).Scan(&savedValue); scanErr != nil {
		t.Fatal(scanErr)
	}
	if scanErr := pool.QueryRow(ctx, `SELECT value,is_encrypted FROM instance_configurations WHERE key='EMAIL_HOST_PASSWORD'`).Scan(&encryptedPassword, &isEncrypted); scanErr != nil {
		t.Fatal(scanErr)
	}
	cipher, err := instancecipher.New("bootstrap-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := cipher.Decrypt(encryptedPassword)
	if err != nil {
		t.Fatal(err)
	}
	var configCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instance_configurations`).Scan(&configCount); err != nil {
		t.Fatal(err)
	}
	if savedValue != "user-selected" || !isEncrypted || plaintext != "initial-password" || configCount != 36 {
		t.Fatalf("configuration should preserve existing values: signup=%q encrypted=%t password=%q count=%d", savedValue, isEncrypted, plaintext, configCount)
	}
}
