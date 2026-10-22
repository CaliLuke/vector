package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/url"
	"strings"
	"time"

	api "github.com/CaliLuke/go-plane/authredirects/gen/auth_redirects"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	core "github.com/CaliLuke/go-plane/internal/authredirects"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

const authRedirectContentType = "text/html; charset=utf-8"

// NewAuthRedirects adapts the domain authentication workflows to Loom's
// generated form endpoints. Session response actions are applied by the
// shared Django auth middleware.
func NewAuthRedirects(service *core.Service) (api.Service, error) {
	if service == nil {
		return nil, errors.New("auth redirect service is required")
	}
	return &authRedirectsService{service: service}, nil
}

type authRedirectsService struct{ service *core.Service }

func (s *authRedirectsService) EmailSignIn(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, false, func(request core.Request) core.Result { return s.service.EmailSignIn(ctx, request, true) })
}
func (s *authRedirectsService) EmailSignUp(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, false, func(request core.Request) core.Result { return s.service.EmailSignUp(ctx, request, true) })
}
func (s *authRedirectsService) SpaceEmailSignIn(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, true, func(request core.Request) core.Result { return s.service.EmailSignIn(ctx, request, false) })
}
func (s *authRedirectsService) SpaceEmailSignUp(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, true, func(request core.Request) core.Result { return s.service.EmailSignUp(ctx, request, false) })
}
func (s *authRedirectsService) SignOut(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, false, func(request core.Request) core.Result {
		return s.service.SignOut(ctx, request, authRedirectSessionUser(request.CurrentSession))
	})
}
func (s *authRedirectsService) SpaceSignOut(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, true, func(request core.Request) core.Result {
		return s.service.SignOut(ctx, request, authRedirectSessionUser(request.CurrentSession))
	})
}
func (s *authRedirectsService) MagicSignIn(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, false, func(request core.Request) core.Result { return s.service.MagicSignIn(ctx, request, true) })
}
func (s *authRedirectsService) MagicSignUp(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, false, func(request core.Request) core.Result { return s.service.MagicSignUp(ctx, request, true) })
}
func (s *authRedirectsService) SpaceMagicSignIn(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, true, func(request core.Request) core.Result { return s.service.MagicSignIn(ctx, request, false) })
}
func (s *authRedirectsService) SpaceMagicSignUp(ctx context.Context, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, true, func(request core.Request) core.Result { return s.service.MagicSignUp(ctx, request, false) })
}
func (s *authRedirectsService) ResetPassword(ctx context.Context, payload *api.ResetPasswordPayload, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, false, func(request core.Request) core.Result {
		request.PasswordUID, request.PasswordToken = payload.Uidb64, payload.Token
		return s.service.ResetPassword(ctx, request)
	})
}
func (s *authRedirectsService) SpaceResetPassword(ctx context.Context, payload *api.SpaceResetPasswordPayload, body io.ReadCloser) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	return s.execute(ctx, body, true, func(request core.Request) core.Result {
		request.PasswordUID, request.PasswordToken = payload.Uidb64, payload.Token
		return s.service.ResetPassword(ctx, request)
	})
}

func (s *authRedirectsService) execute(ctx context.Context, body io.ReadCloser, space bool, run func(core.Request) core.Result) (*api.AuthRedirectResponse, io.ReadCloser, error) {
	request, err := s.request(ctx, body, space)
	if err != nil {
		return nil, nil, authRedirectHTTPError(err)
	}
	result := run(request)
	if result.Session != nil {
		djangosession.SetResponseSession(ctx, *result.Session)
	}
	if result.ExpireSession {
		djangosession.ExpireResponseSession(ctx)
	}
	if result.Status != 302 {
		return nil, nil, authRedirectHTTPError(errors.New("authentication redirect failed"))
	}
	return &api.AuthRedirectResponse{Location: result.Location, ContentType: authRedirectContentType}, io.NopCloser(strings.NewReader(djangoRedirectBody(result.Location))), nil
}

