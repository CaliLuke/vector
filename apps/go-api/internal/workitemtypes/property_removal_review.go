package workitemtypes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"reflect"
	"slices"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PropertyReview struct {
	ID       string `json:"id"`
	Items    int    `json:"items"`
	Types    int    `json:"types"`
	Projects int    `json:"projects"`
}

type propertyRemoval struct {
	Revision   int64                `json:"revision"`
	Action     string               `json:"action"`
	ID         string               `json:"id"`
	TypeID     string               `json:"type_id"`
	Permanent  bool                 `json:"permanent"`
	Definition *property.Definition `json:"definition,omitempty"`
}

func removalCommand(input PropertyInput) (propertyRemoval, error) {
	if input.Action == "update" {
		return propertyEditCommand(input)
	}
	if input.Action != "delete" && input.Action != "detach" {
		return propertyRemoval{}, reject(422, "Review a property deletion or detachment.")
	}
	if input.Action == "detach" && (input.TypeID == "" || input.Permanent) {
		return propertyRemoval{}, reject(422, "Detaching a property must preserve its values.")
	}
	if _, err := uuid.Parse(input.ID); err != nil {
		return propertyRemoval{}, reject(422, "Choose a property.")
	}
	if input.Action == "delete" && input.TypeID != "" {
		return propertyRemoval{}, reject(422, "Property deletion applies to every type.")
	}
	return propertyRemoval{Revision: input.Revision, Action: input.Action, ID: input.ID, TypeID: input.TypeID, Permanent: input.Permanent}, nil
}

