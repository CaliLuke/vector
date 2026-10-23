package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	api "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/externalintegrations"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type externalIntegrationsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type externalIntegrationsRepository interface {
	Configuration(context.Context, []string, map[string]string, bool) (map[string]string, error)
	WorkspaceRole(context.Context, string, string) (int, error)
	ProjectAllowed(context.Context, string, string, string) (bool, error)
	WorkspaceLite(context.Context, string) (json.RawMessage, error)
	ProjectLite(context.Context, string) (json.RawMessage, error)
}

type externalIntegrationsService struct {
	repository   externalIntegrationsRepository
	sessions     externalIntegrationsSessions
	client       *http.Client
	unsplashBase string
	complete     func(context.Context, string, string, string) (string, error)
}

type externalIntegrationsPrincipal struct {
	userID   string
	timezone string
}
type externalIntegrationsPrincipalKey struct{}

func NewExternalIntegrations(repository externalIntegrationsRepository, sessions externalIntegrationsSessions, client *http.Client) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("external integrations repository and session validator are required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	base := os.Getenv("UNSPLASH_API_BASE_URL")
	if base == "" {
		base = "https://api.unsplash.com"
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid Unsplash API base URL")
	}
	return &externalIntegrationsService{repository: repository, sessions: sessions, client: client, unsplashBase: strings.TrimRight(base, "/"), complete: openAIComplete}, nil
}

func (s *externalIntegrationsService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, sessionErr := serviceSessionIdentityFromRequest(ctx, credential, s.sessions)
	if sessionErr != nil {
		userID, timezone = "", ""
	}
	if userID == "" {
		return ctx, s.error(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	if err := djangosession.ValidateUserTimezone(timezone); err != nil {
		return ctx, s.internal()
	}
	return context.WithValue(ctx, externalIntegrationsPrincipalKey{}, externalIntegrationsPrincipal{userID: userID, timezone: timezone}), nil
}

func (s *externalIntegrationsService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	return externalIntegrationsProtocolError(err)
}

func (s *externalIntegrationsService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	return externalIntegrationsProtocolError(err)
}

func (s *externalIntegrationsService) Unsplash(ctx context.Context, p *api.UnsplashPayload) (*api.ExternalIntegrationRawJSON, io.ReadCloser, error) {
	config, err := s.configuration(ctx, "UNSPLASH_ACCESS_KEY")
	if err != nil {
		return nil, nil, s.internal()
	}
	key := config["UNSPLASH_ACCESS_KEY"]
	if key == "" {
		return externalIntegrationsRaw(json.RawMessage(`[]`))
	}
	query, page, perPage := "", "1", "20"
	if p.Query != nil {
		query = *p.Query
	}
	if p.Page != nil {
		page = *p.Page
	}
	if p.PerPage != nil {
		perPage = *p.PerPage
	}
	endpoint := s.unsplashBase + "/photos/"
	queryValues := url.Values{}
	queryValues.Set("client_id", key)
	queryValues.Set("page", page)
	queryValues.Set("per_page", perPage)
	if query != "" {
		endpoint = s.unsplashBase + "/search/photos/"
		queryValues.Set("query", query)
	}
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, s.internal()
	}
	parsedURL.RawQuery = queryValues.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, nil, s.internal()
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return nil, nil, s.internal()
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, s.internal()
	}
	if !json.Valid(body) {
		return nil, nil, s.error(502, map[string]any{"detail": "Invalid JSON response from Unsplash."})
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, nil, s.errorJSON(response.StatusCode, body)
	}
	return externalIntegrationsRaw(json.RawMessage(body))
}

