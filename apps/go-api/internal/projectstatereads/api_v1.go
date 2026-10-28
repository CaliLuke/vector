package projectstatereads

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/jackc/pgx/v5"
)

// APIV1Request keeps API serializer and pagination behavior separate from the
// session-authenticated project-state endpoints in this package.
type APIV1Request = djangoapi.ProjectReadRequest

// AuthorizeAPIV1 mirrors ProjectEntityPermission for safe API requests: any
// active project member, including guests, may read a project state.
func (repository *Repository) AuthorizeAPIV1(ctx context.Context, user, slug, project string, write bool) error {
	allowed, err := djangoapi.ProjectEntityAllowed(ctx, repository.database, user, slug, project, write)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

var apiStateFields = strings.Fields("id created_at updated_at deleted_at created_by updated_by workspace project name description color slug sequence group is_triage default external_source external_id")

// ReadAPIV1 returns the API StateSerializer representation, which is distinct
// from the compact state shape used by the app-facing endpoints above.
func (repository *Repository) ReadAPIV1(ctx context.Context, user string, request APIV1Request) (any, error) {
	if request.ExpectedWorkspaceID != nil {
		if err := validateBoundStateRead(user, request); err != nil {
			return nil, err
		}
	}
	p := request.Params
	detail := request.Operation == "detail"
	limit, page, offset := 1000, 0, 0
	if !detail {
		var err error
		limit, page, offset, err = djangoapi.PageWindow(djangoapi.Query{Params: p, Present: request.Present})
		if err != nil {
			return nil, err
		}
	}

	query, args := stateReadQuery(user, request, limit, offset)
	if request.ExpectedWorkspaceID != nil {
		query, args = boundStateReadQuery(user, request, limit, offset)
	}
	value, err := djangoapi.JSON(ctx, repository.database, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if request.ExpectedWorkspaceID != nil {
				return nil, ErrForbidden
			}
			return nil, ErrNotFound
		}
		return nil, err
	}

	doc := value.(map[string]any)
	rows := doc["results"].([]any)
	if detail && len(rows) == 0 {
		return nil, ErrNotFound
	}

	fields, expand := p["fields"], p["expand"]
	if detail {
		// The detail view passes the complete raw query through its serializer.
		// Missing fields/expand are equivalent to empty values.
		if !request.Present["fields"] {
			fields = ""
		}
		if !request.Present["expand"] {
			expand = ""
		}
	}
	for i, raw := range rows {
		projected, err := djangoapi.Serialize(ctx, repository.database, raw.(map[string]any), apiStateFields, fields, expand, false)
		if err != nil {
			return nil, err
		}
		if sequence, ok := projected["sequence"].(json.Number); ok {
			value, err := sequence.Float64()
			if err != nil {
				return nil, err
			}
			projected["sequence"] = json.Number(pythoncompat.FloatString(value))
		}
		if request.ExpectedWorkspaceID != nil && p["include_revision"] == "true" {
			if revisionErr := addStateRevision(raw.(map[string]any), projected); revisionErr != nil {
				return nil, revisionErr
			}
		}
		rows[i] = projected
	}
	if detail {
		return rows[0], nil
	}
	total, err := strconv.Atoi(doc["total_count"].(json.Number).String())
	if err != nil {
		return nil, err
	}
	result := djangoapi.Envelope(rows, total, limit, page)
	result["next_page_results"] = doc["has_next"]
	return result, nil
}

func stateReadQuery(user string, request APIV1Request, limit, offset int) (string, []any) {
	p := request.Params
	detail := request.Operation == "detail"
	predicate := ""
	slice := " LIMIT $4 OFFSET $5"
	hasNext := "(SELECT count(*) FROM state_rows)-$5::bigint>$4::bigint"
	if detail {
		predicate = " AND state.id=$4::uuid"
		slice = ""
		hasNext = "false"
	}
	query := `WITH state_rows AS MATERIALIZED (
 SELECT to_jsonb(state) AS data,
   row_number() OVER (ORDER BY state.sequence) AS ordinal
 FROM states state
 JOIN projects project ON project.id=state.project_id
 JOIN workspaces workspace ON workspace.id=state.workspace_id
 WHERE workspace.slug=$1
   AND project.id=$2::uuid
   AND project.archived_at IS NULL
   AND state.deleted_at IS NULL
   AND state."group" <> 'triage'
   AND state.is_triage=false
   AND EXISTS (
     SELECT 1 FROM project_members project_member
     WHERE project_member.project_id=project.id
       AND project_member.member_id=$3::uuid
       AND project_member.is_active=true
       AND project_member.deleted_at IS NULL
   )` + predicate + `
)
SELECT jsonb_build_object(
  'total_count',(SELECT count(*) FROM state_rows),
  'has_next',` + hasNext + `,
  'results',COALESCE((SELECT jsonb_agg(data ORDER BY ordinal) FROM (
    SELECT data,ordinal FROM state_rows ORDER BY ordinal` + slice + `
  ) page),'[]'::jsonb)
)`
	args := []any{p["slug"], p["project_id"], user}
	if detail {
		args = append(args, p["state_id"])
	} else {
		args = append(args, limit, offset)
	}
	return query, args
}
