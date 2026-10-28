// Package bootstrap contains the database setup operations used by the Go-only
// installation command.
package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const registrationLockID int64 = 0x504c414e4501

type Registration struct {
	MachineSignature string
	CurrentVersion   string
	IsTest           bool
}

type LatestVersion func(context.Context, string) string
type MetricsAdmission func(context.Context) error

var ErrMetricsAdmissionUnavailable = errors.New("instance metrics admission requires native Go durable task support")

// RegisterInstance mirrors the Django management command's instance defaults.
// The transaction lock makes concurrent startup processes converge on the
// same newest active instance row.
func RegisterInstance(ctx context.Context, pool *pgxpool.Pool, input Registration, latest LatestVersion, admitMetrics MetricsAdmission) (bool, error) {
	if pool == nil {
		return false, errors.New("database pool is required")
	}
	if input.CurrentVersion == "" {
		input.CurrentVersion = "v0.1.0"
	}
	if latest == nil {
		latest = latestReleaseVersion
	}
	latestVersion := latest(ctx, input.CurrentVersion)
	if latestVersion == "" {
		latestVersion = input.CurrentVersion
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin instance registration: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Commit below determines the transaction result.
	if _, execErr := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(@lock_id)`, pgx.StrictNamedArgs{"lock_id": registrationLockID}); execErr != nil {
		return false, fmt.Errorf("lock instance registration: %w", execErr)
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC, id DESC LIMIT 1`).Scan(&id)
	now := time.Now().UTC()
	created := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !created {
		return false, fmt.Errorf("find registered instance: %w", err)
	}
	if created {
		if strings.TrimSpace(input.MachineSignature) == "" {
			return false, errors.New("machine signature is required")
		}
		instanceID, registrationErr := newInstanceID()
		if registrationErr != nil {
			return false, fmt.Errorf("generate instance identifier: %w", registrationErr)
		}
		id = uuid.New()
		_, registrationErr = tx.Exec(ctx, `INSERT INTO instances (
			created_at,updated_at,id,instance_name,whitelist_emails,instance_id,
			current_version,last_checked_at,namespace,is_telemetry_enabled,
			is_support_required,is_setup_done,is_signup_screen_visited,is_verified,
			created_by_id,updated_by_id,domain,latest_version,edition,deleted_at,
			is_test,is_current_version_deprecated
		) VALUES (
			@now,@now,@id,@instance_name,NULL,@instance_id,@current_version,@now,
			NULL,true,true,false,false,false,NULL,NULL,'',@latest_version,
			'PLANE_COMMUNITY',NULL,@is_test,false
		)`, pgx.StrictNamedArgs{
			"now": now, "id": id, "instance_name": "Vector Community Edition", "instance_id": instanceID,
			"current_version": input.CurrentVersion, "latest_version": latestVersion, "is_test": input.IsTest,
		})
		if registrationErr != nil {
			return false, fmt.Errorf("create registered instance: %w", registrationErr)
		}
	} else {
		_, err = tx.Exec(ctx, `UPDATE instances SET updated_at=@now,last_checked_at=@now,
			current_version=@current_version,latest_version=@latest_version,
			is_test=@is_test,edition='PLANE_COMMUNITY' WHERE id=@id`, pgx.StrictNamedArgs{
			"now": now, "current_version": input.CurrentVersion, "latest_version": latestVersion,
			"is_test": input.IsTest, "id": id,
		})
		if err != nil {
			return false, fmt.Errorf("update registered instance: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit instance registration: %w", err)
	}
	if admitMetrics == nil {
		return created, ErrMetricsAdmissionUnavailable
	}
	if err := admitMetrics(ctx); err != nil {
		return created, fmt.Errorf("admit instance metrics task: %w", err)
	}
	return created, nil
}

func newInstanceID() (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func latestReleaseVersion(ctx context.Context, fallback string) string {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/CaliLuke/vector/releases/latest", nil)
	if err != nil {
		return fallback
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fallback
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fallback
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&release); err != nil || release.TagName == "" {
		return fallback
	}
	return release.TagName
}

func ConfigureInstance(ctx context.Context, pool *pgxpool.Pool, secret string, lookupEnv func(string) (string, bool)) error {
	if pool == nil {
		return errors.New("database pool is required")
	}
	if secret == "" {
		return errors.New("SECRET_KEY env variable is required.")
	}
	if lookupEnv == nil {
		return errors.New("environment lookup is required")
	}
	cipher, err := instancecipher.New(secret)
	if err != nil {
		return fmt.Errorf("initialize instance configuration encryption: %w", err)
	}
	configurations, err := InstanceConfigurations(lookupEnv, cipher)
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin instance configuration: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Commit below determines the transaction result.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(@lock_id)`, pgx.StrictNamedArgs{"lock_id": registrationLockID + 1}); err != nil {
		return fmt.Errorf("lock instance configuration: %w", err)
	}
	now := time.Now().UTC()
	for _, item := range configurations {
		id := uuid.New()
		_, err := tx.Exec(ctx, `INSERT INTO instance_configurations (
			created_at,updated_at,id,key,value,category,is_encrypted,
			created_by_id,updated_by_id,deleted_at
		) VALUES (@now,@now,@id,@key,@value,@category,@is_encrypted,NULL,NULL,NULL)
		ON CONFLICT (key) DO NOTHING`, pgx.StrictNamedArgs{
			"now": now, "id": id, "key": item.Key, "value": item.Value,
			"category": item.Category, "is_encrypted": item.IsEncrypted,
		})
		if err != nil {
			return fmt.Errorf("create instance configuration %q: %w", item.Key, err)
		}
		var active bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM instance_configurations WHERE key=@key AND deleted_at IS NULL)`, pgx.StrictNamedArgs{"key": item.Key}).Scan(&active); err != nil {
			return fmt.Errorf("verify instance configuration %q: %w", item.Key, err)
		}
		if !active {
			return fmt.Errorf("instance configuration %q conflicts with a deleted row", item.Key)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit instance configuration: %w", err)
	}
	return nil
}

// ValidateDatabaseURL is shared by bootstrap commands that need PostgreSQL.
func ValidateDatabaseURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return errors.New("DATABASE_URL must be a PostgreSQL URL")
	}
	return nil
}
