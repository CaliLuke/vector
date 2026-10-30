package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/loom/security"
)

const projectMembershipReadsInternalErrorDetail = "Something went wrong please try again later"

type projectMembershipReadsRepository interface {
	AuthorizeWorkspace(context.Context, uuid.UUID, string) error
	Membership(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Roles(context.Context, uuid.UUID, string) (json.RawMessage, error)
	WorkspaceMembers(context.Context, uuid.UUID, string, []string) (json.RawMessage, error)
	WorkspaceMember(context.Context, uuid.UUID, string, uuid.UUID, []string) (json.RawMessage, error)
	WorkspaceProjectMembers(context.Context, uuid.UUID, string) (json.RawMessage, error)
	ProjectMembers(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	ProjectMember(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Preference(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
}

type projectMembershipReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type projectMembershipReadsService struct {
	repository projectMembershipReadsRepository
	sessions   projectMembershipReadsSessions
}

type projectMembershipReadsPrincipal struct{}

type ProjectMembershipReadsResponseCapture struct {
	Data json.RawMessage
}

// ProjectMembershipReadsMetadataAuthorizer performs the lightweight checks
// needed by Django-compatible OPTIONS responses without loading endpoint data.
type ProjectMembershipReadsMetadataAuthorizer interface {
	AuthorizeMetadata(context.Context, string, string, bool) error
}

type projectMembershipReadsResponseCaptureKey struct{}

func WithProjectMembershipReadsResponseCapture(
	ctx context.Context,
	capture *ProjectMembershipReadsResponseCapture,
) context.Context {
	return context.WithValue(ctx, projectMembershipReadsResponseCaptureKey{}, capture)
}

func NewProjectMembershipReads(
	repository projectMembershipReadsRepository,
	sessions projectMembershipReadsSessions,
) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("project membership reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &projectMembershipReadsService{repository: repository, sessions: sessions}, nil
}

func (service *projectMembershipReadsService) APIKeyAuth(
	ctx context.Context,
	key string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.ProjectMembershipReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.ProjectMembershipReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, projectMembershipReadsPrincipal{}, userID), nil
}

func (service *projectMembershipReadsService) AuthorizeMetadata(
	ctx context.Context,
	sessionID string,
	slug string,
	requireWorkspaceMembership bool,
) error {
	ctx, err := service.APIKeyAuth(ctx, sessionID, &security.APIKeyScheme{Name: "plane_session_cookie"})
	if err != nil {
		return err
	}
	if !requireWorkspaceMembership {
		return nil
	}
	userID, err := service.userID(ctx)
	if err != nil {
		return err
	}
	if err := service.repository.AuthorizeWorkspace(ctx, userID, slug); err != nil {
		return projectMembershipReadsRepositoryError(err)
	}
	return nil
}

func (service *projectMembershipReadsService) Membership(
	ctx context.Context,
	payload *api.MembershipPayload,
) (*api.ProjectMembership, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.ProjectMembershipReadsNotFound{Message: "The required object does not exist."}
	}
	data, err := service.repository.Membership(ctx, userID, payload.Slug, projectID)
	if err != nil {
		return nil, projectMembershipReadsRepositoryError(err)
	}
	var membership api.ProjectMembership
	if err := json.Unmarshal(data, &membership); err != nil {
		return nil, &api.ProjectMembershipReadsInternalError{Message: projectMembershipReadsInternalErrorDetail}
	}
	return &membership, nil
}

func (service *projectMembershipReadsService) Roles(
	ctx context.Context,
	payload *api.RolesPayload,
) (map[string]int, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	data, err := service.repository.Roles(ctx, userID, payload.Slug)
	if err != nil {
		return nil, projectMembershipReadsRepositoryError(err)
	}
	var roles map[string]int
	if err := json.Unmarshal(data, &roles); err != nil {
		return nil, &api.ProjectMembershipReadsInternalError{Message: projectMembershipReadsInternalErrorDetail}
	}
	return roles, nil
}

func (service *projectMembershipReadsService) WorkspaceMembers(
	ctx context.Context,
	payload *api.WorkspaceMembersPayload,
) ([]*api.WorkspaceMembershipDirectory, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	searchTerms := []string{}
	if payload.Search != nil {
		searchTerms = memberDirectorySearchPatterns(*payload.Search)
	}
	data, err := service.repository.WorkspaceMembers(ctx, userID, payload.Slug, searchTerms)
	return decodeMemberDirectory[[]*api.WorkspaceMembershipDirectory](ctx, data, err, "")
}

func (service *projectMembershipReadsService) WorkspaceMember(
	ctx context.Context,
	payload *api.WorkspaceMemberPayload,
) (*api.WorkspaceMembershipDirectory, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	membershipID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, &api.ProjectMembershipReadsNotFound{Message: "Workspace member not found"}
	}
	searchTerms := []string{}
	if payload.Search != nil {
		searchTerms = memberDirectorySearchPatterns(*payload.Search)
	}
	data, err := service.repository.WorkspaceMember(ctx, userID, payload.Slug, membershipID, searchTerms)
	return decodeMemberDirectory[*api.WorkspaceMembershipDirectory](ctx, data, err, "Workspace member not found")
}

func (service *projectMembershipReadsService) WorkspaceProjectMembers(
	ctx context.Context,
	payload *api.WorkspaceProjectMembersPayload,
) (map[string][]*api.WorkspaceProjectMembershipRole, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	data, err := service.repository.WorkspaceProjectMembers(ctx, userID, payload.Slug)
	if errors.Is(err, store.ErrForbidden) {
		return nil, &api.ProjectMembershipReadsForbidden{Detail: "You do not have permission to perform this action."}
	}
	return decodeMemberDirectory[map[string][]*api.WorkspaceProjectMembershipRole](ctx, data, err, "")
}

func (service *projectMembershipReadsService) ProjectMembers(
	ctx context.Context,
	payload *api.ProjectMembersPayload,
) ([]*api.ProjectMembershipRole, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.ProjectMembershipReadsPermissionDenied{Message: "You don't have the required permissions."}
	}
	data, err := service.repository.ProjectMembers(ctx, userID, payload.Slug, projectID)
	return decodeMemberDirectory[[]*api.ProjectMembershipRole](ctx, data, err, "")
}

