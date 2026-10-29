package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5"
)

// InvitationCommand records the source action and its authenticated actor. Public
// joins may carry no actor; recipient checks deliberately follow token checks.
type InvitationCommand struct {
	Operation, Slug, ProjectID, ID, UserID, ContentType, BaseHost, Timezone string
	Cache                                                                   InvitationCacheInvalidator
	Body                                                                    io.Reader
	rawBody                                                                 []byte
}

func inviteFailure(status int, message string) error {
	return invitationHTTP(status, map[string]any{"error": message})
}

func inviteMissing() error { return inviteFailure(404, "The required object does not exist.") }

func inviteJSON(ctx context.Context, db database, query string, args ...any) (json.RawMessage, error) {
	var data []byte
	err := db.QueryRow(ctx, query, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inviteMissing()
	}
	return data, err
}

func (r *InvitationsRepository) AuthorizeProjectInvitations(ctx context.Context, userID, slug, projectID string) error {
	var allowed bool
	err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE w.slug=$2 AND pm.project_id=$3 AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL AND (pm.role=20 OR EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$1 AND wm.role=20 AND wm.is_active AND wm.deleted_at IS NULL)))`, userID, slug, projectID).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return inviteFailure(403, "You don't have the required permissions.")
	}
	return nil
}

func (r *InvitationsRepository) Invitation(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	// DRF parses request.data lazily. Resolve targets and self-edit checks at
	// their source boundary, and never parse bodies ignored by an action.
	switch c.Operation {
	case "workspace-list", "workspace-detail", "workspace-public", "project-list", "project-detail", "project-public", "user-workspace-list", "user-project-list":
		return r.invitationRead(ctx, c)
	case "workspace-patch":
		return r.invitationPatch(ctx, c)
	case "workspace-delete", "project-delete":
		return nil, r.invitationDelete(ctx, c)
	case "workspace-join", "project-join":
		return r.invitationJoin(ctx, c)
	case "member-patch", "member-delete", "member-leave":
		return r.mutateWorkspaceMember(ctx, c)
	case "v1-put":
		return r.putAPIWorkspaceInvite(ctx, c)
	}
	fields, err := invitationCommandFields(&c)
	if err != nil {
		return nil, err
	}
	switch c.Operation {
	case "workspace-create", "project-create":
		return r.invitationCreate(ctx, c, fields)
	case "user-workspace-accept":
		return nil, r.acceptWorkspaceInvites(ctx, c, fields)
	case "user-project-join":
		return r.joinUserProjects(ctx, c, fields)
	}
	return nil, fmt.Errorf("unknown invitation operation %q", c.Operation)
}

func (r *InvitationsRepository) trackInvitationEvent(ctx context.Context, userID, slug, workspaceID, event string, properties map[string]any) error {
	properties["user_id"] = userID
	properties["workspace_id"] = workspaceID
	properties["workspace_slug"] = slug
	return r.publisher.Task(ctx, "plane.bgtasks.event_tracking_task.track_event", nil, map[string]any{"user_id": userID, "event_name": event, "slug": slug, "event_properties": properties})
}

// InvitationCacheInvalidator follows the Django decorator key contract.
type InvitationCacheInvalidator interface {
	Invalidate(context.Context, string, string, bool) error
}

type invitationActorContextKey struct{}

func WithInvitationActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, invitationActorContextKey{}, actor)
}

func invitationActor(ctx context.Context) any {
	actor, _ := ctx.Value(invitationActorContextKey{}).(string)
	return nullableInvitationActor(actor)
}

func nullableInvitationActor(actor string) any {
	if actor == "" {
		return nil
	}
	return actor
}

type invitationReplicaContextKey struct{}

func WithInvitationReplica(ctx context.Context) context.Context {
	return context.WithValue(ctx, invitationReplicaContextKey{}, true)
}

// Inherited ModelViewSet lookups raise Http404 through get_object_or_404;
// source actions calling Model.objects.get use the app's generic error instead.
func invitationViewsetMissing(model string) error {
	return invitationHTTP(404, map[string]any{"detail": "No " + model + " matches the given query."})
}
