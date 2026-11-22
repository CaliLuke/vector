package backgroundtasks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type effectActivityAdmission interface {
	AdmitIssueActivityTask(context.Context, string, map[string]any) error
}
type effectVersionAdmission interface {
	AdmitIssueDescriptionVersion(context.Context, string, string, string, string, bool) error
}
type effectDeletionAdmission interface {
	Admit(context.Context, DeletionInput) (string, error)
}
type effectWebhookAdmission interface {
	AdmitWebhookTask(context.Context, string, string, []any, map[string]any) error
}
type effectFileAssetDeleteAdmission interface {
	Admit(context.Context, string, string, string) error
}
type effectFileAssetMetadataAdmission interface {
	Admit(context.Context, string, string) (string, error)
}
type effectFileAssetUpload interface {
	CompleteAttachmentUpload(context.Context, string) error
}

type effectPageAdmission interface {
	AdmitPageTransaction(context.Context, string, any, any, string) error
	AdmitPageVersion(context.Context, string, string, any, string) error
}

type effectLinkPreviewAdmission interface {
	AdmitWithID(context.Context, string, LinkPreviewInput) (string, error)
}

// PostgresWorkItemEffectStore hands committed effects to their existing durable
// workflows. Admission means queued, not completed; workflow status owns execution.
type PostgresWorkItemEffectStore struct {
	pool          *pgxpool.Pool
	activities    effectActivityAdmission
	versions      effectVersionAdmission
	deletions     effectDeletionAdmission
	webhooks      effectWebhookAdmission
	assetDeletes  effectFileAssetDeleteAdmission
	assetMetadata effectFileAssetMetadataAdmission
	linkPreviews  effectLinkPreviewAdmission
	assetUploads  effectFileAssetUpload
	emails        TemplatedEmailTaskAdmitter
	pages         effectPageAdmission
	events        EventTrackingTaskAdmitter
	admissions    metric.Int64Counter
}

func NewPostgresWorkItemEffectStore(pool *pgxpool.Pool, activities effectActivityAdmission, versions effectVersionAdmission, deletions effectDeletionAdmission, webhooks effectWebhookAdmission, assetDeletes effectFileAssetDeleteAdmission, assetMetadata effectFileAssetMetadataAdmission, linkPreviews effectLinkPreviewAdmission, assetUploads effectFileAssetUpload, emails TemplatedEmailTaskAdmitter, events EventTrackingTaskAdmitter, pages ...effectPageAdmission) (*PostgresWorkItemEffectStore, error) {
	if pool == nil || activities == nil || versions == nil || deletions == nil || webhooks == nil || assetDeletes == nil || assetMetadata == nil || linkPreviews == nil || assetUploads == nil || emails == nil || events == nil {
		return nil, errors.New("work item effect database and native admissions are required")
	}
	if len(pages) > 1 {
		return nil, errors.New("at most one page admission may be configured")
	}
	var pageAdmission effectPageAdmission
	if len(pages) == 1 {
		pageAdmission = pages[0]
	}
	return &PostgresWorkItemEffectStore{
		pool: pool, activities: activities, versions: versions, deletions: deletions,
		webhooks: webhooks, assetDeletes: assetDeletes, assetMetadata: assetMetadata,
		linkPreviews: linkPreviews, assetUploads: assetUploads, emails: emails, pages: pageAdmission, events: events,
	}, nil
}

func (s *PostgresWorkItemEffectStore) Drain(ctx context.Context) (int, error) {
	for count := 0; count < 50; count++ {
		found, err := s.admitNext(ctx)
		if err != nil || !found {
			return count, err
		}
	}
	return 50, nil
}

