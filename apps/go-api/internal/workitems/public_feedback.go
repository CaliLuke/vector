package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PublicFeedbackScope is resolved by the public project-board read boundary
// before work-item operations are called.
type PublicFeedbackScope struct {
	Anchor        string
	WorkspaceID   string
	WorkspaceSlug string
	ProjectID     string
	IntakeID      *string
	VotesEnabled  bool
}

const publicVoteDisabled = "Voting is not enabled for this Project Board"

func publicFeedbackObjectRootError(input djangoinput.Input) error {
	if input.Kind == "" {
		return nil
	}
	message := "Invalid data. Expected a dictionary, but got " + input.Kind + "."
	if input.Null {
		message = "No data provided"
	}
	return invalid(map[string]any{"non_field_errors": []string{message}})
}

func publicFeedbackNestedIssueError() error {
	return invalid(map[string]any{"issue": []string{"Expected an object."}})
}

func (repository *Repository) PublicVoteList(ctx context.Context, scope PublicFeedbackScope, issueID string) (Result, error) {
	if !scope.VotesEnabled {
		return Result{Status: 200, Body: []any{}}, nil
	}
	rows, err := repository.rows(ctx, `
SELECT jsonb_build_object(
  'actor', vote.actor_id,
  'actor_detail', jsonb_build_object(
    'id', actor.id,
    'first_name', actor.first_name,
    'last_name', actor.last_name,
    'avatar', actor.avatar,
    'display_name', actor.display_name,
    'is_bot', actor.is_bot
  )||`+djangoapi.FileAssetURLFieldSQL("actor.avatar_asset_id", "actor.avatar", "avatar_url")+`,
  'issue', vote.issue_id,
  'project', vote.project_id,
  'vote', vote.vote,
  'workspace', vote.workspace_id
)
FROM issue_votes vote
JOIN users actor ON actor.id = vote.actor_id
WHERE vote.issue_id = $1::uuid
  AND vote.project_id = $2::uuid
  AND vote.workspace_id = $3::uuid
  AND vote.deleted_at IS NULL
ORDER BY vote.created_at DESC
`, issueID, scope.ProjectID, scope.WorkspaceID)
	if err != nil {
		return Result{}, err
	}
	results := make([]any, len(rows))
	for index, row := range rows {
		results[index] = row
	}
	return Result{Status: 200, Body: results}, nil
}

func (repository *Repository) PublicVoteCreate(
	ctx context.Context,
	scope PublicFeedbackScope,
	issueID string,
	actorID string,
	body io.Reader,
	contentType string,
) (Result, error) {
	if !scope.VotesEnabled {
		return Result{}, failure(403, "error", publicVoteDisabled)
	}
	if err := repository.publicVisibleIssue(ctx, scope, issueID); err != nil {
		return Result{}, err
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return Result{}, err
	}
	if publicFeedbackErr := publicFeedbackObjectRootError(input); publicFeedbackErr != nil {
		return Result{}, publicFeedbackErr
	}
	voteValue, exists := input.Fields["vote"]
	if !exists {
		voteValue = int64(1)
	}
	q := Request{ProjectID: scope.ProjectID, Slug: scope.WorkspaceSlug, IssueID: issueID, ActorID: actorID, Params: map[string]string{"anchor": scope.Anchor}}
	var identifier string
	err = repository.db.QueryRow(ctx, `SELECT id::text FROM issue_votes WHERE issue_id=$1 AND actor_id=$2 AND project_id=$3 AND workspace_id=$4 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`, issueID, actorID, scope.ProjectID, scope.WorkspaceID).Scan(&identifier)
	created := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !created {
		return Result{}, err
	}
	now := timestamp(repository.now())
	if created {
		identifier = uuid.NewString()
		// Django get_or_create persists the model's default before assigning the
		// request value. A database conversion failure therefore leaves the new
		// row behind under the source autocommit contract.
		_, err = repository.db.Exec(ctx, `INSERT INTO issue_votes (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,project_id,workspace_id,issue_id,actor_id,vote) VALUES ($1,$2,$2,NULL,$3,NULL,$4,$5,$6,$3,1)`, identifier, now, actorID, scope.ProjectID, scope.WorkspaceID, issueID)
	}
	if err != nil {
		return Result{}, err
	}
	if collaborationPublicErr := repository.collaborationPublicMember(ctx, q, scope.WorkspaceID); collaborationPublicErr != nil {
		return Result{}, collaborationPublicErr
	}
	vote, ok := publicVoteValue(voteValue)
	if !ok {
		_, err = repository.db.Exec(ctx, `UPDATE issue_votes SET vote=$2,updated_at=$3,updated_by_id=$4 WHERE id=$1`, identifier, voteValue, now, actorID)
		if err == nil {
			return Result{}, fmt.Errorf("invalid persisted public vote value %T", voteValue)
		}
		return Result{}, err
	}
	_, err = repository.db.Exec(ctx, `UPDATE issue_votes SET vote=$2,updated_at=$3,updated_by_id=$4 WHERE id=$1`, identifier, vote, now, actorID)
	if err != nil {
		return Result{}, err
	}
	requested, err := pythoncompat.Dumps(input.Fields, input.Raw)
	if err != nil {
		return Result{}, err
	}
	if publishTaskErr := repository.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{
		"type": "issue_vote.activity.created", "requested_data": requested, "actor_id": actorID,
		"issue_id": issueID, "project_id": scope.ProjectID, "current_instance": nil,
		"epoch": repository.now().Unix(),
	}); publishTaskErr != nil {
		return Result{}, publishTaskErr
	}
	response, err := repository.row(ctx, `
SELECT jsonb_build_object(
  'actor', vote.actor_id,
  'actor_detail', jsonb_build_object('id',actor.id,'first_name',actor.first_name,'last_name',actor.last_name,
    'avatar',actor.avatar,'display_name',actor.display_name,'is_bot',actor.is_bot)||`+djangoapi.FileAssetURLFieldSQL("actor.avatar_asset_id", "actor.avatar", "avatar_url")+`,
  'issue',vote.issue_id,'project',vote.project_id,'vote',vote.vote,'workspace',vote.workspace_id
)
FROM issue_votes vote JOIN users actor ON actor.id=vote.actor_id WHERE vote.id=$1
`, identifier)
	if err != nil {
		return Result{}, err
	}
	// Django serializes the in-memory value assigned by the request. IntegerField
	// coerces it for persistence, but the response retains the original scalar.
	response["vote"] = voteValue
	return Result{Status: 201, Body: response}, nil
}

