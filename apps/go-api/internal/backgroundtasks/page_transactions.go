package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"

	htmlcompat "github.com/CaliLuke/go-plane/internal/htmlcompat"
	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPermanentPageTransaction = errors.New("permanent page transaction task failure")

const (
	pageTransactionWorkflowName    = "plane.pages.track_transactions"
	pageTransactionActivityName    = "plane.pages.write_transactions"
	pageTransactionQueue           = "plane-background"
	pageTransactionActivityTimeout = 30 * time.Second
)

type PageTransactionInput struct {
	PageID             string `json:"page_id"`
	NewDescriptionHTML any    `json:"new_description_html"`
	OldDescriptionHTML any    `json:"old_description_html"`
}

type pageTransactionStore interface {
	TrackPageTransactions(context.Context, PageTransactionInput) error
}

type PageTransactionDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[PageTransactionInput, struct{}]
}

func RegisterPageTransactionWorkflow(app *selvedge.App, store pageTransactionStore, policy *retry.Policy) (*PageTransactionDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("workflow app and page transaction store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentPageTransaction)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[PageTransactionInput, struct{}]{Name: pageTransactionActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input PageTransactionInput) (struct{}, error) {
		if _, err := uuid.Parse(input.PageID); err != nil {
			return struct{}{}, fmt.Errorf("%w: invalid page transaction page ID", ErrPermanentPageTransaction)
		}
		if err := store.TrackPageTransactions(ctx, input); err != nil {
			return struct{}{}, classifyPageTransactionError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: pageTransactionQueue, StartToCloseTimeout: pageTransactionActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register page transaction activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(pageTransactionWorkflowName, func(ctx *selvedge.WorkflowContext, input PageTransactionInput) (struct{}, error) {
		if _, err := uuid.Parse(input.PageID); err != nil {
			return struct{}{}, fmt.Errorf("%w: invalid page transaction page ID", ErrPermanentPageTransaction)
		}
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &PageTransactionDispatcher{app: app, workflow: workflow}, nil
}

func (dispatcher *PageTransactionDispatcher) Admit(ctx context.Context, eventID string, input PageTransactionInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable page transaction dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("page transaction event ID must be a UUID: %w", err)
	}
	if _, err := uuid.Parse(input.PageID); err != nil {
		return "", fmt.Errorf("page transaction page ID must be a UUID: %w", err)
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "page-transaction:"+eventID, input)
}

type PostgresPageTransactionStore struct{ pool *pgxpool.Pool }

func NewPostgresPageTransactionStore(pool *pgxpool.Pool) (*PostgresPageTransactionStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresPageTransactionStore{pool: pool}, nil
}

type pageComponentRecord struct {
	componentType string
	transactionID string
	entityName    *string
	entityType    *string
	entityID      *string
}

func (store *PostgresPageTransactionStore) TrackPageTransactions(ctx context.Context, input PageTransactionInput) (resultErr error) {
	pageID, err := uuid.Parse(input.PageID)
	if err != nil {
		return fmt.Errorf("%w: invalid page transaction page ID", ErrPermanentPageTransaction)
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin page transaction write: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var workspaceID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT workspace_id FROM pages WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, pageID).Scan(&workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("load page for transaction tracking: %w", err)
	}
	var hasExistingLogs bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM page_logs WHERE page_id=$1)`, pageID).Scan(&hasExistingLogs); err != nil {
		return fmt.Errorf("check existing page transaction logs: %w", err)
	}
	oldComponents := extractPageComponents(input.OldDescriptionHTML)
	newComponents := extractPageComponents(input.NewDescriptionHTML)
	oldIDs := map[string]map[string]struct{}{"mention-component": {}, "image-component": {}}
	newIDs := map[string]map[string]struct{}{"mention-component": {}, "image-component": {}}
	for _, component := range oldComponents {
		oldIDs[component.componentType][component.transactionID] = struct{}{}
	}
	for _, component := range newComponents {
		newIDs[component.componentType][component.transactionID] = struct{}{}
	}
	deletedIDs := make([]uuid.UUID, 0)
	for componentType, ids := range oldIDs {
		for id := range ids {
			if _, stillPresent := newIDs[componentType][id]; !stillPresent {
				parsed, parseErr := uuid.Parse(id)
				if parseErr != nil {
					return fmt.Errorf("%w: invalid page transaction ID %q", ErrPermanentPageTransaction, id)
				}
				deletedIDs = append(deletedIDs, parsed)
			}
		}
	}
	newRecords := make([]pageComponentRecord, 0, len(newComponents))
	for _, component := range newComponents {
		if component.transactionID == "" {
			continue
		}
		if _, old := oldIDs[component.componentType][component.transactionID]; old && hasExistingLogs {
			continue
		}
		if _, parseErr := uuid.Parse(component.transactionID); parseErr != nil {
			return fmt.Errorf("%w: invalid page transaction ID %q", ErrPermanentPageTransaction, component.transactionID)
		}
		newRecords = append(newRecords, component)
	}
	if len(newRecords) > 0 {
		batch := &pgx.Batch{}
		for _, component := range newRecords {
			var entityID *uuid.UUID
			if component.entityID != nil {
				parsed, parseErr := uuid.Parse(*component.entityID)
				if parseErr != nil {
					return fmt.Errorf("%w: invalid page transaction entity identifier %q", ErrPermanentPageTransaction, *component.entityID)
				}
				entityID = &parsed
			}
			batch.Queue(`INSERT INTO page_logs(id,"transaction",page_id,workspace_id,entity_identifier,entity_name,entity_type,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8,NULL,NULL) ON CONFLICT (page_id,"transaction") DO NOTHING`, uuid.New(), uuid.MustParse(component.transactionID), pageID, workspaceID, entityID, component.entityName, component.entityType, time.Now().UTC())
		}
		results := tx.SendBatch(ctx, batch)
		for range newRecords {
			if _, err := results.Exec(); err != nil {
				_ = results.Close()
				return fmt.Errorf("insert page transaction log: %w", err)
			}
		}
		if err := results.Close(); err != nil {
			return fmt.Errorf("close page transaction batch: %w", err)
		}
	}
	if len(deletedIDs) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM page_logs WHERE "transaction"=ANY($1::uuid[])`, deletedIDs); err != nil {
			return fmt.Errorf("delete removed page transaction logs: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func classifyPageTransactionError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return fmt.Errorf("%w: invalid page transaction reference", ErrPermanentPageTransaction)
	}
	return err
}

func extractPageComponents(descriptionHTML any) []pageComponentRecord {
	markup, ok := descriptionHTML.(string)
	if !ok || markup == "" {
		return nil
	}
	root, err := htmlcompat.ParseFragment(strings.NewReader(markup), nil)
	if err != nil {
		return nil
	}
	var records []pageComponentRecord
	var walk func(*htmlcompat.Node)
	walk = func(node *htmlcompat.Node) {
		if node.Type == htmlcompat.ElementNode {
			switch strings.ToLower(node.Data) {
			case "mention-component":
				transaction := componentAttribute(node, "id")
				if transaction != nil && *transaction != "" {
					entityName := componentAttribute(node, "entity_name")
					entityID := componentAttribute(node, "entity_identifier")
					records = append(records, pageComponentRecord{componentType: "mention-component", transactionID: *transaction, entityName: entityName, entityID: entityID})
				}
			case "image-component":
				transaction := componentAttribute(node, "id")
				if transaction != nil && *transaction != "" {
					entityName := "image"
					entityID := componentAttribute(node, "src")
					records = append(records, pageComponentRecord{componentType: "image-component", transactionID: *transaction, entityName: &entityName, entityID: entityID})
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range root {
		walk(node)
	}
	return records
}

func componentAttribute(node *htmlcompat.Node, name string) *string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			value := attr.Val
			return &value
		}
	}
	return nil
}