func (s *PostgresWorkItemEffectStore) admitNext(ctx context.Context) (_ bool, resultErr error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var id, kind string
	var payload []byte
	err = tx.QueryRow(ctx, `SELECT id::text,kind,payload FROM work_item_effects WHERE admitted_at IS NULL AND quarantined_at IS NULL AND next_attempt_at<=clock_timestamp() ORDER BY next_attempt_at,created_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id, &kind, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Bound each admission so an unavailable destination cannot stall polling.
	admissionContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	admissionErr := s.admit(admissionContext, id, kind, payload)
	cancel()
	var outcome string
	switch {
	case admissionErr == nil:
		outcome = "admitted"
		_, err = tx.Exec(ctx, `UPDATE work_item_effects SET admitted_at=clock_timestamp(),attempts=attempts+1,payload='{}',last_error='' WHERE id=$1`, id)
	case errors.Is(admissionErr, errInvalidWorkItemEffectPayload):
		outcome = "quarantined"
		// The infinite retry time also protects the row from older pollers during a rollout.
		_, err = tx.Exec(ctx, `UPDATE work_item_effects SET quarantined_at=clock_timestamp(),attempts=attempts+1,next_attempt_at='infinity',last_error=$2 WHERE id=$1`, id, admissionErr.Error())
	default:
		outcome = "retry"
		_, err = tx.Exec(ctx, `UPDATE work_item_effects SET attempts=attempts+1,next_attempt_at=clock_timestamp()+interval '1 minute',last_error=$2 WHERE id=$1`, id, admissionErr.Error())
	}
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	kind = workItemEffectKind(kind)
	if s.admissions != nil {
		s.admissions.Add(ctx, 1, metric.WithAttributes(attribute.String("effect.kind", kind), attribute.String("effect.outcome", outcome)))
	}
	if outcome == "quarantined" {
		slog.WarnContext(ctx, "Quarantined permanently invalid work item effect", "effect.kind", kind, "effect.outcome", outcome)
	}
	return true, nil
}

// ReplayQuarantinedWorkItemEffect requeues a single inspected effect, preserving
// its admission identity and attempt count. The operator owns payload repair.
func ReplayQuarantinedWorkItemEffect(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `UPDATE work_item_effects SET quarantined_at=NULL,next_attempt_at=clock_timestamp() WHERE id=$1 AND admitted_at IS NULL AND quarantined_at IS NOT NULL`, id)
	return tag.RowsAffected() == 1, err
}

var errInvalidWorkItemEffectPayload = errors.New("invalid work item effect payload")

func decodeWorkItemEffect(raw []byte, target any) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return fmt.Errorf("%w: expected a JSON object", errInvalidWorkItemEffectPayload)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%w: %w", errInvalidWorkItemEffectPayload, err)
	}
	return nil
}

func workItemEffectKind(kind string) string {
	switch kind {
	case "event_tracking", "email_task", "page_transaction", "page_version", "asset_storage_upload",
		"link_preview", "webhook", "activity", "description_version", "soft_delete", "asset_storage_delete",
		"asset_storage_metadata":
		return kind
	default:
		return "unknown"
	}
}

func (s *PostgresWorkItemEffectStore) admit(ctx context.Context, id, kind string, raw []byte) error {
	switch kind {
	case "event_tracking":
		var payload EventTrackingInput
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		return s.events.AdmitEventTracking(ctx, id, payload)
	case "email_task":
		var payload struct {
			Name string `json:"name"`
			Args []any  `json:"args"`
		}
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		return s.emails.AdmitTemplatedEmailTask(ctx, id, payload.Name, payload.Args)
	case "page_transaction", "page_version":
		var payload struct {
			PageID   string `json:"page_id"`
			UserID   string `json:"user_id"`
			NewHTML  any    `json:"new_description_html"`
			OldHTML  any    `json:"old_description_html"`
			Existing any    `json:"existing_instance"`
		}
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		if s.pages == nil {
			return errors.New("page effect admission is not configured")
		}
		if kind == "page_transaction" {
			return s.pages.AdmitPageTransaction(ctx, id, payload.NewHTML, payload.OldHTML, payload.PageID)
		}
		return s.pages.AdmitPageVersion(ctx, id, payload.PageID, payload.Existing, payload.UserID)
	case "asset_storage_upload":
		var payload struct {
			AssetID string `json:"asset_id"`
		}
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		return s.assetUploads.CompleteAttachmentUpload(ctx, payload.AssetID)

	case "link_preview":
		var payload LinkPreviewInput
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		_, err := s.linkPreviews.AdmitWithID(ctx, id, payload)
		return err
	case "webhook":
		return s.admitWebhookEffect(ctx, id, raw)
	case "activity":
		var payload map[string]any
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		return s.activities.AdmitIssueActivityTask(ctx, id, payload)
	case "description_version":
		var payload struct {
			Snapshot   string `json:"snapshot"`
			IssueID    string `json:"issue_id"`
			UserID     string `json:"user_id"`
			IsCreating bool   `json:"is_creating"`
		}
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		return s.versions.AdmitIssueDescriptionVersion(ctx, id, payload.Snapshot, payload.IssueID, payload.UserID, payload.IsCreating)
	case "soft_delete":
		var payload DeletionInput
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		payload.EventID = id
		_, err := s.deletions.Admit(ctx, payload)
		return err
	case "asset_storage_delete":
		var payload struct {
			AssetID   string `json:"asset_id"`
			ObjectKey string `json:"object_key"`
		}
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		return s.assetDeletes.Admit(ctx, payload.AssetID, payload.ObjectKey, id)
	case "asset_storage_metadata":
		var payload struct {
			AssetID string `json:"asset_id"`
		}
		if err := decodeWorkItemEffect(raw, &payload); err != nil {
			return err
		}
		_, err := s.assetMetadata.Admit(ctx, payload.AssetID, id)
		return err
	default:
		// A newer producer can add kinds during a rolling release.
		return fmt.Errorf("unsupported work item effect %q", kind)
	}
}

func (s *PostgresWorkItemEffectStore) admitWebhookEffect(ctx context.Context, id string, raw []byte) error {
	var payload struct {
		Name   string         `json:"name"`
		Args   []any          `json:"args"`
		Kwargs map[string]any `json:"kwargs"`
	}
	if err := decodeWorkItemEffect(raw, &payload); err != nil {
		return err
	}
	return s.webhooks.AdmitWebhookTask(ctx, id, payload.Name, payload.Args, payload.Kwargs)
}
