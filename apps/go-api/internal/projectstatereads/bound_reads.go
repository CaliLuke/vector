package projectstatereads

import "uuid"

func validateBoundStateRead(user string, request APIV1Request) error {
	if request.Operation != "list" && request.Operation != "detail" {
		return ErrForbidden
	}
	if *request.ExpectedWorkspaceID == uuid.Nil() || request.Params["slug"] == "" {
		return ErrForbidden
	}
	for _, field := range []string{"fields", "expand"} {
		if request.Present[field] || request.Params[field] != "" {
			return ErrForbidden
		}
	}
	ids := []string{user}
	if project := request.Params["project_id"]; project != "" {
		ids = append(ids, project)
	}
	if request.Operation == "detail" {
		ids = append(ids, request.Params["state_id"])
	}
	for _, value := range ids {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil() {
			return ErrForbidden
		}
	}
	return nil
}

// Workspace admission is independent of accessible project rows, so an empty
// aggregate remains distinguishable from denied scope. Count and rows share the
// authorization snapshot, including the immutable workspace identity.
func boundStateReadQuery(user string, request APIV1Request, limit, offset int) (string, []any) {
	var project, target any
	if request.Params["project_id"] != "" {
		project = request.Params["project_id"]
	}
	if request.Operation == "detail" {
		target = request.Params["state_id"]
	}
	return `WITH workspace_scope AS MATERIALIZED (
 SELECT w.id FROM workspaces w WHERE w.id=$1::uuid AND w.slug=$2 AND w.deleted_at IS NULL
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=$3::uuid AND u.is_active)
 AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id
  AND wm.member_id=$3::uuid AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20))
), project_scope AS MATERIALIZED (
 SELECT p.id,p.workspace_id FROM projects p JOIN workspace_scope w ON w.id=p.workspace_id
 WHERE ($4::uuid IS NULL OR p.id=$4::uuid) AND p.deleted_at IS NULL AND p.archived_at IS NULL
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=w.id
  AND pm.member_id=$3::uuid AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20))
), state_rows AS MATERIALIZED (
 SELECT to_jsonb(state) AS data,
  row_number() OVER(ORDER BY state.project_id,state.sequence,state.id) AS ordinal
 FROM states state JOIN project_scope p ON p.id=state.project_id AND p.workspace_id=state.workspace_id
 WHERE state.deleted_at IS NULL AND state."group"<>'triage' AND NOT state.is_triage
 AND ($5::uuid IS NULL OR state.id=$5::uuid)
), page AS (
 SELECT data,ordinal FROM state_rows ORDER BY ordinal LIMIT $6 OFFSET $7
)
SELECT jsonb_build_object('total_count',(SELECT count(*) FROM state_rows),
 'has_next',(SELECT count(*) FROM state_rows)-$7::bigint>$6::bigint,
 'results',coalesce((SELECT jsonb_agg(data ORDER BY ordinal) FROM page),'[]'::jsonb))
 FROM workspace_scope WHERE $4::uuid IS NULL OR EXISTS(SELECT 1 FROM project_scope)`,
		[]any{request.ExpectedWorkspaceID, request.Params["slug"], user, project, target, limit, offset}
}
