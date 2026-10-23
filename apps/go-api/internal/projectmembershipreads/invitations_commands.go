package projectmembershipreads

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"

	"github.com/google/uuid"
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

func invitationCommandFields(c *InvitationCommand) (map[string]any, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return nil, err
	}
	if input.Kind != "" {
		return nil, errors.New("invitation data must be an object")
	}
	c.rawBody = input.Raw
	return input.Fields, nil
}

func invitationMemberPatchFields(c InvitationCommand) (map[string]any, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return nil, err
	}
	// The source evaluates "role" in request.data before constructing the
	// serializer. Lists and strings support membership; the remaining scalar
	// types raise before validation, as does .get on a matching list/string.
	switch input.Kind {
	case "":
	case "str":
		var value string
		_ = json.Unmarshal(input.Raw, &value)
		if strings.Contains(value, "role") {
			return nil, errors.New("invitation data has no get method")
		}
	case "list":
		var values []any
		_ = json.Unmarshal(input.Raw, &values)
		for _, value := range values {
			if text, ok := value.(string); ok && text == "role" {
				return nil, errors.New("invitation data has no get method")
			}
		}
	default:
		return nil, errors.New("invitation data does not support membership")
	}
	return invitationSerializerData(input)
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

const inviteWorkspaceSQL = `jsonb_build_object('id',w.id,'name',w.name,'slug',w.slug)||`

func workspaceInviteProjection(public bool) string {
	full := `(to_jsonb(i)-'workspace_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('created_by',i.created_by_id,'updated_by',i.updated_by_id,'workspace',` + inviteWorkspaceSQL + djangoapi.FileAssetURLFieldSQL("w.logo_asset_id", "w.logo", "logo_url") + `,'invite_link','/workspace-invitations/?invitation_id='||i.id::text||'&slug='||w.slug||'&token='||i.token)`
	if public {
		return `(` + full + `)-'token'-'invite_link'-'deleted_at'-'updated_by'`
	}
	return full
}
func projectInviteProjection(public bool) string {
	project := `jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'description',p.description,'cover_image',p.cover_image,'logo_props',p.logo_props)||` + djangoapi.FileAssetURLFieldSQL("p.cover_image_asset_id", "p.cover_image", "cover_image_url")
	full := `(to_jsonb(i)-'workspace_id'-'project_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('created_by',i.created_by_id,'updated_by',i.updated_by_id,'workspace',` + inviteWorkspaceSQL + djangoapi.FileAssetURLFieldSQL("w.logo_asset_id", "w.logo", "logo_url") + `,'project',` + project + `)`
	if public {
		return `(` + full + `)-'token'-'email'-'created_at'-'updated_at'-'deleted_at'-'created_by'-'updated_by'`
	}
	return full
}
func (r *InvitationsRepository) invitationRead(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	project := strings.HasPrefix(c.Operation, "project-") || c.Operation == "user-project-list"
	public := strings.HasSuffix(c.Operation, "-public")
	table, projection, join := "workspace_member_invites", workspaceInviteProjection(public), ""
	if project {
		table = "project_member_invites"
		projection = projectInviteProjection(public)
		join = " JOIN projects p ON p.id=i.project_id"
	}
	where := "i.deleted_at IS NULL"
	args := []any{}
	if strings.HasPrefix(c.Operation, "user-") {
		where += ` AND i.email=(SELECT email FROM users WHERE id=$1)`
		args = append(args, c.UserID)
	} else {
		where += ` AND w.slug=$1`
		args = append(args, c.Slug)
	}
	if project && !strings.HasPrefix(c.Operation, "user-") {
		args = append(args, c.ProjectID)
		where += fmt.Sprintf(" AND i.project_id=$%d", len(args))
	}
	detail := strings.HasSuffix(c.Operation, "-detail") || public
	if detail {
		args = append(args, c.ID)
		where += fmt.Sprintf(" AND i.id=$%d", len(args))
	} else {
		projection = `COALESCE(jsonb_agg(` + projection + ` ORDER BY i.created_at DESC),'[]'::jsonb)`
	}

	var data []byte
	err := r.writer.QueryRow(ctx, `SELECT `+projection+` FROM `+table+` i JOIN workspaces w ON w.id=i.workspace_id`+join+` WHERE `+where, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		if c.Operation == "workspace-detail" {
			return nil, invitationViewsetMissing("WorkspaceMemberInvite")
		}
		if c.Operation == "project-detail" {
			return nil, invitationViewsetMissing("ProjectMemberInvite")
		}
		return nil, inviteMissing()
	}
	return json.RawMessage(data), err
}

