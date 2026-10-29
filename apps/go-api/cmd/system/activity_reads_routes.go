package main

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	activityreadssvr "github.com/CaliLuke/go-plane/activityreads/gen/http/plane_internal_django_migration_api/server"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountActivityReadsProtocolRoutes(mux loomhttp.Muxer, server *activityreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	loomhttp.MountDerivedHead(mux, "/api/users/me/activities/", corsPolicy.Handler(loomhttp.AsHandlerFunc(server.GetAPIUsersMeActivities)))
}

type activityReadsMethodAuthorizer struct {
	readOnly systemapi.ActivityReadsProtocolAuthorizer
	primary  systemapi.ActivityReadsProtocolAuthorizer
	identity systemapi.APIIdentityProtocolAuthorizer
}

func (authorizer activityReadsMethodAuthorizer) AuthorizeAPIIdentityProtocol(ctx context.Context, key string) (int, map[string]any) {
	if authorizer.identity == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return authorizer.identity.AuthorizeAPIIdentityProtocol(ctx, key)
}

func (authorizer activityReadsMethodAuthorizer) AuthorizeAPIActivityProtocol(ctx context.Context, key, slug, projectID, method string) (int, map[string]any) {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return authorizer.readOnly.AuthorizeAPIActivityProtocol(ctx, key, slug, projectID, method)
	}
	return authorizer.primary.AuthorizeAPIActivityProtocol(ctx, key, slug, projectID, method)
}

func (authorizer activityReadsMethodAuthorizer) AuthorizeUserActivityProtocol(ctx context.Context, session string) (int, map[string]any) {
	return authorizer.primary.AuthorizeUserActivityProtocol(ctx, session)
}

func newActivityReadsProtocolMiddleware(
	authorizer systemapi.ActivityReadsProtocolAuthorizer,
	corsPolicy loomhttp.RuntimeCORSPolicy,
	stack func(http.Handler) http.Handler,
	compatibilityOptions apiV1CompatibilityOptions,
	cookieConfig djangoSessionCookieConfig,
	bodySizeLimit int64,
	debugMode bool,
) func(http.Handler) http.Handler {
	apiMethodNotAllowed := activityReadsAPIMethodNotAllowedHandler(authorizer)
	apiOrdinary := corsPolicy.Handler(loomhttp.AsHandlerFunc(stack(apiMethodNotAllowed)))
	apiPreflight := djangoProtocolCORSOptionsHandler(corsPolicy, []string{http.MethodGet}, apiMethodNotAllowed)
	userOrdinary := corsPolicy.Handler(loomhttp.AsHandlerFunc(stack(activityReadsUserMethodNotAllowedHandler(authorizer, cookieConfig))))
	userPreflight := djangoProtocolCORSOptionsHandler(corsPolicy, []string{http.MethodGet}, userOrdinary)
	return func(next http.Handler) http.Handler {
		slashRedirect := activityReadsSlashRedirectMiddleware(next, debugMode)
		apiSlashlessHandler := activityReadsSlashlessCompatibilityHandler(stack, slashRedirect)
		userSlashlessHandler := activityReadsSlashlessCompatibilityHandler(func(next http.Handler) http.Handler {
			next = activityReadsCompatibilityMiddleware(compatibilityOptions)(next)
			next = activityReadsRequestSizeMiddleware(bodySizeLimit)(next)
			return next
		}, slashRedirect)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
			slashless := false
			if !matched && !strings.HasSuffix(request.URL.Path, "/") {
				candidatePath := request.URL.Path + "/"
				protocol, matched = resolveReadServiceProtocolForMethod(candidatePath, request.Method)
				if matched && readServiceProtocolHasMalformedUUID(candidatePath, protocol) {
					matched = false
				}
				format, _, stickyFormatPath := apiV1StickyFormatSuffix(request.URL.Path)
				if stickyFormatPath && format == "json" {
					slashless = false
				} else {
					slashless = matched
				}
			}
			activityAPI := matched && activityReadsAPIProtocol(protocol.pattern)
			userActivity := matched && protocol.pattern == "/api/users/me/activities/"
			preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
			if activityAPI && preflight {
				request = withReadServicePathValues(request, protocol)
				apiPreflight.ServeHTTP(response, request)
				return
			}
			if userActivity && preflight {
				userPreflight.ServeHTTP(response, request)
				return
			}
			if slashless {
				if activityAPI {
					apiSlashlessHandler.ServeHTTP(response, request)
					return
				}
				if userActivity {
					userSlashlessHandler.ServeHTTP(response, request)
					return
				}
			}
			unsupportedUserMethod := userActivity && !readServiceMethodAllowed(protocol.allow, request.Method)
			if slices.Contains(protocol.passthroughMethods, request.Method) || (!activityAPI || request.Method == http.MethodGet && protocol.apiAuthorization != "assets" && !protocol.generatedMethods || (protocol.workItem || protocol.generatedMethods) && readServiceMethodAllowed(protocol.allow, request.Method) && (protocol.apiAuthorization != "invitations" || request.Method != http.MethodOptions)) && !unsupportedUserMethod {
				next.ServeHTTP(response, request)
				return
			}
			request = withReadServicePathValues(request, protocol)
			if activityAPI {
				apiOrdinary.ServeHTTP(response, request)
				return
			}
			userOrdinary.ServeHTTP(response, request)
		})
	}
}

