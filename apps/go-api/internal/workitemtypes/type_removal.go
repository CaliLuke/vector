package workitemtypes

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TypeRemovalInput struct {
	Revision                        int64
	TypeID, ReplacementID, ReviewID string
}
type TypeRemovalReview struct {
	ID            string `json:"id"`
	Items         int    `json:"items"`
	Drafts        int    `json:"drafts"`
	Projects      int    `json:"projects"`
	Preserved     int    `json:"preserved"`
	InvalidItems  int    `json:"invalid_items"`
	BlockedReason string `json:"blocked_reason"`
}
type typeRemovalCommand struct {
	Operation     string `json:"operation"`
	Revision      int64  `json:"revision"`
	TypeID        string `json:"type_id"`
	ReplacementID string `json:"replacement_id"`
}

func normalizeTypeRemoval(in TypeRemovalInput) (typeRemovalCommand, error) {
	source, err := uuid.Parse(in.TypeID)
	if err != nil {
		return typeRemovalCommand{}, reject(422, "Choose a work item type to delete.")
	}
	replacement, err := uuid.Parse(in.ReplacementID)
	if err != nil || source == replacement {
		return typeRemovalCommand{}, reject(422, "Choose a different replacement type.")
	}
	return typeRemovalCommand{Operation: "replace_and_delete_type", Revision: in.Revision, TypeID: source.String(), ReplacementID: replacement.String()}, nil
}
func validateTypeRemoval(command typeRemovalCommand, catalog Catalog) error {
	if !catalog.Enabled || catalog.Revision != command.Revision {
		return reject(409, "Work item types changed. Refresh the review.")
	}
	index := slices.IndexFunc(catalog.Types, func(t Type) bool { return t.ID == command.TypeID })
	if index < 0 {
		return reject(404, "Work item type not found.")
	}
	if catalog.Types[index].Default {
		return reject(422, "The default type cannot be deleted.")
	}
	index = slices.IndexFunc(catalog.Types, func(t Type) bool { return t.ID == command.ReplacementID })
	if index < 0 || !catalog.Types[index].Active {
		return reject(422, "Choose an active replacement type in this workspace.")
	}
	return nil
}

func (r *Repository) PreviewTypeRemoval(ctx context.Context, actor, slug string, input TypeRemovalInput) (TypeRemovalReview, error) {
	command, err := normalizeTypeRemoval(input)
	if err != nil {
		return TypeRemovalReview{}, err
	}
	return withSchemaWrite(ctx, r, actor, slug, func(tx pgx.Tx, workspace string) (TypeRemovalReview, error) {
		catalog, err := loadCatalog(ctx, tx, workspace, true)
		if err != nil {
			return TypeRemovalReview{}, err
		}
		if err = validateTypeRemoval(command, catalog); err != nil {
			return TypeRemovalReview{}, err
		}
		plan, err := planTypeRemoval(ctx, tx, workspace, actor, command)
		if err != nil || plan.review.InvalidItems > 0 {
			return plan.review, err
		}
		// Exercise the final writer and its constraints, including configuration
		// cleanup. Roll back values, history and activity before saving the receipt.
		dry, err := tx.Begin(ctx)
		if err != nil {
			return plan.review, err
		}
		applyErr := applyTypeRemoval(ctx, dry, workspace, actor, command, plan)
		if err = dry.Rollback(ctx); err != nil {
			return plan.review, err
		}
		if applyErr != nil {
			var constraint *pgconn.PgError
			if errors.As(applyErr, &constraint) && constraint.Code == "23514" {
				plan.review.BlockedReason = "The replacement would invalidate work item relationships. Migrate affected items individually before deleting this type."
				return plan.review, nil
			}
			return plan.review, applyErr
		}
		plan.review.ID, err = saveSchemaReview(ctx, tx, workspace, actor, command, plan.fingerprint)
		return plan.review, err
	})
}

func (r *Repository) CommitTypeRemoval(ctx context.Context, actor, slug string, input TypeRemovalInput) (Catalog, error) {
	command, err := normalizeTypeRemoval(input)
	if err != nil {
		return Catalog{}, err
	}
	return withSchemaWrite(ctx, r, actor, slug, func(tx pgx.Tx, workspace string) (Catalog, error) {
		receipt, err := loadSchemaReview(ctx, tx, workspace, actor, input.ReviewID, command)
		if err != nil {
			return Catalog{}, err
		}
		catalog, err := loadCatalog(ctx, tx, workspace, true)
		if err != nil || receipt.Committed {
			return catalog, err
		}
		if receipt.Expired {
			return Catalog{}, rejectedTypeRemoval("This review expired. Refresh it before confirming.")
		}
		if err = validateTypeRemoval(command, catalog); err != nil {
			return Catalog{}, rejectedTypeRemoval("Work item types changed. Refresh the review.")
		}
		plan, err := planTypeRemoval(ctx, tx, workspace, actor, command)
		if err != nil {
			return Catalog{}, err
		}
		if plan.review.InvalidItems > 0 || plan.fingerprint != receipt.Fingerprint {
			return Catalog{}, rejectedTypeRemoval("Affected work items changed. Refresh the review before confirming.")
		}
		if err = applyTypeRemoval(ctx, tx, workspace, actor, command, plan); err != nil {
			return Catalog{}, err
		}
		if err = finishSchemaReview(ctx, tx, workspace, input.ReviewID); err != nil {
			return Catalog{}, err
		}
		return loadCatalog(ctx, tx, workspace, true)
	})
}

func rejectedTypeRemoval(message string) error {
	// Only called after an exact actor-bound, uncommitted receipt was loaded.
	return &Error{Status: 409, Message: message, ReviewAgain: true}
}