// memberDirectorySearchTerms mirrors DRF's search_smart_split behavior: quoted
// phrases stay intact, while unquoted values split on whitespace and commas.
func memberDirectorySearchTerms(value string) []string {
	characters := []rune(value)
	tokens := make([]string, 0)
	for start := 0; start < len(characters); {
		for start < len(characters) && isSearchSpace(characters[start]) {
			start++
		}
		if start == len(characters) {
			break
		}

		end := start
		var quote rune
		for end < len(characters) {
			character := characters[end]
			if quote == 0 {
				if isSearchSpace(character) {
					break
				}
				if (character == '\'' || character == '"') && searchClosingQuote(characters, end+1, character) {
					quote = character
				}
			} else if character == '\\' {
				end++
			} else if character == quote {
				quote = 0
			}
			end++
		}

		token := strings.Trim(string(characters[start:end]), ",")
		if len(token) >= 2 && (token[0] == '\'' || token[0] == '"') && token[len(token)-1] == token[0] {
			quote := token[0]
			token = strings.ReplaceAll(token[1:len(token)-1], "\\"+string(quote), string(quote))
			token = strings.ReplaceAll(token, `\\`, `\`)
			if token != "" {
				tokens = append(tokens, token)
			}
		} else {
			for _, part := range strings.Split(token, ",") {
				if part = strings.TrimSpace(part); part != "" {
					tokens = append(tokens, part)
				}
			}
		}
		start = end
	}
	return tokens
}

func memberDirectorySearchPatterns(value string) []string {
	terms := memberDirectorySearchTerms(value)
	for index := range terms {
		terms[index] = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(terms[index])
	}
	return terms
}

func searchClosingQuote(value []rune, start int, quote rune) bool {
	for index := start; index < len(value); index++ {
		if value[index] == '\\' {
			index++
			continue
		}
		if value[index] == quote {
			return true
		}
	}
	return false
}

func isSearchSpace(value rune) bool {
	return unicode.IsSpace(value)
}

func (service *projectMembershipReadsService) ProjectMember(
	ctx context.Context,
	payload *api.ProjectMemberPayload,
) (*api.ProjectMembershipFullDirectoryOrProjectMembershipGuestDirectory, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.ProjectMembershipReadsNotFound{Message: "Project member not found"}
	}
	membershipID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, &api.ProjectMembershipReadsNotFound{Message: "Project member not found"}
	}
	data, err := service.repository.ProjectMember(ctx, userID, payload.Slug, projectID, membershipID)
	return decodeMemberDirectory[*api.ProjectMembershipFullDirectoryOrProjectMembershipGuestDirectory](ctx, data, err, "Project member not found")
}

func (service *projectMembershipReadsService) Preference(
	ctx context.Context,
	payload *api.PreferencePayload,
) (*api.ProjectMembershipPreference, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.ProjectMembershipReadsNotFound{Message: "The required object does not exist."}
	}
	memberID, err := uuid.Parse(payload.MemberID)
	if err != nil {
		return nil, &api.ProjectMembershipReadsNotFound{Message: "The required object does not exist."}
	}
	data, err := service.repository.Preference(ctx, userID, payload.Slug, projectID, memberID)
	return decodeMemberDirectory[*api.ProjectMembershipPreference](ctx, data, err, "The required object does not exist.")
}

func decodeMemberDirectory[T any](ctx context.Context, data json.RawMessage, err error, notFoundMessage string) (T, error) {
	var result T
	if err != nil {
		switch {
		case errors.Is(err, store.ErrForbidden):
			return result, &api.ProjectMembershipReadsPermissionDenied{Message: "You don't have the required permissions."}
		case errors.Is(err, store.ErrNotFound):
			return result, &api.ProjectMembershipReadsNotFound{Message: notFoundMessage}
		default:
			return result, &api.ProjectMembershipReadsInternalError{Message: projectMembershipReadsInternalErrorDetail}
		}
	}
	if err := decodeProjectMembershipJSON(data, &result); err != nil {
		return result, &api.ProjectMembershipReadsInternalError{Message: projectMembershipReadsInternalErrorDetail}
	}
	captureProjectMembershipReadsResponse(ctx, data)
	return result, nil
}

var projectMembershipRawJSONFields = map[string]struct{}{
	"default_props":             {},
	"explored_features":         {},
	"getting_started_checklist": {},
	"issue_props":               {},
	"logo_props":                {},
	"preferences":               {},
	"tips":                      {},
	"view_props":                {},
}

func decodeProjectMembershipJSON(data json.RawMessage, target any) error {
	sanitized, err := sanitizeProjectMembershipJSON(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(sanitized, target)
}

func sanitizeProjectMembershipJSON(data json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("empty JSON response")
	}
	switch trimmed[0] {
	case '{':
		var object map[string]json.RawMessage
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, err
		}
		for key, value := range object {
			if _, dynamic := projectMembershipRawJSONFields[key]; dynamic {
				object[key] = json.RawMessage(`{}`)
				continue
			}
			sanitized, err := sanitizeProjectMembershipJSON(value)
			if err != nil {
				return nil, err
			}
			object[key] = sanitized
		}
		return json.Marshal(object)
	case '[':
		var values []json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil {
			return nil, err
		}
		for index := range values {
			sanitized, err := sanitizeProjectMembershipJSON(values[index])
			if err != nil {
				return nil, err
			}
			values[index] = sanitized
		}
		return json.Marshal(values)
	default:
		return append(json.RawMessage(nil), data...), nil
	}
}

func captureProjectMembershipReadsResponse(ctx context.Context, data json.RawMessage) {
	capture, ok := ctx.Value(projectMembershipReadsResponseCaptureKey{}).(*ProjectMembershipReadsResponseCapture)
	if ok && capture != nil {
		capture.Data = append(capture.Data[:0], data...)
	}
}

func (service *projectMembershipReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(projectMembershipReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.ProjectMembershipReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func projectMembershipReadsRepositoryError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return &api.ProjectMembershipReadsNotFound{Message: "The required object does not exist."}
	case errors.Is(err, store.ErrForbidden):
		return &api.ProjectMembershipReadsForbidden{Detail: "You do not have permission to perform this action."}
	default:
		return &api.ProjectMembershipReadsInternalError{Message: projectMembershipReadsInternalErrorDetail}
	}
}

var _ projectMembershipReadsRepository = (*store.Repository)(nil)
var _ ProjectMembershipReadsMetadataAuthorizer = (*projectMembershipReadsService)(nil)