func activityReadsAPIMethodNotAllowedHandler(authorizer systemapi.ActivityReadsProtocolAuthorizer) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var status int
		var body map[string]any
		protocol, _ := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)

		switch protocol.apiAuthorization {
		case "stickies":
			if stickies, ok := authorizer.(systemapi.APIV1StickyProtocolAuthorizer); ok {
				status, body = stickies.AuthorizeAPIV1StickyProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), request.PathValue("slug"))
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		case "project_configuration":
			if configuration, ok := authorizer.(interface {
				AuthorizeAPIV1LabelProtocol(context.Context, string, string, string) (int, map[string]any)
			}); ok {
				status, body = configuration.AuthorizeAPIV1LabelProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), request.PathValue("slug"), request.PathValue("project_id"))
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		case "project_lifecycle":
			if project, ok := authorizer.(interface {
				AuthorizeProjectLifecycleAPIProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any)
			}); ok {
				status, body = project.AuthorizeProjectLifecycleAPIProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), sessionProtocolResource(request, protocol))
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		case "assets":
			if assets, ok := authorizer.(systemapi.AssetsAPIProtocolAuthorizer); ok {
				status, body = assets.AuthorizeAssetsAPIProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), sessionProtocolResource(request, protocol))
			} else {
				status = 500
				body = map[string]any{"error": "Something went wrong please try again later"}
			}
		case "invitations":
			if invitations, ok := authorizer.(systemapi.InvitationsMembershipProtocolAuthorizer); ok {
				status, body = invitations.AuthorizeInvitationsMembershipProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), sessionProtocolResource(request, protocol), true)
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		case "membership":
			if membership, ok := authorizer.(systemapi.MembershipLifecycleProtocolAuthorizer); ok {
				status, body = membership.AuthorizeMembershipLifecycleProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), systemapi.SessionProtocolResource{Method: request.Method, Pattern: protocol.pattern, Slug: request.PathValue("slug"), ProjectID: request.PathValue("project_id")}, true)
			} else {
				status = 500
				body = map[string]any{"error": "Something went wrong please try again later"}
			}
		case "identity":
			identity, ok := authorizer.(systemapi.APIIdentityProtocolAuthorizer)
			if !ok {
				writeReadServiceJSON(response, 500, map[string]string{"error": "Something went wrong please try again later"})
				return
			}
			status, body = identity.AuthorizeAPIIdentityProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"))
		default:
			status, body = authorizer.AuthorizeAPIActivityProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), request.PathValue("slug"), request.PathValue("project_id"), request.Method)
		}

		if status != 0 {
			writeReadServiceJSON(response, status, body)
			return
		}
		if protocol, ok := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method); ok {
			response.Header().Set("Allow", protocol.allow)
			if protocol.apiAuthorization == "invitations" && request.Method == http.MethodOptions && readServiceMethodAllowed(protocol.allow, http.MethodOptions) {
				status, metadata := notificationProtocolMetadata(protocol)
				writeReadServiceJSON(response, status, metadata)
				return
			}
		}
		writeReadServiceJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}

