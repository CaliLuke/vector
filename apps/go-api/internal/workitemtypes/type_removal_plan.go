package workitemtypes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

type typeRemovalItem struct {
	ID, Project, Creator, Parent, Updated, Description string
	Draft                                              bool
	Values                                             map[string]any
	Transition                                         ItemTransition
}
type typeRemovalPlan struct {
	review      TypeRemovalReview
	items       []typeRemovalItem
	fingerprint string
}

func planTypeRemoval(ctx context.Context, tx pgx.Tx, workspace, actor string, command typeRemovalCommand) (typeRemovalPlan, error) {
	plan := typeRemovalPlan{items: []typeRemovalItem{}}
	// The schema lock precedes these row locks. Historical records and private
	// drafts participate in migration, but their identifiers never leave the plan.
	for _, table := range []string{"issues", "draft_issues"} {
		rows, err := tx.Query(ctx, `SELECT id::text,COALESCE(project_id::text,''),COALESCE(created_by_id::text,''),COALESCE(parent_id::text,''),updated_at::text,COALESCE(description_html,''),COALESCE(custom_properties,'{}') FROM `+table+` WHERE workspace_id=$1 AND type_id=$2 ORDER BY id FOR UPDATE`, workspace, command.TypeID)
		if err != nil {
			return plan, err
		}
		for rows.Next() {
			item := typeRemovalItem{Draft: table == "draft_issues"}
			if err = rows.Scan(&item.ID, &item.Project, &item.Creator, &item.Parent, &item.Updated, &item.Description, &item.Values); err != nil {
				rows.Close()
				return plan, err
			}
			plan.items = append(plan.items, item)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return plan, err
		}
	}
	projects := map[string]bool{}
	for i := range plan.items {
		item := &plan.items[i]
		if item.Draft {
			plan.review.Drafts++
		} else {
			plan.review.Items++
		}
		if item.Project != "" {
			projects[item.Project] = true
		}
		in := ItemValueInput{WorkspaceID: workspace, ActorID: actor, ProjectID: item.Project, PreviousProjectID: item.Project, CreatorID: item.Creator, ParentID: item.Parent, TypeID: command.ReplacementID, PreviousTypeID: command.TypeID, Stored: item.Values, Existing: true, TypeSpecified: true, Draft: item.Draft}
		var err error
		// Only this authorized schema migration resolves hidden labels internally.
		// It returns counts, never an item's description, values or field errors.
		item.Transition, err = planItemTransition(ctx, tx, in, item.Description, true)
		if err != nil {
			var invalid *ItemValueError
			var configuration *Error
			if errors.As(err, &invalid) || errors.As(err, &configuration) && (configuration.Status == 409 || configuration.Status == 422) {
				plan.review.InvalidItems++
				continue
			}
			return plan, err
		}
		if len(item.Transition.Preserved) > 0 {
			plan.review.Preserved++
		}
	}
	plan.review.Projects = len(projects)
	if plan.review.InvalidItems > 0 {
		plan.review.BlockedReason = "Some work items cannot use this replacement. Check project availability and required properties, or migrate those items individually first."
	}
	// Neighbours must be bound too: a newly added child can invalidate a formerly
	// valid replacement even though the source type's usage count is unchanged.
	var graph json.RawMessage
	err := tx.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(node ORDER BY draft,id),'[]') FROM (SELECT id,type_id,parent_id,project_id,updated_at,false AS draft FROM issues WHERE workspace_id=$1 UNION ALL SELECT id,type_id,parent_id,project_id,updated_at,true FROM draft_issues WHERE workspace_id=$1) node`, workspace).Scan(&graph)
	if err != nil {
		return plan, err
	}
	raw, err := json.Marshal(struct {
		Command typeRemovalCommand
		Items   []typeRemovalItem
		Graph   json.RawMessage
	}{command, plan.items, graph})
	if err != nil {
		return plan, err
	}
	sum := sha256.Sum256(raw)
	plan.fingerprint = hex.EncodeToString(sum[:])
	return plan, nil
}