func publicVoteValue(value any) (int64, bool) {
	switch value := value.(type) {
	case int:
		return int64(value), true
	case int64:
		return value, true
	case bool:
		if value {
			return 1, true
		}
		return 0, true
	case json.Number:
		parsed, err := pythoncompat.DecimalInteger(value.String())
		if err == nil && parsed.IsInt64() {
			return parsed.Int64(), true
		}
		floating, floatErr := pythoncompat.DecimalFloat64(value.String())
		if floatErr != nil || floating > math.MaxInt64 || floating < math.MinInt64 {
			return 0, false
		}
		return int64(floating), true
	case string:
		parsed, err := pythoncompat.DecimalInteger(value)
		if err != nil || !parsed.IsInt64() {
			return 0, false
		}
		return parsed.Int64(), true
	default:
		return 0, false
	}
}

func (repository *Repository) PublicVoteDelete(ctx context.Context, scope PublicFeedbackScope, issueID, actorID string) (Result, error) {
	if !scope.VotesEnabled {
		return Result{}, failure(403, "error", publicVoteDisabled)
	}
	var identifier, vote string
	err := repository.db.QueryRow(ctx, `SELECT id::text,vote::text FROM issue_votes WHERE issue_id=$1::uuid AND actor_id=$2::uuid AND project_id=$3::uuid AND workspace_id=$4::uuid AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`, issueID, actorID, scope.ProjectID, scope.WorkspaceID).Scan(&identifier, &vote)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, requiredObject()
	}
	if err != nil {
		return Result{}, err
	}
	if publicVisibleIssueErr := repository.publicVisibleIssue(ctx, scope, issueID); publicVisibleIssueErr != nil {
		return Result{}, publicVisibleIssueErr
	}
	current, err := orderedDump(map[string]any{"vote": vote, "identifier": identifier}, []string{"vote", "identifier"})
	if err != nil {
		return Result{}, err
	}
	if err := repository.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{
		"type": "issue_vote.activity.deleted", "requested_data": nil, "actor_id": actorID,
		"issue_id": issueID, "project_id": scope.ProjectID, "current_instance": current,
		"epoch": repository.now().Unix(),
	}); err != nil {
		return Result{}, err
	}
	return Result{Status: 204}, repository.softDeleteRow(ctx, "issue_votes", "issuevote", identifier, actorID)
}

func (repository *Repository) publicVisibleIssue(ctx context.Context, scope PublicFeedbackScope, issueID string) error {
	var visible bool
	err := repository.db.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM issues issue
  JOIN projects project ON project.id=issue.project_id
  JOIN workspaces workspace ON workspace.id=issue.workspace_id
  LEFT JOIN states state ON state.id=issue.state_id
  WHERE issue.id=$1::uuid
    AND issue.project_id=$2::uuid
    AND issue.workspace_id=$3::uuid
    AND workspace.slug=$4
    AND issue.deleted_at IS NULL
    AND issue.archived_at IS NULL
    AND issue.is_draft=FALSE
    AND state."group" IS DISTINCT FROM 'triage'
    AND project.deleted_at IS NULL
    AND project.archived_at IS NULL
    AND workspace.deleted_at IS NULL
)
`, issueID, scope.ProjectID, scope.WorkspaceID, scope.WorkspaceSlug).Scan(&visible)
	if err != nil {
		return fmt.Errorf("check public vote target: %w", err)
	}
	if !visible {
		return failure(404, "error", "Issue not found")
	}
	return nil
}
