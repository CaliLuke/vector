package workitemtypes

import (
	"context"
	"slices"
	"strings"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

type FormulaAuthoringInput struct {
	Revision                   int64
	Definition                 property.Definition
	TypeID, ProjectID, IssueID string
}
type FormulaAuthoringResult struct {
	Valid           bool             `json:"valid"`
	ValidationError string           `json:"validation_error"`
	Active          bool             `json:"active"`
	EvaluatedAt     time.Time        `json:"evaluated_at"`
	Contexts        []PropertyChoice `json:"contexts"`
	Result          *FormulaResult   `json:"result,omitempty"`
}

// PreviewFormula compiles a proposed definition, then evaluates an authorized
// sample in one read-only snapshot. It never installs the proposed definition.
func (r *Repository) PreviewFormula(ctx context.Context, actor, slug string, input FormulaAuthoringInput) (_ FormulaAuthoringResult, resultErr error) {
	out := FormulaAuthoringResult{Contexts: []PropertyChoice{}}
	input.TypeID, input.ProjectID, input.IssueID = strings.ToLower(input.TypeID), strings.ToLower(input.ProjectID), strings.ToLower(input.IssueID)
	if _, err := uuid.Parse(input.TypeID); err != nil || input.IssueID != "" && input.ProjectID == "" {
		return out, reject(400, "Choose a type and the sample work item's project.")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return out, err
	}
	workspace, _, err := authorize(ctx, tx, actor, slug, true)
	if err != nil {
		return out, err
	}
	catalog, err := loadProperties(ctx, tx, workspace, actor, true)
	if err != nil {
		return out, err
	}
	if !catalog.Enabled || catalog.Revision != input.Revision {
		return out, reject(409, "Work item configuration changed. Reopen the property before previewing.")
	}
	if !slices.ContainsFunc(catalog.Types, func(kind Type) bool { return kind.ID == input.TypeID }) {
		return out, reject(404, "Work item type not found.")
	}
	if input.ProjectID != "" {
		if _, _, err = authorizeProject(ctx, tx, actor, slug, input.ProjectID, false); err != nil {
			return out, err
		}
	}
	scope := ItemSchemaInput{Mode: "create", ProjectID: input.ProjectID, IssueID: input.IssueID}
	if input.IssueID != "" {
		scope.Mode = "edit"
	}
	identity, err := schemaItemIdentity(ctx, tx, workspace, actor, scope)
	if err != nil {
		return out, err
	}
	if input.IssueID != "" && identity.TypeID != input.TypeID {
		return out, reject(422, "Choose a sample work item with the selected type.")
	}
	if err = tx.QueryRow(ctx, `SELECT transaction_timestamp()`).Scan(&out.EvaluatedAt); err != nil {
		return out, err
	}
	out.EvaluatedAt = out.EvaluatedAt.UTC().Truncate(time.Microsecond)
	proposed, err := formulaAuthoringDefinition(input.Definition, catalog)
	if err == nil {
		err = validatePropertyScopes(ctx, tx, workspace, proposed)
	}
	definitions, loadErr := itemPropertyDefinitions(ctx, tx, workspace, input.TypeID)
	if loadErr != nil {
		return out, loadErr
	}
	definitions = slices.DeleteFunc(definitions, func(def property.Definition) bool { return def.ID == proposed.ID })
	definitions = append(definitions, proposed)
	if err == nil {
		err = validateStandaloneFormula(proposed, formulaRecords(definitions))
	}
	if err != nil {
		out.ValidationError = err.Error()
		return out, tx.Commit(ctx)
	}
	out.Valid = true
	config, _, err := property.Effective(proposed, input.ProjectID, input.TypeID)
	if err != nil {
		return out, err
	}
	out.Active = config.Active
	out.Contexts, err = formulaAuthoringContexts(definitions, input.ProjectID, input.TypeID)
	if err != nil {
		return out, err
	}
	builtins, err := loadItemFormulaInputs(ctx, tx, workspace, scope, true)
	if err != nil {
		return out, err
	}
	values := schemaFormulaValues(definitions, scope, input.TypeID, identity.Values)
	if computed, ok := EvaluateItemFormulas(definitions, input.ProjectID, input.TypeID, builtins, values, out.EvaluatedAt)[proposed.ID]; ok {
		out.Result = &computed
	}
	return out, tx.Commit(ctx)
}

func formulaAuthoringDefinition(proposed property.Definition, catalog PropertyCatalog) (property.Definition, error) {
	proposed = canonicalProperty(proposed)
	if proposed.Kind != property.Formula {
		return proposed, reject(422, "Choose a formula property.")
	}
	var previous *property.Definition
	var used bool
	after := catalog
	after.Properties = slices.Clone(catalog.Properties)
	for i, record := range after.Properties {
		if record.Definition.ID == proposed.ID {
			previous, used = &record.Definition, record.Used
			after.Properties[i].Definition = proposed
		}
	}
	if previous == nil {
		if proposed.Name == "" {
			proposed.Name = formula.ReferenceKey(proposed.Title)
		}
		after.Properties = append(after.Properties, PropertyRecord{Definition: proposed})
	}
	if err := property.ValidateDefinition(proposed, previous, used); err != nil {
		return proposed, err
	}
	if err := validatePropertyChange(proposed, after); err != nil {
		return proposed, err
	}
	return proposed, nil
}

func formulaRecords(definitions []property.Definition) []PropertyRecord {
	records := make([]PropertyRecord, 0, len(definitions))
	for _, def := range definitions {
		records = append(records, PropertyRecord{Definition: def})
	}
	return records
}

func formulaAuthoringContexts(definitions []property.Definition, project, kind string) ([]PropertyChoice, error) {
	out := []PropertyChoice{}
	for _, def := range definitions {
		_, contextID, err := property.Effective(def, project, kind)
		if err != nil {
			return nil, err
		}
		name := "Default"
		for _, candidate := range def.Contexts {
			if candidate.ID == contextID {
				name = candidate.Name
			}
		}
		out = append(out, PropertyChoice{ID: def.ID, Name: def.Title + " · " + name})
	}
	return out, nil
}
