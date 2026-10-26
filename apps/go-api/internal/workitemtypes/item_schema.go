package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

type ItemSchemaInput struct {
	Preview                           *ItemFormulaPreview
	ProjectID, IssueID, DraftID, Mode string
	TypeID, ParentID                  *string
}
type ResolvedProperty struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Icon          string                 `json:"icon"`
	Kind          property.Kind          `json:"kind"`
	Configuration property.Configuration `json:"configuration"`
	ContextID     string                 `json:"context_id"`
}
type ItemReferenceChoice struct {
	PropertyChoice
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
}
type ItemSchema struct {
	StoredValues    map[string]any           `json:"stored_values"`
	ItemUpdatedAt   *time.Time               `json:"item_updated_at,omitempty"`
	Computed        map[string]FormulaResult `json:"computed"`
	EvaluatedAt     time.Time                `json:"evaluated_at"`
	Enabled         bool                     `json:"enabled"`
	Revision        int64                    `json:"revision"`
	TypeID          string                   `json:"type_id"`
	EligibleTypeIDs []string                 `json:"eligible_type_ids"`
	SelectionReason string                   `json:"selection_reason"`
	Properties      []ResolvedProperty       `json:"properties"`
	Members         []ItemReferenceChoice    `json:"members"`
	Releases        []ItemReferenceChoice    `json:"releases"`
}
type schemaIdentity struct {
	UpdatedAt                              *time.Time
	CreatorID, TypeID, ProjectID, ParentID string
	Values                                 map[string]any
}