func activityReadsUserMethodNotAllowedHandler(authorizer systemapi.ActivityReadsProtocolAuthorizer, cookieConfig djangoSessionCookieConfig) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		sessionID := ""
		if cookie, err := request.Cookie(sessionCookieName(cookieConfig)); err == nil {
			sessionID = cookie.Value
		}
		status, body := authorizer.AuthorizeUserActivityProtocol(request.Context(), sessionID)
		if status != 0 {
			writeReadServiceJSON(response, status, body)
			return
		}
		writeReadServiceJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}

type activityReadsSessionAccess struct {
	accessed bool
}

type activityReadsAuditState struct {
	logged bool
}

type activityReadsAuditStateKey struct{}

type activityReadsSessionAccessKey struct{}

type activityReadsSessionFinalizationResponseWriter struct {
	http.ResponseWriter
	request       *http.Request
	sessions      readServicesSessionLifecycle
	sessionCookie djangoSessionCookieConfig
	access        *activityReadsSessionAccess
	wroteHeader   bool
	failed        bool
}

func (writer *activityReadsSessionFinalizationResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	if !writer.access.accessed {
		if err := finalizeUnaccessedDjangoSession(writer.Header(), writer.request, status, writer.sessionCookie, writer.sessions); err != nil {
			logPublicProjectMetadataCompatibilityError(writer.request.Context(), err)
			writer.failed = true
			writeReadServicesSessionError(writer.ResponseWriter, writer.request, err)
			return
		}
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *activityReadsSessionFinalizationResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.failed {
		return len(body), nil
	}
	return writer.ResponseWriter.Write(body)
}

func activityReadsSessionFinalizationMiddleware(sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if authResponseSessionPath(request) {
				next.ServeHTTP(response, request)
				return
			}
			activityRoute := isActivityReadPath(request.URL.Path)
			if protocol, matched := resolveReadServiceProtocolForMethod(strings.TrimSuffix(request.URL.Path, "/")+"/", request.Method); matched && protocol.authenticateMethod {
				activityRoute = true
			}
			preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
			if !activityRoute || preflight || sessions == nil {
				next.ServeHTTP(response, request)
				return
			}
			access := &activityReadsSessionAccess{}
			request = normalizeDjangoCookie(request, sessionCookieName(cookie))
			request = request.WithContext(context.WithValue(request.Context(), activityReadsSessionAccessKey{}, access))
			next.ServeHTTP(&activityReadsSessionFinalizationResponseWriter{
				ResponseWriter: response,
				request:        request,
				sessions:       sessions,
				sessionCookie:  cookie,
				access:         access,
			}, request)
		})
	}
}

func isActivityReadPath(path string) bool {
	protocol, matched := resolveReadServiceProtocol(path)
	if !matched && !strings.HasSuffix(path, "/") {
		protocol, matched = resolveReadServiceProtocol(path + "/")
	}
	if !matched {
		// Generated API-v1 lifecycle actions do not have a retained read
		// protocol entry, but their slashless requests still use the shared
		// Django append-slash behavior.
		protocol, matched = moduleLifecycleProtocolForPath(path)
		if !matched {
			protocol, matched = projectLifecycleProtocolForPath(path)
		}
	}
	if !matched || (!activityReadsAPIProtocol(protocol.pattern) && protocol.pattern != "/api/users/me/activities/") {
		return false
	}
	return true
}

func activityReadsSlashlessProtocol(path string) bool {
	if strings.HasSuffix(path, "/") {
		return false
	}
	return isActivityReadPath(path + "/")
}

func activityReadsSlashRedirectMiddleware(next http.Handler, debugMode bool) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if _, _, stickyFormatPath := apiV1StickyFormatSuffix(request.URL.Path); stickyFormatPath {
			next.ServeHTTP(response, request)
			return
		}
		if strings.HasSuffix(request.URL.Path, "/") {
			next.ServeHTTP(response, request)
			return
		}
		if !isActivityReadPath(request.URL.Path) {
			next.ServeHTTP(response, request)
			return
		}
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		mergeVary(response.Header(), "Cookie", "origin")
		switch request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if debugMode {
				body, contentType := djangoDebugAppendSlashResponse(request)
				response.Header().Set("Content-Type", contentType)
				response.WriteHeader(http.StatusInternalServerError)
				_, _ = response.Write(body)
				return
			}
		}
		location := request.URL.Path + "/"
		if request.URL.RawQuery != "" {
			location += "?" + request.URL.RawQuery
		}
		response.Header().Set("Location", location)
		response.WriteHeader(http.StatusMovedPermanently)
	})
}