func (s *externalIntegrationsService) WorkspaceAiAssistant(ctx context.Context, p *api.WorkspaceAiAssistantPayload, body io.ReadCloser) (*api.ExternalIntegrationRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		closeExternalBody(body)
		return nil, nil, err
	}
	role, err := s.repository.WorkspaceRole(ctx, actor.userID, p.Slug)
	if err != nil {
		closeExternalBody(body)
		if errors.Is(err, store.ErrWorkspaceForbidden) {
			return nil, nil, s.error(403, map[string]any{"error": "You don't have the required permissions."})
		}
		return nil, nil, s.internal()
	}
	if role != 15 && role != 20 {
		closeExternalBody(body)
		return nil, nil, s.error(403, map[string]any{"error": "You don't have the required permissions."})
	}
	config, err := s.llmConfiguration(ctx)
	if err != nil {
		closeExternalBody(body)
		return nil, nil, s.error(400, map[string]any{"error": "LLM provider API key and model are required"})
	}
	request, err := parseExternalAIRequest(ctx, body)
	if err != nil {

		return nil, nil, s.error(400, map[string]any{"error": err.Error()})
	}
	text, err := s.generate(ctx, request, config)
	if err != nil {
		return nil, nil, s.internal()
	}
	return externalIntegrationsRaw(map[string]any{"response": text, "response_html": strings.ReplaceAll(text, "\n", "<br/>")})
}

func (s *externalIntegrationsService) ProjectAiAssistant(ctx context.Context, p *api.ProjectAiAssistantPayload, body io.ReadCloser) (*api.ExternalIntegrationRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		closeExternalBody(body)
		return nil, nil, err
	}
	allowed, err := s.repository.ProjectAllowed(ctx, actor.userID, p.Slug, p.ProjectID)
	if err != nil {
		closeExternalBody(body)
		return nil, nil, s.internal()
	}
	if !allowed {
		closeExternalBody(body)
		return nil, nil, s.error(403, map[string]any{"error": "You don't have the required permissions."})
	}
	config, err := s.llmConfiguration(ctx)
	if err != nil {
		closeExternalBody(body)
		return nil, nil, s.error(400, map[string]any{"error": "LLM provider API key and model are required"})
	}
	request, err := parseExternalAIRequest(ctx, body)
	if err != nil {

		return nil, nil, s.error(400, map[string]any{"error": err.Error()})
	}
	text, err := s.generate(ctx, request, config)
	if err != nil {
		return nil, nil, s.internal()
	}
	workspace, err := s.repository.WorkspaceLite(ctx, p.Slug)
	if err != nil {
		if errors.Is(err, store.ErrWorkspaceMissing) {
			return nil, nil, s.error(404, map[string]any{"detail": "Not found."})
		}
		return nil, nil, s.internal()
	}
	project, err := s.repository.ProjectLite(ctx, p.ProjectID)
	if err != nil {
		if errors.Is(err, store.ErrProjectMissing) {
			return nil, nil, s.error(404, map[string]any{"detail": "Not found."})
		}
		return nil, nil, s.internal()
	}
	var workspaceValue, projectValue any
	if json.Unmarshal(workspace, &workspaceValue) != nil || json.Unmarshal(project, &projectValue) != nil {
		return nil, nil, s.internal()
	}
	return externalIntegrationsRaw(map[string]any{"response": text, "response_html": strings.ReplaceAll(text, "\n", "<br/>"), "project_detail": projectValue, "workspace_detail": workspaceValue})
}

type externalAIRequest struct {
	task   any
	prompt any
}

func parseExternalAIRequest(ctx context.Context, body io.ReadCloser) (externalAIRequest, error) {
	defer closeExternalBody(body)
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return externalAIRequest{}, err
	}
	if input.Kind != "" {
		return externalAIRequest{}, errExternalAIExpectedObject
	}
	request := externalAIRequest{task: input.Fields["task"], prompt: input.Fields["prompt"]}
	if !pythoncompat.Truthy(request.task) {
		return request, errExternalTaskRequired
	}
	if _, ok := request.task.(string); !ok {
		return request, errExternalAIStringFields
	}
	if _, ok := request.prompt.(string); !ok {
		return request, errExternalAIStringFields
	}
	return request, nil
}

var errExternalTaskRequired = errors.New("Task is required")
var errExternalAIExpectedObject = errors.New("Expected an object.")
var errExternalAIStringFields = errors.New("Task and prompt must be strings.")

type externalLLMConfiguration struct {
	key, provider, model string
	models               []string
}