func invitationRole(value any) (int, bool) {
	// ChoiceField uses Python str(value), so a JSON floating token is not an
	// integer choice even when its mathematical value is integral.
	switch v := value.(type) {
	case json.Number:
		return invitationRole(string(v))
	case int:
		if v == 5 || v == 15 || v == 20 {
			return v, true
		}
	case string:
		if v == "5" || v == "15" || v == "20" {
			n, _ := strconv.Atoi(v)
			return n, true
		}
	}
	return 0, false
}
func invitationInt(value any) (int, error) {
	switch v := value.(type) {
	case json.Number:
		f, e := strconv.ParseFloat(string(v), 64)
		return int(f), e
	case float64:
		return int(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.Atoi(strings.TrimSpace(v))
	default:
		return 0, errors.New("invalid integer")
	}
}
func invitationToken(secret string, item map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"email": item, "timestamp": float64(time.Now().UnixNano()) / 1e9})
	message := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return message + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (r *InvitationsRepository) invitationCreate(ctx context.Context, c InvitationCommand, fields map[string]any) (json.RawMessage, error) {
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
	var actorRole int
	// Workspace creation preserves its eager role conversion and membership check.
	if !project {
		if err := r.writer.QueryRow(ctx, `SELECT w.id::text,u.email,wm.role FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=$2 JOIN users u ON u.id=wm.member_id WHERE w.slug=$1 AND wm.deleted_at IS NULL AND wm.is_active`, c.Slug, c.UserID).Scan(&workspaceID, &actorEmail, &actorRole); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, inviteMissing()
			}
			return nil, err
		}
	}
	if !project {
		for _, raw := range emails {
			item, ok := raw.(map[string]any)
			if !ok {
				return nil, errors.New("invalid invitation")
			}
			role := any(json.Number("5"))
			if v, ok := item["role"]; ok {
				role = v
			}
			n, err := invitationInt(role)
			if err != nil {
				return nil, err
			}
			if n > actorRole {
				return nil, inviteFailure(400, "You cannot invite a user with higher role")
			}
		}
		if err := r.writer.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, c.Slug).Scan(&workspaceID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, inviteMissing()
			}
			return nil, err
		}
		rawEmails := []string{}
		for _, raw := range emails {
			item := raw.(map[string]any)
			s, _ := item["email"].(string)
			rawEmails = append(rawEmails, s)
		}
		existing, err := inviteJSON(ctx, r.writer, `SELECT COALESCE(jsonb_agg(`+workspaceMemberProjection()+` ORDER BY m.created_at DESC),'[]'::jsonb) FROM workspace_members m JOIN users u ON u.id=m.member_id WHERE m.workspace_id=$1 AND u.email=ANY($2) AND m.is_active AND m.deleted_at IS NULL`, workspaceID, rawEmails)
		if err != nil {
			return nil, err
		}
		if string(existing) != "[]" {
			var users any
			_ = json.Unmarshal(existing, &users)
			return nil, invitationHTTP(400, map[string]any{"error": "Some users are already member of workspace", "workspace_users": users})
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
	// Commit or the returned error closes the transaction; rollback is best-effort cleanup.
	defer func() { _ = tx.Rollback(ctx) }()
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

func (r *InvitationsRepository) trackInvitationEvent(ctx context.Context, userID, slug, workspaceID, event string, properties map[string]any) error {
	properties["user_id"] = userID
	properties["workspace_id"] = workspaceID
	properties["workspace_slug"] = slug
	return r.publisher.Task(ctx, "plane.bgtasks.event_tracking_task.track_event", nil, map[string]any{"user_id": userID, "event_name": event, "slug": slug, "event_properties": properties})
}
func (r *InvitationsRepository) invitationDelete(ctx context.Context, c InvitationCommand) error {
	table, model := "workspace_member_invites", "workspacememberinvite"
	if c.Operation == "project-delete" {
		table = "project_member_invites"
		model = "projectmemberinvite"
	}
	var id string
	query := `SELECT i.id::text FROM ` + table + ` i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND w.slug=$2 AND i.deleted_at IS NULL`
	args := []any{c.ID, c.Slug}
	if c.ProjectID != "" {
		query += ` AND i.project_id=$3`
		args = append(args, c.ProjectID)
	}
	err := r.writer.QueryRow(ctx, query, args...).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		if c.Operation == "project-delete" {
			return invitationViewsetMissing("ProjectMemberInvite")
		}
		return inviteMissing()
	}
	if err != nil {
		return err
	}
	if _, err = r.writer.Exec(ctx, `UPDATE `+table+` SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, id, nullableInvitationActor(c.UserID)); err != nil {
		return err
	}
	return r.publisher.Task(ctx, "plane.bgtasks.deletion_task.soft_delete_related_objects", []any{"db", model, map[string]any{"__type__": "uuid", "__value__": map[string]string{"hex": strings.ReplaceAll(id, "-", "")}}}, map[string]any{"using": nil})
}

// WorkspaceDefaultProps is the default Plane view configuration assigned to a new workspace member.
const WorkspaceDefaultProps = `{"filters":` + memberFilters + `,"display_filters":` + memberDisplayFilters + `,"display_properties":` + memberDisplayProperties + `}`

// WorkspaceIssueProps is the default issue subscription configuration for a new workspace member.
const WorkspaceIssueProps = `{"subscribed":true,"assigned":true,"created":true,"all_issues":true}`

func insertWorkspaceMembership(ctx context.Context, db database, workspace, user string, role int, creator any) error {
	_, err := db.Exec(ctx, `INSERT INTO workspace_members(id,workspace_id,member_id,role,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$5,$6,true,'{}','{}','{}',clock_timestamp(),clock_timestamp(),$7) ON CONFLICT DO NOTHING`, uuid.NewString(), workspace, user, role, WorkspaceDefaultProps, WorkspaceIssueProps, creator)
	return err
}
func insertProjectMembership(ctx context.Context, db database, project, workspace, user string, role int, creator any, property bool) error {
	if property {
		lr := &LifecycleRepository{Repository: &Repository{database: db}}
		if err := lr.insertLifecycleProperty(ctx, LifecycleCommand{ProjectID: project, UserID: user}, user, false); err != nil {
			return err
		}
	}
	_, err := db.Exec(ctx, `INSERT INTO project_members(id,project_id,workspace_id,member_id,role,view_props,default_props,preferences,sort_order,is_active,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7,65535,true,clock_timestamp(),clock_timestamp(),$8) ON CONFLICT DO NOTHING`, uuid.NewString(), project, workspace, user, role, memberDefaultProps, memberPreferences, creator)
	return err
}
func invitationTruth(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case json.Number:
		f, _ := strconv.ParseFloat(string(v), 64)
		return f != 0
	case float64:
		return v != 0
	case []any:
		return len(v) != 0
	case map[string]any:
		return len(v) != 0
	default:
		return true
	}
}
func (r *InvitationsRepository) invitationJoin(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	project := c.Operation == "project-join"
	table, kind := "workspace_member_invites", "workspace"
	if project {
		table = "project_member_invites"
		kind = "project"
	}
	var token, email, workspaceID string
	var role int
	var responded *time.Time
	args := []any{c.Slug, c.ID}
	query := `SELECT i.token,i.email,i.workspace_id::text,i.role,i.responded_at FROM ` + table + ` i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.id=$2 AND i.deleted_at IS NULL`
	if project {
		query += ` AND i.project_id=$3`
		args = append(args, c.ProjectID)
	}
	err := r.writer.QueryRow(ctx, query, args...).Scan(&token, &email, &workspaceID, &role, &responded)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inviteMissing()
	}
	if err != nil {
		return nil, err
	}
	fields, err := invitationCommandFields(&c)
	if err != nil {
		return nil, err
	}
	submitted, _ := fields["token"].(string)
	if submitted == "" || submitted != token {
		return nil, inviteFailure(403, "You do not have permission to join the "+kind)
	}
	if c.UserID == "" {
		return nil, inviteFailure(401, "Authentication required to accept "+kind+" invitation")
	}
	var actorEmail string
	if err = r.writer.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, c.UserID).Scan(&actorEmail); err != nil {
		return nil, err
	}
	if !strings.EqualFold(email, actorEmail) {
		return nil, inviteFailure(403, "You do not have permission to accept this invitation")
	}
	if responded != nil {
		return nil, inviteFailure(400, "You have already responded to the invitation request")
	}
	accepted, storedAccepted := false, false
	if raw, present := fields["accepted"]; present {
		if project {
			var valid bool
			accepted, valid = raw.(bool)
			if !valid {
				return nil, inviteFailure(400, "`accepted` must be a boolean")
			}
			storedAccepted = accepted
		} else {
			accepted = invitationTruth(raw)
			var valid bool
			storedAccepted, valid = invitationModelBoolean(raw)
			if !valid {
				return nil, inviteFailure(400, "Please provide valid detail")
			}
		}
	}
	if _, err = r.writer.Exec(ctx, `UPDATE `+table+` SET accepted=$2,responded_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$3 WHERE id=$1`, c.ID, storedAccepted, c.UserID); err != nil {
		return nil, err
	}
	if !accepted {
		return json.RawMessage(`{"message":"` + strings.Title(kind) + ` Invitation was not accepted"}`), nil
	}
	userID := c.UserID
	if !project {
		err = r.writer.QueryRow(ctx, `SELECT id::text FROM users WHERE email=$1 ORDER BY created_at ASC LIMIT 1`, email).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return json.RawMessage(`{"message":"Workspace Invitation Accepted"}`), nil
		}
		if err != nil {
			return nil, err
		}
	}
	var membership string
	err = r.writer.QueryRow(ctx, `SELECT id::text FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT 1`, workspaceID, userID).Scan(&membership)
	if errors.Is(err, pgx.ErrNoRows) {
		workspaceRole := role
		if project && role >= 15 {
			workspaceRole = 15
		}
		err = insertWorkspaceMembership(ctx, r.writer, workspaceID, userID, workspaceRole, c.UserID)
	} else if err == nil {
		if project {
			_, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=true,updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, membership, c.UserID)
		} else {
			_, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=true,role=$2,updated_at=clock_timestamp(),updated_by_id=$3 WHERE id=$1`, membership, role, c.UserID)
		}
	}
	if err != nil {
		return nil, err
	}
	if project {
		var target string
		err = r.writer.QueryRow(ctx, `SELECT id::text FROM project_members WHERE workspace_id=$1 AND project_id=$2 AND member_id=$3 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT 1`, workspaceID, c.ProjectID, c.UserID).Scan(&target)
		if errors.Is(err, pgx.ErrNoRows) {
			err = insertProjectMembership(ctx, r.writer, c.ProjectID, workspaceID, c.UserID, role, c.UserID, true)
		} else if err == nil {
			_, err = r.writer.Exec(ctx, `UPDATE project_members SET is_active=true,updated_at=clock_timestamp(),updated_by_id=$2,workspace_id=(SELECT workspace_id FROM projects WHERE id=project_members.project_id) WHERE id=$1`, target, c.UserID)
		}
		if err != nil {
			return nil, err
		}
		return json.RawMessage(`{"message":"Project Invitation Accepted"}`), nil
	}
	if _, err = r.writer.Exec(ctx, `UPDATE users SET email=lower(btrim(email)),display_name=CASE WHEN display_name='' THEN split_part(lower(btrim(email)),'@',1) ELSE display_name END,is_staff=(is_staff OR is_superuser),token=CASE WHEN token_updated_at IS NOT NULL THEN replace(gen_random_uuid()::text,'-','')||replace(gen_random_uuid()::text,'-','') ELSE token END,token_updated_at=CASE WHEN token_updated_at IS NOT NULL THEN clock_timestamp() ELSE NULL END,updated_at=clock_timestamp() WHERE id=$1`, userID); err != nil {
		return nil, err
	}
	if err = r.trackInvitationEvent(ctx, userID, c.Slug, workspaceID, "user_joined_workspace", map[string]any{"role": role, "joined_at": time.Now().UTC().Format("2006-01-02 15:04:05.999999+00:00")}); err != nil {
		return nil, err
	}
	return json.RawMessage(`{"message":"Workspace Invitation Accepted"}`), r.invitationDelete(ctx, InvitationCommand{Operation: "workspace-delete", Slug: c.Slug, ID: c.ID, UserID: c.UserID})
}