func activityReadsSlashlessCompatibilityHandler(stack func(http.Handler) http.Handler, redirect http.Handler) http.Handler {
	innerNotFound := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Content-Length", strconv.Itoa(len(djangoNotFoundBody)))
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write(djangoNotFoundBody)
	})
	inner := stack(innerNotFound)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		captured := newAPIV1BufferedResponse()
		inner.ServeHTTP(captured, request)
		if captured.status == 0 {
			captured.status = http.StatusOK
		}
		if captured.status == http.StatusNotFound {
			if values := captured.header.Values("Set-Cookie"); len(values) != 0 {
				response.Header()["Set-Cookie"] = append([]string(nil), values...)
			}
			redirect.ServeHTTP(response, request)
			return
		}
		for name, values := range captured.header {
			response.Header()[name] = append([]string(nil), values...)
		}
		response.WriteHeader(captured.status)
		if request.Method != http.MethodHead {
			_, _ = response.Write(captured.body.Bytes())
		}
	})
}

func activityReadsMiddlewareStack(
	options apiV1CompatibilityOptions,
	sessions readServicesSessionLifecycle,
	cookie djangoSessionCookieConfig,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		next = readServicesQueryPresenceMiddleware(next)
		next = activityReadsCompatibilityMiddleware(options)(next)
		next = activityReadsSessionMiddleware(sessions, cookie)(next)
		next = readServicesContentNegotiationMiddleware(next)
		next = activityReadsAuditFallbackMiddleware(options)(next)
		next = activityReadsRequestSizeMiddleware(options.bodySizeLimit)(next)
		next = readServicesGzipMiddleware(next)
		next = activityReadsIntegerQueryMiddleware(next)
		return next
	}
}

func activityReadsIntegerQueryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		original := request.URL.Query()
		request = request.WithContext(systemapi.WithReadServiceQueryPresence(request.Context(), original))
		rawValues, present := original["per_page"]
		if !present || len(rawValues) == 0 {
			next.ServeHTTP(response, request)
			return
		}
		normalized := "1"
		if value, err := pythoncompat.DecimalInteger(rawValues[len(rawValues)-1]); err == nil && value.IsInt64() {
			normalized = value.String()
		}
		query := request.URL.Query()
		query.Set("per_page", normalized)
		clone := request.Clone(request.Context())
		clonedURL := *request.URL
		clonedURL.RawQuery = query.Encode()
		clone.URL = &clonedURL
		next.ServeHTTP(response, clone)
	})
}

func activityReadsSessionMiddleware(sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		session := readServicesSessionMiddleware(sessions, cookie)(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/v1/") {
				next.ServeHTTP(response, request)
				return
			}
			if access, ok := request.Context().Value(activityReadsSessionAccessKey{}).(*activityReadsSessionAccess); ok {
				access.accessed = true
			}
			session.ServeHTTP(response, request)
		})
	}
}

func activityReadsRequestSizeMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			_, tooLarge, err := readPublicProjectMetadataRequestBody(request, limit)
			if tooLarge {
				// Django rejects the body before entering the view or accessing
				// the session. Remove headers installed by the outer dispatcher.
				response.Header().Del("Allow")
				removeVary(response.Header(), "Cookie")
				response.Header().Set("Content-Type", "application/json")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
				response.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = response.Write(djangoRequestTooLargeBody)
				return
			}
			if err != nil {
				response.Header().Set("Content-Type", "text/html; charset=utf-8")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				if request.Method != http.MethodHead {
					_, _ = response.Write(djangoServerErrorBody)
				}
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func activityReadsAuditFallbackMiddleware(options apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if djangoJoinedHeader(request, "X-Api-Key") == "" {
				next.ServeHTTP(response, request)
				return
			}
			requestBody, _, err := readPublicProjectMetadataRequestBody(request, options.bodySizeLimit)
			if err != nil {
				next.ServeHTTP(response, request)
				return
			}
			state := &activityReadsAuditState{}
			request = request.WithContext(context.WithValue(request.Context(), activityReadsAuditStateKey{}, state))
			captured := newAPIV1BufferedResponse()
			next.ServeHTTP(captured, request)
			if captured.status == 0 {
				captured.status = http.StatusOK
			}
			if !state.logged {
				writeAPIActivityLog(request.Context(), options.activityLogger, options.activitySecret, request, requestBody, captured.status, captured.body.Bytes())
			}
			for name, values := range captured.header {
				response.Header()[name] = append([]string(nil), values...)
			}
			response.WriteHeader(captured.status)
			if request.Method != http.MethodHead {
				_, _ = response.Write(captured.body.Bytes())
			}
		})
	}
}