func (r *Repository) PreviewPropertyRemoval(ctx context.Context, actor, slug string, input PropertyInput) (PropertyReview, error) {
	command, err := removalCommand(input)
	if err != nil {
		return PropertyReview{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return PropertyReview{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	workspace, _, err := lockWorkspaceSchema(ctx, tx, actor, slug)
	if err != nil {
		return PropertyReview{}, err
	}
	catalog, err := loadProperties(ctx, tx, workspace, actor, true)
	if err != nil {
		return PropertyReview{}, err
	}
	if !catalog.Enabled || catalog.Revision != command.Revision {
		return PropertyReview{}, reject(409, "Work item configuration changed. Refresh the review.")
	}
	// Validate references and dependencies using the actual removal writer, then
	// roll back only this dry run before persisting its review receipt.
	if _, err = tx.Exec(ctx, `SAVEPOINT property_removal_preview`); err != nil {
		return PropertyReview{}, err
	}
	if err = writeReviewedProperty(ctx, tx, workspace, actor, command, catalog); err != nil {
		return PropertyReview{}, err
	}
	if _, err = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT property_removal_preview`); err != nil {
		return PropertyReview{}, err
	}
	plan, err := planPropertyRemoval(ctx, tx, workspace, command, catalog)
	if err != nil {
		return PropertyReview{}, err
	}
	raw, err := json.Marshal(command)
	if err != nil {
		return PropertyReview{}, err
	}
	review := plan.review
	review.ID = uuid.NewString()
	if _, err = tx.Exec(ctx, `INSERT INTO work_item_schema_reviews(id,workspace_id,actor_id,command,fingerprint) VALUES($1,$2,$3,$4,$5)`, review.ID, workspace, actor, raw, plan.fingerprint); err != nil {
		return PropertyReview{}, err
	}
	return review, tx.Commit(ctx)
}

func (r *Repository) commitPropertyRemoval(ctx context.Context, actor, slug string, input PropertyInput) (PropertyCatalog, error) {
	command, err := removalCommand(input)
	if err != nil {
		return PropertyCatalog{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return PropertyCatalog{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	workspace, _, err := lockWorkspaceSchema(ctx, tx, actor, slug)
	if err != nil {
		return PropertyCatalog{}, err
	}
	var raw []byte
	var fingerprint string
	var committed, expired bool
	err = tx.QueryRow(ctx, `SELECT command,fingerprint,committed_at IS NOT NULL,created_at<clock_timestamp()-interval '30 minutes' FROM work_item_schema_reviews WHERE id=$1 AND workspace_id=$2 AND actor_id=$3 FOR UPDATE`, input.ReviewID, workspace, actor).Scan(&raw, &fingerprint, &committed, &expired)
	if errors.Is(err, pgx.ErrNoRows) {
		return PropertyCatalog{}, reject(404, "Review not found.")
	}
	if err != nil {
		return PropertyCatalog{}, err
	}
	var saved propertyRemoval
	if err = json.Unmarshal(raw, &saved); err != nil {
		return PropertyCatalog{}, err
	}
	if !reflect.DeepEqual(saved, command) {
		return PropertyCatalog{}, reject(409, "The proposed change no longer matches this review.")
	}
	catalog, err := loadProperties(ctx, tx, workspace, actor, true)
	if err != nil {
		return PropertyCatalog{}, err
	}
	if committed {
		return catalog, tx.Commit(ctx)
	}
	if expired || catalog.Revision != command.Revision {
		return PropertyCatalog{}, reject(409, "This review is out of date. Refresh it before confirming.")
	}
	plan, err := planPropertyRemoval(ctx, tx, workspace, command, catalog)
	if err != nil {
		return PropertyCatalog{}, err
	}
	if plan.fingerprint != fingerprint {
		return PropertyCatalog{}, reject(409, "Affected work items changed. Refresh the review before confirming.")
	}
	if err = writeReviewedProperty(ctx, tx, workspace, actor, command, catalog); err != nil {
		return PropertyCatalog{}, err
	}
	if err = applyPropertyRemoval(ctx, tx, workspace, actor, command, plan); err != nil {
		return PropertyCatalog{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, workspace); err != nil {
		return PropertyCatalog{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE work_item_schema_reviews SET committed_at=clock_timestamp() WHERE id=$1`, input.ReviewID); err != nil {
		return PropertyCatalog{}, err
	}
	out, err := loadProperties(ctx, tx, workspace, actor, true)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

type removalItem struct {
	ID, Project, Type, Updated, Description, HTML string
	Draft                                         bool
	Value                                         any
}
type propertyRemovalPlan struct {
	review      PropertyReview
	fingerprint string
	items       []removalItem
	definition  property.Definition
}

func planPropertyRemoval(ctx context.Context, tx pgx.Tx, workspace string, command propertyRemoval, catalog PropertyCatalog) (propertyRemovalPlan, error) {
	plan := propertyRemovalPlan{}
	index := slices.IndexFunc(catalog.Properties, func(record PropertyRecord) bool { return record.Definition.ID == command.ID })
	if index < 0 {
		return plan, reject(404, "Property not found.")
	}
	plan.definition = catalog.Properties[index].Definition
	types := map[string]bool{}
	projects := map[string]bool{}
	for _, link := range catalog.Attachments {
		if (command.Action != "detach" || link.TypeID == command.TypeID) && slices.Contains(link.PropertyIDs, command.ID) {
			types[link.TypeID] = true
		}
	}
	if command.Action == "detach" && len(types) == 0 {
		return plan, reject(404, "Property attachment not found.")
	}
	// Include archived and trashed records as well as private drafts: restoration
	// must not bring back a value whose definition no longer exists. Only counts
	// leave the server; descriptions and identifiers remain private.
	rows, err := tx.Query(ctx, `SELECT id::text,COALESCE(project_id::text,''),COALESCE(type_id::text,''),updated_at::text,COALESCE(description_html,''),custom_properties->$2,false FROM issues WHERE workspace_id=$1 AND custom_properties ? $2 AND ($3='' OR type_id=NULLIF($3,'')::uuid) UNION ALL SELECT id::text,COALESCE(project_id::text,''),COALESCE(type_id::text,''),updated_at::text,COALESCE(description_html,''),custom_properties->$2,true FROM draft_issues WHERE workspace_id=$1 AND custom_properties ? $2 AND ($3='' OR type_id=NULLIF($3,'')::uuid) ORDER BY 7,1`, workspace, command.ID, command.TypeID)
	if err != nil {
		return plan, err
	}
	for rows.Next() {
		var item removalItem
		var raw []byte
		if err = rows.Scan(&item.ID, &item.Project, &item.Type, &item.Updated, &item.Description, &raw, &item.Draft); err != nil {
			rows.Close()
			return plan, err
		}
		if err = json.Unmarshal(raw, &item.Value); err != nil {
			rows.Close()
			return plan, err
		}
		if command.Definition != nil {
			scope := property.Scope{ProjectID: item.Project, TypeID: item.Type}
			removed, changeErr := property.CascadingSelectionRequiresPreservation(plan.definition, *command.Definition, scope, scope, item.Value)
			if changeErr != nil {
				rows.Close()
				return plan, reject(422, changeErr.Error())
			}
			if !removed {
				continue
			}
		}
		plan.items = append(plan.items, item)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return plan, err
	}
	for i := range plan.items {
		item := &plan.items[i]
		if item.Project != "" {
			projects[item.Project] = true
		}
		config, _, e := property.Effective(plan.definition, item.Project, item.Type)
		if e != nil {
			return plan, e
		}
		labels, e := preservationLabels(ctx, tx, workspace, item.Project, plan.definition.Kind, item.Value)
		if e != nil {
			return plan, e
		}
		item.HTML, e = property.PreservationHTML(plan.definition, config, item.Value, labels)
		if e != nil {
			return plan, e
		}
		if !command.Permanent && item.HTML != "" && len(item.Description)+len("<h3>"+html.EscapeString(preservationReason(command, plan.definition))+"</h3>")+len(item.HTML) > content.MaxContentBytes {
			return plan, reject(422, "Preserving these values would exceed a work item's description limit.")
		}
	}
	plan.review = PropertyReview{Items: len(plan.items), Types: len(types), Projects: len(projects)}
	raw, err := json.Marshal(struct {
		Command    propertyRemoval
		Items      []removalItem
		Definition property.Definition
	}{command, plan.items, plan.definition})
	if err != nil {
		return plan, err
	}
	sum := sha256.Sum256(raw)
	plan.fingerprint = hex.EncodeToString(sum[:])
	return plan, nil
}
