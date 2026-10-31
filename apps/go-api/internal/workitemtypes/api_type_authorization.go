package workitemtypes

import (
	"context"
	"strings"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type apiTypeResolution struct {
	id     uuid.UUID
	change string
	adopt  bool
}

// Authorize every explicit project and the work currently needed by resolve
// before receipt lookup. The schema and membership locks retain this decision
// through replay or application of the mutation.
func authorizeAPITypeAction(ctx context.Context, tx pgx.Tx, in APIMutation, access apiAccess) (apiTypeResolution, error) {
	if (in.Action == "create" || in.Action == "update" || in.Action == "delete") && !access.workspaceAdmin || in.Action == "import_to_project" && !access.projectAdmin {
		return apiTypeResolution{}, projectmutations.ErrForbidden
	}
	for _, project := range in.Changes.ProjectIDs {
		scope := in.APIScope
		scope.ProjectID = project
		projectAccess, err := authorizeAPI(ctx, tx, scope, true)
		if err != nil {
			return apiTypeResolution{}, err
		}
		if !projectAccess.projectAdmin {
			return apiTypeResolution{}, projectmutations.ErrForbidden
		}
	}
	if in.Action == "resolve" {
		return planAPITypeResolution(ctx, tx, in, access)
	}
	return apiTypeResolution{}, nil
}

func planAPITypeResolution(ctx context.Context, tx pgx.Tx, in APIMutation, access apiAccess) (apiTypeResolution, error) {
	plan := apiTypeResolution{}
	catalog, err := loadProjectCatalog(ctx, tx, in.WorkspaceID.String(), in.ProjectID.String(), access.projectAdmin)
	if err != nil {
		return plan, err
	}
	target := strings.TrimSpace(*in.Changes.Name)
	for _, item := range catalog.Types {
		if strings.TrimSpace(item.Name) == target && item.Available {
			plan.id, err = uuid.Parse(item.ID)
			return plan, err
		}
	}
	if !access.projectAdmin {
		return plan, projectmutations.ErrForbidden
	}
	plan.adopt, plan.change = true, "create"
	for _, item := range catalog.Types {
		if strings.TrimSpace(item.Name) != target {
			continue
		}
		plan.id, err = uuid.Parse(item.ID)
		if err != nil {
			return plan, err
		}
		if item.Active {
			plan.change = ""
			return plan, nil
		}
		plan.change = "update"
		break
	}
	if !access.workspaceAdmin {
		return plan, projectmutations.ErrForbidden
	}
	return plan, nil
}