func (s *authRedirectsService) request(ctx context.Context, body io.ReadCloser, space bool) (core.Request, error) {
	if body != nil {
		defer body.Close()
	}
	input, err := djangoinput.Parse(body, authRedirectRequestContentType(ctx))
	if err != nil {
		return core.Request{}, err
	}
	text := func(key string) string {
		value, present := input.FieldValue(key, false, true, false)
		if !present || value == nil {
			return ""
		}
		parsed, _ := djangoinput.TextField(value, true, 0, nil)
		return parsed
	}
	web, err := s.service.ConfigValue(ctx, "WEB_URL", "")
	if err != nil {
		return core.Request{}, err
	}
	app, err := s.service.ConfigValue(ctx, "APP_BASE_URL", "")
	if err != nil {
		return core.Request{}, err
	}
	base := web
	if base == "" {
		base = app
	}
	if !space && app != "" {
		base = app
	}
	if space {
		spaceURL, configErr := s.service.ConfigValue(ctx, "SPACE_BASE_URL", "")
		if configErr != nil {
			return core.Request{}, configErr
		}
		spacePath, configErr := s.service.ConfigValue(ctx, "SPACE_BASE_PATH", "/spaces/")
		if configErr != nil {
			return core.Request{}, configErr
		}
		if spaceURL != "" {
			base = spaceURL
		}
		if !strings.HasPrefix(spacePath, "/") {
			spacePath = "/" + spacePath
		}
		if !strings.HasSuffix(spacePath, "/") {
			spacePath += "/"
		}
		base = strings.TrimRight(base, "/") + spacePath
	} else {
		base = strings.TrimRight(base, "/")
	}
	allowed := make([]string, 0, 4)
	admin, err := s.service.ConfigValue(ctx, "ADMIN_BASE_URL", "")
	if err != nil {
		return core.Request{}, err
	}
	spaceURL, err := s.service.ConfigValue(ctx, "SPACE_BASE_URL", "")
	if err != nil {
		return core.Request{}, err
	}
	for _, configured := range []string{web, app, admin, spaceURL} {
		if parsed, parseErr := url.Parse(configured); parseErr == nil && parsed.Host != "" {
			allowed = append(allowed, parsed.Host)
		}
	}
	var userAgent, clientIP string
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		userAgent, clientIP = metadata.UserAgent, metadata.ClientAddr
	}
	current, _ := djangosession.CurrentSessionFromContext(ctx)
	request := core.Request{Email: text("email"), Password: text("password"), Code: text("code"), NextPath: text("next_path"), BaseURL: base, AllowedHosts: allowed, UserAgent: userAgent, ClientIP: clientIP, IsSpace: space, BaseOrigin: web, CurrentSession: current, Now: time.Now().UTC()}
	return request, nil
}

func authRedirectRequestContentType(ctx context.Context) string {
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	if !ok {
		return ""
	}
	return metadata.Headers().Get("Content-Type")
}

func authRedirectSessionUser(session *djangosession.SessionRefresh) string {
	if session == nil {
		return ""
	}
	value, _ := session.Get("_auth_user_id")
	userID, _ := value.(string)
	return userID
}

func authRedirectHTTPError(err error) error {
	body, _ := json.Marshal(map[string]string{"detail": "Something went wrong please try again later"})
	_ = err
	return &api.AuthRedirectError{Name: "status_500", Body: loom.JSONValue(body)}
}

func djangoRedirectBody(location string) string {
	return "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n  <meta http-equiv=\"content-type\" content=\"text/html; charset=utf-8\">\n  <title>Redirecting...</title>\n</head>\n<body>\n  <h1>Redirecting...</h1>\n  <p>You should be redirected automatically to the target URL: <a href=\"" + html.EscapeString(location) + "\">" + html.EscapeString(location) + "</a>.</p>\n</body>\n</html>"
}