func (s *externalIntegrationsService) llmConfiguration(ctx context.Context) (externalLLMConfiguration, error) {
	config, err := s.configuration(ctx, "LLM_API_KEY", "LLM_PROVIDER", "LLM_MODEL")
	if err != nil {
		return externalLLMConfiguration{}, err
	}
	key, provider, model := config["LLM_API_KEY"], strings.ToLower(config["LLM_PROVIDER"]), config["LLM_MODEL"]
	if provider == "" {
		provider = "openai"
	}
	defaults, ok := map[string]struct {
		model  string
		models []string
	}{
		"openai":    {"gpt-4o-mini", []string{"gpt-3.5-turbo", "gpt-4o-mini", "gpt-4o", "o1-mini", "o1-preview"}},
		"anthropic": {"claude-3-sonnet-20240229", []string{"claude-3-5-sonnet-20240620", "claude-3-haiku-20240307", "claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-2.1", "claude-2", "claude-instant-1.2", "claude-instant-1"}},
		"gemini":    {"gemini-pro", []string{"gemini-pro", "gemini-1.5-pro-latest", "gemini-pro-vision"}},
	}[provider]
	if !ok || key == "" {
		return externalLLMConfiguration{}, fmt.Errorf("missing or unsupported LLM configuration")
	}
	if model == "" {
		model = defaults.model
	}
	valid := false
	for _, candidate := range defaults.models {
		if model == candidate {
			valid = true
			break
		}
	}
	if !valid {
		return externalLLMConfiguration{}, fmt.Errorf("unsupported model for provider")
	}
	return externalLLMConfiguration{key: key, provider: provider, model: model, models: defaults.models}, nil
}

func (s *externalIntegrationsService) generate(ctx context.Context, request externalAIRequest, config externalLLMConfiguration) (string, error) {
	key, provider, model := config.key, config.provider, config.model
	task, taskOK := request.task.(string)
	prompt, promptOK := request.prompt.(string)
	if !taskOK || !promptOK {
		return "", fmt.Errorf("source concatenation type error")
	}
	if provider == "gemini" {
		model = "gemini/" + model
	}
	return s.complete(ctx, key, model, task+"\n"+prompt)
}

func (s *externalIntegrationsService) configuration(ctx context.Context, keys ...string) (map[string]string, error) {
	environment := make(map[string]string, len(keys))
	for _, key := range keys {
		environment[key] = os.Getenv(key)
	}
	return s.repository.Configuration(ctx, keys, environment, os.Getenv("SKIP_ENV_VAR") == "true" || os.Getenv("SKIP_ENV_VAR") == "1")
}

func (s *externalIntegrationsService) actor(ctx context.Context) (externalIntegrationsPrincipal, error) {
	actor, ok := ctx.Value(externalIntegrationsPrincipalKey{}).(externalIntegrationsPrincipal)
	if !ok || actor.userID == "" {
		return externalIntegrationsPrincipal{}, s.error(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	return actor, nil
}

func (s *externalIntegrationsService) error(status int, body any) error {
	if status < 300 || status > 599 {
		status = 502
	}
	data, err := json.Marshal(body)
	if err != nil {
		data = []byte(`{"detail":"Something went wrong please try again later"}`)
	}
	return s.errorJSON(status, data)
}

func (s *externalIntegrationsService) errorJSON(status int, body []byte) error {
	if status < 300 || status > 599 {
		status = 502
	}
	if !json.Valid(body) {
		body = []byte(`null`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(append([]byte(nil), body...)))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.ExternalIntegrationErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}

func (s *externalIntegrationsService) internal() error {
	return s.error(500, map[string]any{"error": "An internal error has occurred."})
}

func externalIntegrationsRaw(value any) (*api.ExternalIntegrationRawJSON, io.ReadCloser, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, nil, err
	}
	return &api.ExternalIntegrationRawJSON{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func closeExternalBody(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}

func openAIComplete(ctx context.Context, key, model, prompt string) (string, error) {
	client := openai.NewClient(option.WithAPIKey(key))
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{Model: model, Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt)}})
	if err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("LLM response contained no choices")
	}
	return completion.Choices[0].Message.Content, nil
}

func externalIntegrationsProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var response *api.ExternalIntegrationErrorResponse
	if errors.As(err, &response) {
		status, _ := strconv.Atoi(strings.TrimPrefix(response.Name, "status_"))
		var body map[string]any
		if json.Unmarshal([]byte(response.Body), &body) != nil {
			body = map[string]any{"detail": "Something went wrong please try again later"}
		}
		return status, body
	}
	return http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"}
}
