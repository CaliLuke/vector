package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"uuid"

	authapi "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	authstore "github.com/CaliLuke/go-plane/internal/accountauthentication"
	preferencesstore "github.com/CaliLuke/go-plane/internal/projectmemberpreferences"
	preferencesapi "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
	searchapi "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

const privateOperationDetail = "fixture private operation detail"

func capturedOperationContext(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()
	var output bytes.Buffer
	ctx := log.Context(t.Context(),
		log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}),
		log.WithDisableBuffering(func(context.Context) bool { return true }),
	)
	return ctx, &output
}

func assertOperationDiagnostic(t *testing.T, output *bytes.Buffer, operation, stage, kind string) {
	t.Helper()
	require.NotContains(t, output.String(), privateOperationDetail)
	decoder := json.NewDecoder(output)
	var entry map[string]any
	require.NoError(t, decoder.Decode(&entry))
	require.Equal(t, operation, entry["operation"])
	require.Equal(t, stage, entry["stage"])
	require.Equal(t, kind, entry["error_kind"])
	require.ErrorIs(t, decoder.Decode(&entry), io.EOF)
}

func TestPasswordResetRecordsSafeFailureStage(t *testing.T) {
	for _, stage := range []string{
		"check_rate_limit", "read_instance_configuration", "validate_email", "read_mail_configuration", "lookup_account",
		"issue_reset_token", "admit_reset_delivery",
	} {
		t.Run(stage, func(t *testing.T) {
			ctx, logs := capturedOperationContext(t)
			const email = "synthetic.inbox@example.com"
			const token = "synthetic-reset-secret"
			repo := &accountAuthenticationTestRepo{configured: true, config: map[string]string{"EMAIL_HOST": "smtp.test"}, user: authstore.User{Email: email}}
			tasks := &accountAuthenticationTestTasks{}
			throttle := &accountAuthenticationTestThrottle{allowed: true}
			service := newAccountAuthenticationTestService(t, repo, throttle, tasks, accountAuthenticationTestMagic{})
			failure := fmt.Errorf("%s: %w", privateOperationDetail, context.DeadlineExceeded)
			reset := accountAuthenticationTestResetTokens{uid: "fixture-user", token: token}
			operation := "account_authentication.forgot_password"
			input := email
			switch stage {
			case "check_rate_limit":
				throttle.err = failure
				operation = "account_authentication.throttle"
			case "read_instance_configuration":
				repo.configurationErr = failure
			case "validate_email":
				input = "invalid existing address"
				repo.userErr = failure
			case "read_mail_configuration":
				repo.mailConfigErr = failure
			case "lookup_account":
				repo.userErr = failure
			case "issue_reset_token":
				reset.err = failure
			case "admit_reset_delivery":
				tasks.err = failure
			}
			service.resetTokens = reset
			contentType := "application/json"
			_, _, err := service.ForgotPassword(ctx, &authapi.ForgotPasswordPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"email":"`+input+`"}`)))
			status, response := accountAuthenticationProtocolError(err)
			require.Equal(t, http.StatusInternalServerError, status)
			require.Equal(t, accountAuthenticationInternalMessage, response["error"])
			require.NotContains(t, logs.String(), email)
			require.NotContains(t, logs.String(), token)
			assertOperationDiagnostic(t, logs, operation, stage, "deadline_exceeded")
		})
	}
}

type projectPreferenceDiagnosticRepo struct {
	failureStage string
	failure      error
	response     map[string]any
}

func (r projectPreferenceDiagnosticRepo) Authorize(context.Context, string, string, string) error {
	if r.failureStage == "authorize" {
		return r.failure
	}
	return nil
}

func (r projectPreferenceDiagnosticRepo) Lookup(context.Context, string, string, string) error {
	if r.failureStage == "lookup_member" {
		return r.failure
	}
	return nil
}

func (r projectPreferenceDiagnosticRepo) Patch(context.Context, string, string, string, string, map[string]any) (map[string]any, error) {
	if r.failureStage == "patch" {
		return nil, r.failure
	}
	return r.response, nil
}

func TestProjectMemberPreferencesRecordsSafeFailureStage(t *testing.T) {
	pgFailure := &pgconn.PgError{Code: "XX000", Message: privateOperationDetail}
	for _, tc := range []struct {
		name, stage, kind string
		failure           error
		status            int
	}{
		{"authorize", "authorize", "postgres", pgFailure, 500},
		{"lookup", "lookup_member", "postgres", pgFailure, 500},
		{"write", "patch", "postgres", pgFailure, 500},
		{"encoding", "encode_response", "other", nil, 500},
		{"known denial", "authorize", "", &preferencesstore.HTTPError{Status: 403, Body: map[string]any{"detail": "Denied"}}, 403},
		{"known internal error", "patch", "other", &preferencesstore.HTTPError{Status: 500, Body: map[string]any{"detail": "Internal"}}, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logs := capturedOperationContext(t)
			ctx = context.WithValue(ctx, projectMemberPreferencesPrincipal{}, projectMemberPreferencesPrincipal{userID: "fixture", timezone: "UTC"})
			service := projectMemberPreferencesService{repository: projectPreferenceDiagnosticRepo{
				failureStage: tc.stage,
				failure:      tc.failure,
				response:     map[string]any{"preferences": func() {}},
			}}
			_, err := service.Patch(ctx, &preferencesapi.PatchPayload{}, nil)
			status, _ := preferenceProtocolError(err)
			require.Equal(t, tc.status, status)
			if tc.kind == "" {
				require.NotContains(t, logs.String(), projectPreferencesFailure)
				return
			}
			assertOperationDiagnostic(t, logs, projectPreferencesPatch, tc.stage, tc.kind)
		})
	}
}

type searchDiagnosticRepo struct{ authorizeErr, queryErr error }

func (r searchDiagnosticRepo) WorkspaceAllowed(context.Context, uuid.UUID, string) (bool, error) {
	return true, r.authorizeErr
}
func (r searchDiagnosticRepo) Global(context.Context, uuid.UUID, string, string, string, string, string) (json.RawMessage, error) {
	return nil, r.queryErr
}
func (r searchDiagnosticRepo) Entity(context.Context, uuid.UUID, string, string, string, string, int) (json.RawMessage, error) {
	return nil, r.queryErr
}

func TestWorkspaceSearchRecordsSafeFailureStage(t *testing.T) {
	for _, tc := range []struct{ name, operation, stage string }{
		{"global query", "workspace_search.global", "query"},
		{"entity query", workspaceSearchEntity, "query"},
		{"entity authorization", workspaceSearchEntity, "authorize"},
		{"invalid session timezone", workspaceSearchEntity, "validate_timezone"},
		{"invalid response", "workspace_search.global", "validate_response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logs := capturedOperationContext(t)
			timezone := "UTC"
			if tc.stage == "validate_timezone" {
				timezone = ""
			}
			ctx = context.WithValue(ctx, workspaceSearchActorKey{}, workspaceSearchActor{id: uuid.New(), timezone: timezone})
			failure := errors.New(privateOperationDetail)
			repo := searchDiagnosticRepo{queryErr: failure}
			if tc.stage == "validate_response" {
				repo.queryErr = nil
			}
			if tc.stage == "authorize" {
				repo.authorizeErr = failure
			}
			service := workspaceSearchService{repository: repo}
			var err error
			if tc.operation == workspaceSearchEntity {
				_, _, err = service.EntitySearch(ctx, &searchapi.EntitySearchPayload{})
			} else {
				_, _, err = service.GlobalSearch(ctx, &searchapi.GlobalSearchPayload{})
			}
			status, body := workspaceSearchProtocolError(err)
			require.Equal(t, http.StatusInternalServerError, status)
			require.Equal(t, "Something went wrong please try again later", body["detail"])
			assertOperationDiagnostic(t, logs, tc.operation, tc.stage, "other")
		})
	}
}