// ItemSchema returns only the selected scope's effective presentation contract.
// It does not expose configuration belonging to other project/type contexts.
func (r *Repository) ItemSchema(ctx context.Context, actor, slug string, input ItemSchemaInput) (ItemSchema, error) {
	input.ProjectID = strings.ToLower(input.ProjectID)
	if input.TypeID != nil {
		selected := strings.ToLower(*input.TypeID)
		input.TypeID = &selected
	}
	out := ItemSchema{StoredValues: map[string]any{}, EligibleTypeIDs: []string{}, Computed: map[string]FormulaResult{}, EvaluatedAt: time.Now().UTC(), Properties: []ResolvedProperty{}, Members: []ItemReferenceChoice{}, Releases: []ItemReferenceChoice{}}
	if err := validateSchemaInput(input); err != nil {
		return out, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return out, err
	}
	workspace, _, err := authorize(ctx, tx, actor, slug, false)
	if err != nil {
		return out, err
	}
	projectVisible := false
	if input.ProjectID != "" && (input.Mode != "draft" || input.DraftID == "") {
		if _, _, err = authorizeProject(ctx, tx, actor, slug, input.ProjectID, false); err != nil {
			return out, err
		}
		projectVisible = true
	}
	identity, err := schemaItemIdentity(ctx, tx, workspace, actor, input)
	if err != nil {
		return out, err
	}
	if input.ProjectID != "" && !projectVisible {
		_, _, accessErr := authorizeProject(ctx, tx, actor, slug, input.ProjectID, false)
		projectVisible = accessErr == nil
		if accessErr != nil {
			var denied *Error
			ownedSavedTarget := input.Mode == "draft" && input.DraftID != "" && identity.ProjectID == input.ProjectID
			if !ownedSavedTarget || !errors.As(accessErr, &denied) || denied.Status != 403 && denied.Status != 404 {
				return out, accessErr
			}
		}
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT enabled FROM work_item_type_settings WHERE workspace_id=$1),false),COALESCE((SELECT revision FROM work_item_type_settings WHERE workspace_id=$1),0)`, workspace).Scan(&out.Enabled, &out.Revision); err != nil {
		return out, err
	}
	if !out.Enabled {
		return out, tx.Commit(ctx)
	}
	out.ItemUpdatedAt = identity.UpdatedAt
	selected := identity.TypeID
	if input.TypeID != nil {
		selected = *input.TypeID
	}
	parent, err := schemaParent(ctx, tx, workspace, actor, input, identity)
	if err != nil {
		return out, err
	}
	valueInput := ItemValueInput{WorkspaceID: workspace, ProjectID: input.ProjectID, PreviousProjectID: identity.ProjectID, TypeID: selected, PreviousTypeID: identity.TypeID, ParentID: parent, CreatorID: identity.CreatorID, Draft: input.Mode == "draft", Creating: input.Mode == "create" || input.Mode == "publish" || input.Mode == "draft" && input.DraftID == "", Existing: input.IssueID != "" || input.DraftID != "", ProjectChanged: identity.ProjectID != input.ProjectID}
	if err = resolveSchemaType(ctx, tx, valueInput, &out); err != nil {
		return out, err
	}
	if out.TypeID == "" {
		return out, tx.Commit(ctx)
	}
	definitions, err := itemPropertyDefinitions(ctx, tx, workspace, out.TypeID)
	if err != nil {
		return out, err
	}
	projection, err := CompileItemProjection(definitions, input.ProjectID, out.TypeID)
	if err != nil {
		return out, err
	}
	for _, field := range projection.properties {
		resolved := field.resolved
		if !resolved.Configuration.Active || valueInput.Draft && resolved.Kind == property.Cascading {
			continue
		}
		if value, exists := identity.Values[resolved.ID]; exists && resolved.Kind != property.Formula {
			out.StoredValues[resolved.ID] = value
		}
		out.Properties = append(out.Properties, resolved)
	}
	allowed, err := itemValueReferences(ctx, tx, valueInput, definitions)
	if err != nil {
		return out, err
	}
	if input.ProjectID != "" && !projectVisible {
		allowed = func(property.Kind, string) bool { return false }
	}
	if slices.ContainsFunc(out.Properties, func(p ResolvedProperty) bool { return p.Kind == property.Member }) {
		out.Members, err = schemaMembers(ctx, tx, workspace, allowed, definitions, identity.Values)
		if err != nil {
			return out, err
		}
	}
	if slices.ContainsFunc(out.Properties, func(p ResolvedProperty) bool { return p.Kind == property.Release }) {
		out.Releases, err = schemaReleases(ctx, tx, workspace, input.ProjectID, projectVisible, out.Properties, identity.Values)
		if err != nil {
			return out, err
		}
	}
	if err = schemaFormulas(ctx, tx, workspace, input, identity, projectVisible, definitions, projection, &out); err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func validateSchemaInput(input ItemSchemaInput) error {
	if !slices.Contains([]string{"create", "edit", "draft", "publish"}, input.Mode) {
		return reject(400, "Choose a valid editing mode.")
	}
	if input.Mode != "draft" && input.ProjectID == "" {
		return reject(400, "A project is required.")
	}
	if input.Mode == "edit" && input.IssueID == "" || input.Mode != "edit" && input.IssueID != "" {
		return reject(400, "Select the work item to edit.")
	}
	if input.Mode == "publish" && input.DraftID == "" || (input.Mode == "create" || input.Mode == "edit") && input.DraftID != "" {
		return reject(400, "Select a private draft for this action.")
	}
	return nil
}

func schemaItemIdentity(ctx context.Context, tx pgx.Tx, workspace, actor string, input ItemSchemaInput) (schemaIdentity, error) {
	out := schemaIdentity{CreatorID: actor, ProjectID: input.ProjectID, Values: map[string]any{}}
	var raw []byte
	var err error
	if input.IssueID != "" {
		err = tx.QueryRow(ctx, `SELECT COALESCE(i.created_by_id::text,''),COALESCE(i.type_id::text,''),i.project_id::text,COALESCE(i.parent_id::text,''),i.custom_properties,i.updated_at FROM issues i JOIN projects p ON p.id=i.project_id WHERE i.id=$1 AND i.workspace_id=$2 AND i.project_id=$3 AND i.deleted_at IS NULL AND (p.guest_view_all_features OR i.created_by_id=$4 OR EXISTS(SELECT 1 FROM project_members m WHERE m.project_id=i.project_id AND m.member_id=$4 AND m.role>=15 AND m.is_active AND m.deleted_at IS NULL))`, input.IssueID, workspace, input.ProjectID, actor).Scan(&out.CreatorID, &out.TypeID, &out.ProjectID, &out.ParentID, &raw, &out.UpdatedAt)
	} else if input.DraftID != "" {
		err = tx.QueryRow(ctx, `SELECT COALESCE(type_id::text,''),COALESCE(project_id::text,''),COALESCE(parent_id::text,''),custom_properties,updated_at FROM draft_issues WHERE id=$1 AND workspace_id=$2 AND created_by_id=$3 AND deleted_at IS NULL`, input.DraftID, workspace, actor).Scan(&out.TypeID, &out.ProjectID, &out.ParentID, &raw, &out.UpdatedAt)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return out, reject(404, "Work item or private draft not found.")
	}
	if err != nil {
		return out, err
	}
	if raw != nil {
		err = json.Unmarshal(raw, &out.Values)
	}
	return out, err
}

func schemaMembers(ctx context.Context, tx pgx.Tx, workspace string, allowed property.ReferenceAllowed, definitions []property.Definition, values map[string]any) ([]ItemReferenceChoice, error) {
	members, err := propertyChoices(ctx, tx, `SELECT u.id::text,COALESCE(NULLIF(u.display_name,''),NULLIF(trim(u.first_name||' '||u.last_name),''),u.username) FROM users u JOIN workspace_members m ON m.member_id=u.id WHERE m.workspace_id=$1 AND m.is_active AND m.deleted_at IS NULL AND u.is_active ORDER BY u.id`, workspace)
	if err != nil {
		return nil, err
	}
	out := []ItemReferenceChoice{}
	seen := map[string]bool{}
	for _, member := range members {
		eligible := allowed(property.Member, member.ID)
		reason := ""
		if !eligible {
			reason = "This member cannot access this work item."
		}
		out = append(out, ItemReferenceChoice{PropertyChoice: member, Eligible: eligible, Reason: reason})
		seen[member.ID] = true
	}
	for _, def := range definitions {
		if def.Kind != property.Member {
			continue
		}
		for _, id := range memberValueIDs(values[def.ID]) {
			if seen[id] {
				continue
			}
			out = append(out, ItemReferenceChoice{PropertyChoice: PropertyChoice{ID: id, Name: "Unavailable member"}, Reason: "This selection is no longer available."})
			seen[id] = true
		}
	}
	return out, nil
}

func schemaReleases(ctx context.Context, tx pgx.Tx, workspace, project string, visible bool, properties []ResolvedProperty, values map[string]any) ([]ItemReferenceChoice, error) {
	out := []ItemReferenceChoice{}
	seen := map[string]bool{}
	if visible {
		choices, err := propertyChoices(ctx, tx, `SELECT id::text,name FROM work_item_releases WHERE workspace_id=$1 AND project_id=$2 ORDER BY name,id`, workspace, project)
		if err != nil {
			return nil, err
		}
		for _, choice := range choices {
			out = append(out, ItemReferenceChoice{PropertyChoice: choice, Eligible: true})
			seen[choice.ID] = true
		}
	}
	for _, def := range properties {
		if def.Kind != property.Release {
			continue
		}
		for _, id := range memberValueIDs(values[def.ID]) {
			if seen[id] {
				continue
			}
			out = append(out, ItemReferenceChoice{PropertyChoice: PropertyChoice{ID: id, Name: "Unavailable release"}, Reason: "This selection is no longer available."})
			seen[id] = true
		}
	}
	return out, nil
}
