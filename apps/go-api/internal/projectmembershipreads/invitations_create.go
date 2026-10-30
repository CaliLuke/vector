package projectmembershipreads

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func invitationToken(secret string, item map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"email": item, "timestamp": float64(time.Now().UnixNano()) / 1e9})
	message := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return message + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (r *InvitationsRepository) invitationCreate(ctx context.Context, c InvitationCommand, fields map[string]any) (_ json.RawMessage, resultErr error) {
	project := c.Operation == "project-create"
	rawEmailsValue := fields["emails"]
	emails, ok := rawEmailsValue.([]any)
	if !ok || len(emails) == 0 {
		if project || !invitationTruth(rawEmailsValue) {
			return nil, inviteFailure(400, "Emails are required")
		}
		return nil, errors.New("invitation emails must be iterable objects")
	}
	type entry struct {
		email   string
		role    int
		token   string
		rawRole any
	}
	entries := make([]entry, 0, len(emails))
	var workspaceID, actorEmail string
	if !project {
		var err error
		workspaceID, actorEmail, err = r.validateWorkspaceInvitationBatch(ctx, c.UserID, c.Slug, emails)
		if err != nil {
			return nil, err
		}
	}
	for index, raw := range emails {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, inviteFailure(400, "Invalid invitation")
		}
		email, isString := item["email"].(string)
		if project && !isString {
			return nil, inviteFailure(400, "Invalid email")
		}
		value := any(json.Number("5"))
		if v, ok := item["role"]; ok {
			value = v
		}
		role, valid := invitationRole(value)
		if project && !valid {
			return nil, inviteFailure(400, "Invalid role")
		}
		if !project {
			var e error
			role, e = invitationInt(value)
			if e != nil {
				return nil, e
			}
		}
		normalized := normalizeInvitationEmail(email)
		checked := email
		if project {
			checked = normalized
		}
		if !validInvitationEmail(checked) {
			label := invitationPythonEntry(item, invitationRawEntry(c.rawBody, index))
			if project {
				label = email
			}
			return nil, inviteFailure(400, "Invalid email - "+label+" provided a valid email address is required to send the invite")
		}
		entries = append(entries, entry{normalized, role, invitationToken(r.secretKey, item), value})
	}
	if project {
		if err := r.writer.QueryRow(ctx, `SELECT w.id::text,u.email FROM workspaces w CROSS JOIN users u WHERE w.slug=$1 AND w.deleted_at IS NULL AND u.id=$2`, c.Slug, c.UserID).Scan(&workspaceID, &actorEmail); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, inviteMissing()
			}
			return nil, err
		}
		for _, e := range entries {
			var role int
			err := r.writer.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN users u ON u.id=wm.member_id WHERE wm.workspace_id=$1 AND lower(u.email)=lower($2) AND wm.is_active AND wm.deleted_at IS NULL ORDER BY wm.created_at ASC LIMIT 1`, workspaceID, e.email).Scan(&role)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			if err == nil && (role == 5 || role == 20) && role != e.role {
				return nil, inviteFailure(400, "You cannot invite a user with different role than workspace role")
			}
		}
	}
	// bulk_create is one transaction; no task can escape a rejected batch.
	tx, err := r.writer.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// Cleanup keeps its own deadline and retains failures alongside the operation error.
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	for _, e := range entries {
		if project {
			_, err = tx.Exec(ctx, `INSERT INTO project_member_invites(id,project_id,workspace_id,email,role,token,accepted,created_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,false,$7,clock_timestamp(),clock_timestamp()) ON CONFLICT DO NOTHING`, uuid.NewString(), c.ProjectID, workspaceID, e.email, e.role, e.token, c.UserID)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO workspace_member_invites(id,workspace_id,email,role,token,accepted,created_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,false,$6,clock_timestamp(),clock_timestamp()) ON CONFLICT DO NOTHING`, uuid.NewString(), workspaceID, e.email, e.role, e.token, c.UserID)
		}
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	host := r.baseHost
	for _, e := range entries {
		task, id := "plane.bgtasks.workspace_invitation_task.workspace_invitation", workspaceID
		if project {
			task = "plane.bgtasks.project_invitation_task.project_invitation"
			id = c.ProjectID
		}
		if err = r.publisher.Task(ctx, task, []any{e.email, id, e.token, host, actorEmail}, nil); err != nil {
			return nil, err
		}
		if !project {
			if err = r.trackInvitationEvent(ctx, c.UserID, c.Slug, workspaceID, "user_invited_to_workspace", map[string]any{"invitee_role": e.rawRole, "invited_at": time.Now().UTC().Format("2006-01-02 15:04:05.999999+00:00"), "invitee_email": e.email}); err != nil {
				return nil, err
			}
		}
	}
	if project {
		return json.RawMessage(`{"message":"Email sent successfully"}`), nil
	}
	return json.RawMessage(`{"message":"Emails sent successfully"}`), nil
}

// validateWorkspaceInvitationBatch preserves workspace invitation eligibility
// checks before shared email validation or any durable effects.
func (r *InvitationsRepository) validateWorkspaceInvitationBatch(ctx context.Context, userID, slug string, emails []any) (string, string, error) {
	var workspaceID, actorEmail string
	var actorRole int
	if err := r.writer.QueryRow(ctx, `SELECT w.id::text,u.email,wm.role FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=$2 JOIN users u ON u.id=wm.member_id WHERE w.slug=$1 AND wm.deleted_at IS NULL AND wm.is_active`, slug, userID).Scan(&workspaceID, &actorEmail, &actorRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", inviteMissing()
		}
		return "", "", err
	}
	for _, raw := range emails {
		item, ok := raw.(map[string]any)
		if !ok {
			return "", "", errors.New("invalid invitation")
		}
		role := any(json.Number("5"))
		if v, ok := item["role"]; ok {
			role = v
		}
		n, err := invitationInt(role)
		if err != nil {
			return "", "", err
		}
		if n > actorRole {
			return "", "", inviteFailure(400, "You cannot invite a user with higher role")
		}
	}
	if err := r.writer.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, slug).Scan(&workspaceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", inviteMissing()
		}
		return "", "", err
	}
	rawEmails := []string{}
	for _, raw := range emails {
		item := raw.(map[string]any)
		s, _ := item["email"].(string)
		rawEmails = append(rawEmails, s)
	}
	existing, err := inviteJSON(ctx, r.writer, `SELECT COALESCE(jsonb_agg(`+workspaceMemberProjection()+` ORDER BY m.created_at DESC),'[]'::jsonb) FROM workspace_members m JOIN users u ON u.id=m.member_id WHERE m.workspace_id=$1 AND u.email=ANY($2) AND m.is_active AND m.deleted_at IS NULL`, workspaceID, rawEmails)
	if err != nil {
		return "", "", err
	}
	if string(existing) != "[]" {
		var users any
		_ = json.Unmarshal(existing, &users)
		return "", "", invitationHTTP(400, map[string]any{"error": "Some users are already member of workspace", "workspace_users": users})
	}
	return workspaceID, actorEmail, nil
}