func activityReadsCompatibilityMiddleware(options apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		apiV1 := apiV1CompatibilityMiddleware(options)(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/v1/") {
				apiV1.ServeHTTP(response, request)
				return
			}
			requestBody, tooLarge, err := readPublicProjectMetadataRequestBody(request, options.bodySizeLimit)
			if tooLarge {
				response.Header().Set("Content-Type", "application/json")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
				response.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = response.Write(djangoRequestTooLargeBody)
				return
			}
			if err != nil {
				response.Header().Set("Content-Type", "text/html; charset=utf-8")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				if request.Method != http.MethodHead {
					_, _ = response.Write(djangoServerErrorBody)
				}
				return
			}
			if djangoJoinedHeader(request, "X-Api-Key") == "" {
				next.ServeHTTP(response, request)
				return
			}
			captured := newAPIV1BufferedResponse()
			next.ServeHTTP(captured, request)
			if captured.status == 0 {
				captured.status = http.StatusOK
			}
			writeAPIActivityLog(request.Context(), options.activityLogger, options.activitySecret, request, requestBody, captured.status, captured.body.Bytes())
			if state, ok := request.Context().Value(activityReadsAuditStateKey{}).(*activityReadsAuditState); ok {
				state.logged = true
			}
			for name, values := range captured.header {
				response.Header()[name] = append([]string(nil), values...)
			}
			response.WriteHeader(captured.status)
			if request.Method != http.MethodHead {
				_, _ = response.Write(captured.body.Bytes())
			}
		})
	}
}

func mountActivityReadsRoutes(dbg bool, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) (*apiV1CompatibilityOptions, func(http.Handler) http.Handler) {
	var activityOptions *apiV1CompatibilityOptions
	var activityProtocolMiddleware func(http.Handler) http.Handler
	if pending.activityReads != nil {
		activityOptions = &apiV1CompatibilityOptions{
			activityLogger: pending.activityReadsActivity,
			activitySecret: pending.activityReadsSecret,
			bodySizeLimit:  pending.activityReadsBodyLimit,
		}
		server := activityreadssvr.New(pending.activityReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		stack := activityReadsMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
		server.Use(stack)
		activityreadssvr.Mount(mux, server)
		mountActivityReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
		mountAPIV1StickiesRoutes(pending, mux, readServicesCORSPolicy, activityOptions)
		mountAPIUserIdentityRoutes(pending, mux, readServicesCORSPolicy, stack)

		mountAPICycleReadsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountAPIModuleReadsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountAPILabelReadsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountMembershipLifecycleRoutes(pending, mux, readServicesCORSPolicy, activityOptions, stack)
		mountAPIStateReadsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountWorkItemCoreRoutes(pending, mux, readServicesCORSPolicy, activityOptions)
		mountTicketCollaborationRoutes(pending, mux, readServicesCORSPolicy, activityOptions)
		activityProtocolMiddleware = newActivityReadsProtocolMiddleware(membershipActivityProtocolDispatch{configuration: pending.projectConfigurationMetadata, ActivityReadsProtocolAuthorizer: pending.activityReadsMetadata, membership: pending.membershipLifecycleMetadata, assets: pending.assetsAPIMetadata, invitations: pending.invitationsMetadata, projects: pending.projectLifecycleMetadata, stickies: pending.apiV1StickiesProtocol}, readServicesCORSPolicy, stack, *activityOptions, pending.readServicesCookie, pending.activityReadsBodyLimit, dbg)
	}
	return activityOptions, activityProtocolMiddleware
}
