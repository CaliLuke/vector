package workitems

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

// Resolve parents only within the actor's readable workspace. Missing and
// inaccessible IDs deliberately have the same validation response.
func (r *Repository) parentKey(ctx context.Context, value any, options issueValidation) (string, []string, error) {
	id, err := djangoinput.UUID(value)
	if err != nil {
		return "", nil, invalid(map[string]any{"parent_id": []string{"Enter a valid parent work item ID."}})
	}
	var visible bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND `+liveIssue+` AND `+djangoapi.IssueVisibleSQL("i", "NULLIF($2,'')::uuid", "NULLIF($3,'')::uuid")+`)`, id.String(), options.ActorID, options.WorkspaceID).Scan(&visible)
	if err != nil {
		return "", nil, err
	}
	if !visible {
		return "", []string{"Select an available parent work item in this workspace."}, nil
	}
	return id.String(), nil, nil
}