func invitationIDs(value any) ([]string, error) {
	raw, ok := value.([]any)
	if value == nil {
		return nil, errors.New("invitation ids must be iterable")
	}
	if !ok {
		return nil, errors.New("invalid id list")
	}
	ids := make([]string, 0, len(raw))
	for _, id := range raw {
		u, err := djangoinput.UUID(id)
		if err != nil {
			return nil, inviteFailure(400, "Invalid input.")
		}
		ids = append(ids, u.String())
	}
	return ids, nil
}
func (r *InvitationsRepository) acceptWorkspaceInvites(ctx context.Context, c InvitationCommand, fields map[string]any) error {
	rawIDs, exists := fields["invitations"]
	if !exists {
		rawIDs = []any{}
	}
	ids, err := invitationIDs(rawIDs)
	if err != nil {
		return err
	}
	rows, err := r.writer.Query(ctx, `SELECT i.id::text,i.workspace_id::text,w.slug,i.role FROM workspace_member_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id::text=ANY($1) AND i.email=(SELECT email FROM users WHERE id=$2) AND i.deleted_at IS NULL ORDER BY i.created_at DESC`, ids, c.UserID)
	if err != nil {
		return err
	}
	type invitation struct {
		id, workspace, slug string
		role                int
	}
	invites := []invitation{}
	for rows.Next() {
		var i invitation
		if err = rows.Scan(&i.id, &i.workspace, &i.slug, &i.role); err != nil {
			rows.Close()
			return err
		}
		invites = append(invites, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, i := range invites {
		if c.Cache != nil {
			if err = c.Cache.Invalidate(ctx, "/api/workspaces/"+i.slug+"/members/", "", true); err != nil {
				return err
			}
		}
		if _, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=true,role=$3 WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL`, i.workspace, c.UserID, i.role); err != nil {
			return err
		}
		if err = r.trackInvitationEvent(ctx, c.UserID, i.slug, i.workspace, "user_joined_workspace", map[string]any{"role": i.role, "joined_at": time.Now().UTC().Format("2006-01-02 15:04:05.999999+00:00")}); err != nil {
			return err
		}
	}
	for _, i := range invites {
		if err = insertWorkspaceMembership(ctx, r.writer, i.workspace, c.UserID, i.role, c.UserID); err != nil {
			return err
		}
	}
	for _, i := range invites {
		if _, err = r.writer.Exec(ctx, `UPDATE workspace_member_invites SET deleted_at=clock_timestamp() WHERE id=$1`, i.id); err != nil {
			return err
		}
	}
	return nil
}
func (r *InvitationsRepository) joinUserProjects(ctx context.Context, c InvitationCommand, fields map[string]any) (json.RawMessage, error) {
	rawIDs, exists := fields["project_ids"]
	if !exists {
		rawIDs = []any{}
	}
	ids, err := invitationIDs(rawIDs)
	if err != nil {
		return nil, err
	}
	var role int
	var workspace string
	if err = r.writer.QueryRow(ctx, `SELECT m.role,m.workspace_id::text FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL`, c.Slug, c.UserID).Scan(&role, &workspace); err != nil {
		return nil, err
	}
	rows, err := r.writer.Query(ctx, `SELECT p.id::text,p.network FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id::text=ANY($1) AND w.slug=$2 AND p.deleted_at IS NULL ORDER BY p.created_at DESC`, ids, c.Slug)
	if err != nil {
		return nil, err
	}
	projects := []string{}
	for rows.Next() {
		var id string
		var network int
		if err = rows.Scan(&id, &network); err != nil {
			rows.Close()
			return nil, err
		}
		if network == 0 && role != 20 {
			rows.Close()
			return nil, inviteFailure(403, "Only workspace admins can join private project")
		}
		projects = append(projects, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, id := range projects {
		if _, err = r.writer.Exec(ctx, `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2 AND deleted_at IS NULL`, id, c.UserID); err != nil {
			return nil, err
		}
	}
	for _, id := range projects {
		if err = insertProjectMembership(ctx, r.writer, id, workspace, c.UserID, role, c.UserID, false); err != nil {
			return nil, err
		}
	}
	for _, id := range projects {
		_, err = r.writer.Exec(ctx, `INSERT INTO project_user_properties(id,project_id,workspace_id,user_id,filters,display_filters,display_properties,rich_filters,preferences,sort_order,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$6,$7,'{}',$8,65535,clock_timestamp(),clock_timestamp(),$4) ON CONFLICT DO NOTHING`, uuid.NewString(), id, workspace, c.UserID, memberFilters, memberDisplayFilters, memberDisplayProperties, memberPreferences)
		if err != nil {
			return nil, err
		}
	}
	return json.RawMessage(`{"message":"Projects joined successfully"}`), nil
}

func workspaceMemberProjection() string {
	return `(to_jsonb(m)-'workspace_id'-'member_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('workspace',m.workspace_id,'created_by',m.created_by_id,'updated_by',m.updated_by_id,'member',jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'is_bot',u.is_bot,'display_name',u.display_name)||` + djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + `)`
}
func (r *InvitationsRepository) mutateWorkspaceMember(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	leave := c.Operation == "member-leave"
	var id, userID, workspace string
	var role int
	query := `SELECT m.id::text,m.member_id::text,m.workspace_id::text,m.role FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id JOIN users u ON u.id=m.member_id WHERE w.slug=$1 AND m.deleted_at IS NULL AND m.is_active`
	args := []any{c.Slug}
	if leave {
		query += ` AND m.member_id=$2`
		args = append(args, c.UserID)
	} else {
		query += ` AND m.id=$2 AND NOT u.is_bot`
		args = append(args, c.ID)
	}
	err := r.writer.QueryRow(ctx, query, args...).Scan(&id, &userID, &workspace, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inviteMissing()
	}
	if err != nil {
		return nil, err
	}
	if c.Operation == "member-patch" {
		if userID == c.UserID {
			return nil, inviteFailure(400, "You cannot update your own role")
		}
		fields, err := invitationMemberPatchFields(c)
		if err != nil {
			return nil, err
		}
		// Keep source int coercion before serializer validation, but make validated
		// role and every cascade one transaction (ACCESS-001).
		cascade := false
		if raw, ok := fields["role"]; ok {
			n, e := invitationInt(raw)
			if e != nil {
				return nil, e
			}
			cascade = n == 5
		}
		updates, args, fieldError := workspaceMemberUpdates(fields, id, c.UserID, c.Timezone)
		err = invitationValidationErrors(fieldError, r.invitationRelationErrors(ctx, fields, workspaceMembershipFields))
		if err != nil {
			return nil, err
		}
		tx, err := r.writer.Begin(ctx)
		if err != nil {
			return nil, err
		}
		// Commit or the returned error closes the transaction; rollback is best-effort cleanup.
		defer func() { _ = tx.Rollback(ctx) }()
		if cascade {
			if _, err = tx.Exec(ctx, `UPDATE project_members SET role=5 WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL`, workspace, userID); err != nil {
				return nil, err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE workspace_members SET `+strings.Join(updates, ",")+` WHERE id=$1`, args...); err != nil {
			return nil, err
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		return inviteJSON(ctx, r.writer, `SELECT `+workspaceMemberProjection()+` FROM workspace_members m JOIN users u ON u.id=m.member_id WHERE m.id=$1`, id)
	}
	if !leave {
		if userID == c.UserID {
			return nil, inviteFailure(400, "You cannot remove yourself from the workspace. Please use leave workspace")
		}
		var actorRole int
		if err = r.writer.QueryRow(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL`, workspace, c.UserID).Scan(&actorRole); err != nil {
			return nil, err
		}
		if actorRole < role {
			return nil, inviteFailure(400, "You cannot remove a user having role higher than you")
		}
	} else if role == 20 {
		var admins int
		if err = r.writer.QueryRow(ctx, `SELECT count(*) FROM workspace_members WHERE workspace_id=$1 AND role=20 AND is_active AND deleted_at IS NULL`, workspace).Scan(&admins); err != nil {
			return nil, err
		}
		if admins <= 1 {
			return nil, inviteFailure(400, "You cannot leave the workspace as you are the only admin of the workspace you will have to either delete the workspace or promote another user to admin.")
		}
	}
	var sole bool
	// Reverse relation counts include inactive and deleted member rows; the
	// Project manager alone excludes deleted projects, as the source annotation.
	err = r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id WHERE p.workspace_id=$1 AND p.deleted_at IS NULL GROUP BY p.id HAVING count(*)=1 AND count(*) FILTER(WHERE pm.member_id=$2 AND pm.role=20)=1)`, workspace, userID).Scan(&sole)
	if err != nil {
		return nil, err
	}
	if sole {
		if leave {
			return nil, inviteFailure(400, "You are a part of some projects where you are the only admin, you should either leave the project or promote another user to admin.")
		}
		return nil, inviteFailure(400, "User is a part of some projects where they are the only admin, they should either leave that project or promote another user to admin.")
	}
	if _, err = r.writer.Exec(ctx, `UPDATE project_members SET is_active=false,updated_at=clock_timestamp() WHERE workspace_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL`, workspace, userID); err != nil {
		return nil, err
	}
	_, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=false,updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, id, c.UserID)
	return nil, err
}

var workspaceMembershipFields = []string{"role", "company_role", "is_active", "view_props", "default_props", "issue_props", "getting_started_checklist", "tips", "explored_features", "workspace", "created_by", "updated_by", "deleted_at"}
var workspaceInvitationFields = []string{"role", "accepted", "created_by", "updated_by", "deleted_at"}

func workspaceMemberUpdates(fields map[string]any, id, actor, zone string) ([]string, []any, error) {
	return invitationFieldUpdates(fields, id, actor, zone, workspaceMembershipFields)
}
func invitationChoiceMessages(value any) []string {
	if value == nil {
		return []string{"This field may not be null."}
	}
	label := fmt.Sprint(value)
	if flag, ok := value.(bool); ok {
		label = "False"
		if flag {
			label = "True"
		}
	}
	return []string{fmt.Sprintf("\"%s\" is not a valid choice.", label)}
}
func invitationFieldUpdates(fields map[string]any, id, actor, zone string, keys []string) ([]string, []any, error) {
	updates := []string{"updated_at=clock_timestamp()", "updated_by_id=$2"}
	args := []any{id, actor}
	failures := map[string]any{}
	for _, key := range keys {
		value, present := fields[key]
		if !present {
			continue
		}
		column := key
		var messages []string
		nullable := key == "company_role" || key == "created_by" || key == "updated_by" || key == "deleted_at"
		if value == nil && !nullable {
			failures[key] = []string{"This field may not be null."}
			continue
		}
		switch key {
		case "role":
			n, valid := invitationRole(value)
			if !valid {
				messages = invitationChoiceMessages(value)
			} else {
				value = n
			}
		case "is_active", "accepted":
			n, valid := invitationBoolean(value)
			if !valid {
				messages = []string{"Must be a valid boolean."}
			} else {
				value = n
			}
		case "workspace", "created_by", "updated_by":
			column += "_id"
			if value != nil {
				parsed, err := djangoinput.UUID(value)
				if err != nil {
					messages = []string{fmt.Sprintf("“%v” is not a valid UUID.", value)}
				} else {
					value = parsed.String()
				}
			}
		case "deleted_at":
			if value != nil {
				text, valid := value.(string)
				location := time.UTC
				if zone != "" {
					if loaded, err := time.LoadLocation(zone); err == nil {
						location = loaded
					}
				}
				parsed, err := pythoncompat.DjangoDateTime(text, location)
				if !valid || err != nil {
					messages = []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
				} else {
					value = parsed
				}
			}
		case "company_role":
			if value != nil {
				value, messages = djangoinput.TextField(value, true, 0, nil)
			}
		default:
			data, err := json.Marshal(value)
			if err != nil {
				return nil, nil, err
			}
			value = string(data)
		}
		if len(messages) > 0 {
			failures[key] = messages
			continue
		}
		// AuditModel.save validates submitted updated_by through the serializer,
		// then unconditionally replaces it with the authenticated request actor.
		if key == "updated_by" {
			continue
		}
		args = append(args, value)
		updates = append(updates, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if len(failures) > 0 {
		return nil, nil, invitationHTTP(400, failures)
	}
	return updates, args, nil
}
func (r *InvitationsRepository) invitationRelationErrors(ctx context.Context, fields map[string]any, keys []string) error {
	failures := map[string]any{}
	for _, key := range keys {
		if key != "workspace" && key != "created_by" && key != "updated_by" {
			continue
		}
		raw, present := fields[key]
		if !present || raw == nil {
			continue
		}
		id, err := djangoinput.UUID(raw)
		if err != nil {
			continue
		}
		table, predicate := "users", ""
		if key == "workspace" {
			table = "workspaces"
			predicate = " AND deleted_at IS NULL"
		}
		var exists bool
		if err = r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id=$1`+predicate+`)`, id.String()).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			failures[key] = []string{fmt.Sprintf("Invalid pk \"%v\" - object does not exist.", raw)}
		}
	}
	if len(failures) > 0 {
		return invitationHTTP(400, failures)
	}
	return nil
}
func invitationValidationErrors(left, right error) error {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	var first, second *djangoapi.HTTPError
	if errors.As(left, &first) && errors.As(right, &second) && first.Status == 400 && second.Status == 400 {
		body := map[string]any{}
		for key, value := range first.Body {
			body[key] = value
		}
		for key, value := range second.Body {
			body[key] = value
		}
		return invitationHTTP(400, body)
	}
	return right
}

func invitationBoolean(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes", "t", "on":
			return true, true
		case "false", "0", "no", "f", "off":
			return false, true
		}
	case json.Number:
		number, err := strconv.ParseFloat(string(v), 64)
		if err == nil && (number == 0 || number == 1) {
			return number == 1, true
		}
	case float64:
		if v == 1 {
			return true, true
		}
		if v == 0 {
			return false, true
		}
	}
	return false, false
}
func (r *InvitationsRepository) invitationPatch(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	if _, err := r.invitationRead(ctx, InvitationCommand{Operation: "workspace-detail", Slug: c.Slug, ID: c.ID}); err != nil {
		return nil, err
	}
	fields, err := decodeInvitationSerializerPayload(c.Body, c.ContentType)
	if err != nil {
		return nil, err
	}
	updates, args, fieldError := invitationFieldUpdates(fields, c.ID, c.UserID, c.Timezone, workspaceInvitationFields)
	if err := invitationValidationErrors(fieldError, r.invitationRelationErrors(ctx, fields, workspaceInvitationFields)); err != nil {
		return nil, err
	}
	if _, err := r.writer.Exec(ctx, `UPDATE workspace_member_invites SET `+strings.Join(updates, ",")+` WHERE id=$1`, args...); err != nil {
		return nil, err
	}
	return inviteJSON(ctx, r.writer, `SELECT `+workspaceInviteProjection(false)+` FROM workspace_member_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1`, c.ID)
}

func (r *InvitationsRepository) putAPIWorkspaceInvite(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	id, err := uuid.Parse(c.ID)
	if err != nil {
		return nil, inviteFailure(400, "Please provide valid detail")
	}
	current, err := r.WorkspaceInvite(ctx, c.Slug, id)
	if err != nil {
		return nil, err
	}
	fields, err := invitationCommandFields(&c)
	if err != nil {
		return nil, err
	}
	return r.updateAPIWorkspaceInvite(ctx, c.Slug, id, current, fields, false)
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

// Model BooleanField coercion differs from DRF BooleanField and from Python
// truthiness. Django leaves the instance attribute uncoerced after saving.
func invitationModelBoolean(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case json.Number:
		if v == "1" || v == "1.0" {
			return true, true
		}
		if v == "0" || v == "0.0" {
			return false, true
		}
	case string:
		switch v {
		case "t", "True", "1":
			return true, true
		case "f", "False", "0":
			return false, true
		}
	}
	return false, false
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
