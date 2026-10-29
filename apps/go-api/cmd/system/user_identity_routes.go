package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	systemapi "github.com/CaliLuke/go-plane"
	apiuseridentitysvr "github.com/CaliLuke/go-plane/apiuseridentity/gen/http/api_user_identity/server"
	useridentitysvr "github.com/CaliLuke/go-plane/gen/http/user_identity/server"
	userinstanceadminsvr "github.com/CaliLuke/go-plane/gen/http/user_instance_admin/server"
	userinstanceadmin "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	loomhttp "github.com/CaliLuke/loom/http"
)

func userIdentityCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(path string) string {
		if path == "/api/users/me/" {
			return "GET, PATCH, DELETE, HEAD, OPTIONS"
		}
		if path == "/api/users/me/profile/" {
			return "GET, PATCH, HEAD, OPTIONS"
		}
		return "GET, HEAD, OPTIONS"
	})(next)
}

type userMeCacheControlResponseWriter struct {
	http.ResponseWriter
}

func (writer userMeCacheControlResponseWriter) WriteHeader(status int) {
	if status == http.StatusOK {
		writer.Header().Set("Cache-Control", "private, max-age=12")
		writer.Header().Set("Vary", "Cookie, Accept-Encoding, origin")
	}
	if status == http.StatusUnauthorized {
		writer.Header().Set("Vary", "Cookie, origin")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func userMeCacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(userMeCacheControlResponseWriter{ResponseWriter: response}, request)
	})
}

func userMeReadCompatibilityMiddleware(
	sessions userIdentitySessionLifecycle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	bodySizeLimit int64,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			headers.Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Referrer-Policy", "same-origin")
			mergeVary(headers, "origin")
			publicProjectMetadataReadsCORSHeaders(headers, request, allowedCORSOrigins)
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			requestBody, tooLarge, bodyErr := readPublicProjectMetadataRequestBody(request, bodySizeLimit)
			if tooLarge {
				headers.Del("Allow")
				if sessions != nil {
					writePublicProjectMetadataRequestTooLarge(response, request, sessionCookie, sessions)
				} else {
					headers.Set("Content-Type", "application/json")
					headers.Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
					response.WriteHeader(http.StatusRequestEntityTooLarge)
					_, _ = response.Write(djangoRequestTooLargeBody)
				}
				return
			}
			if bodyErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), bodyErr)
				headers.Del("Allow")
				if sessions != nil {
					if err := finalizeUnaccessedDjangoSession(headers, request, http.StatusInternalServerError, sessionCookie, sessions); err != nil {
						writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
						replacePublicProjectMetadataSessionError(writer, err)
						writer.finish()
						return
					}
				}
				headers.Set("Content-Type", "text/html; charset=utf-8")
				headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				_, _ = response.Write(djangoServerErrorBody)
				return
			}

			writer := &publicProjectMetadataReadsResponseWriter{
				ResponseWriter: response,
				acceptsGzip:    acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")),
				appendGzipVary: true,
				skipCookieVary: true,
			}
			capture := &systemapi.UserIdentityResponseCapture{}
			writer.capturedResponse = func() json.RawMessage { return capture.Data }
			var preparedSessionRefresh *djangosession.SessionRefresh
			var sessionCookieErr error
			var userID string
			sessionInspected := false
			inspectSessionAfterResponse := false
			applySessionEffects := func(expireSession bool, refreshSession bool, rotatedSession *djangosession.SessionRefresh, resolvedUserID string) {
				if resolvedUserID != "" {
					userID = resolvedUserID
				}
				if expireSession {
					sessionCookieErr = expireDjangoSessionCookie(headers, sessionCookie)
				}
				if rotatedSession != nil {
					preparedSessionRefresh = rotatedSession
				} else if refreshSession {
					if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
						preparedSessionRefresh = &djangosession.SessionRefresh{SessionKey: cookie.Value}
					}
				}
			}
			defer func() {
				if inspectSessionAfterResponse && sessions != nil {
					writer.skipCookieVary = false
					mergeVary(headers, "Cookie")
					expireSession, refreshSession, rotatedSession, resolvedUserID, err := sessions.InspectSession(request.Context(), request)
					applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
					if err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						writePublicProjectMetadataInternalError(writer, err, false)
					}
				}
				writeAPIActivityLog(
					request.Context(), activityLogger, activitySecret, request, requestBody,
					writer.status, writer.renderedBody(),
				)
				if !sessionInspected && !inspectSessionAfterResponse && sessions != nil {
					if err := finalizeUnaccessedDjangoSession(headers, request, writer.status, sessionCookie, sessions); err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						replacePublicProjectMetadataSessionError(writer, err)
					}
				} else if sessionCookieErr != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), sessionCookieErr)
					replacePublicProjectMetadataSessionError(writer, sessionCookieErr)
				} else if sessions != nil && preparedSessionRefresh != nil && writer.status < http.StatusInternalServerError {
					refresh, err := sessions.RefreshSession(request.Context(), request, *preparedSessionRefresh)
					if err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						replacePublicProjectMetadataSessionError(writer, err)
					} else if err := refreshDjangoSessionCookie(headers, sessionCookie, refresh); err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						replacePublicProjectMetadataSessionError(writer, err)
					}
				}
				writer.finish()
			}()

			formats, queryErr := djangoQueryValues(readServicesOriginalRawQuery(request), "format")
			if queryErr != nil {
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if len(formats) > 0 {
				format := formats[len(formats)-1]
				if format != "" && format != "json" {
					inspectSessionAfterResponse = true
					writeStickyJSON(writer, http.StatusNotFound, map[string]string{"detail": "Not found."})
					return
				}
			}
			accept, acceptPresent := djangoRequestAcceptHeader(request)
			acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
			if negotiationErr != nil {
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if !acceptsJSON {
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusNotAcceptable, map[string]string{
					"detail": "Could not satisfy the request Accept header.",
				})
				return
			}
			writer.jsonIndent = jsonIndent
			writer.skipCookieVary = false
			mergeVary(headers, "Cookie")
			if sessions != nil {
				sessionInspected = true
				expireSession, refreshSession, rotatedSession, resolvedUserID, err := sessions.InspectSessionWithTimezone(request.Context(), request)
				applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
				if err != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					writePublicProjectMetadataInternalError(writer, err, true)
					if writer.Header().Get("Allow") != "" {
						writer.Header().Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
					}
					return
				}
				resolvedContext, err := systemapi.WithUserIdentitySessionInspection(request.Context(), userID)
				if err != nil {
					writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
						"error": "Something went wrong please try again later",
					})
					return
				}
				request = request.WithContext(resolvedContext)
			}
			request = request.WithContext(systemapi.WithUserIdentityResponseCapture(request.Context(), capture))
			next.ServeHTTP(writer, request)
		})
	}
}

func userIdentityOptionsHandler(name string, patch bool) http.Handler {
	return djangoProtocolOptionsHandler(userIdentityCompatibilityMiddleware, name, patch)
}

func mountUserInstanceAdminRoutes(ctx context.Context, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, mux loomhttp.Muxer) {
	if userInstanceAdminEndpoints != nil {
		userInstanceAdminServer := userinstanceadminsvr.New(userInstanceAdminEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		userInstanceAdminServer.Use(djangoJSONCompatibilityMiddleware)
		userinstanceadminsvr.Mount(mux, userInstanceAdminServer)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/instance-admin/", djangoJSONHeadHandler(userInstanceAdminServer.Retrieve))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/instance-admin/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range userInstanceAdminServer.Mounts {
			logHTTPRouteMount(ctx, "user_instance_admin", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}

func mountUserIdentityRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.userIdentity != nil {
		bodySizeLimit := pending.userIdentityBodyLimit
		if !pending.userIdentityBodySet {
			bodySizeLimit = defaultFileSizeLimit
		}
		server := useridentitysvr.New(pending.userIdentity, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userIdentityCompatibilityMiddleware)
		server.Me = userMeCacheControlMiddleware(server.Me)
		server.Me = userMeReadCompatibilityMiddleware(
			pending.userIdentitySession,
			pending.userIdentityCORS,
			pending.userIdentityCookie,
			bodySizeLimit,
			pending.userIdentityActivity,
			pending.userIdentitySecret,
		)(server.Me)
		useridentitysvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/users/session/", djangoProtocolHeadHandler(userIdentityCompatibilityMiddleware, server.Session))
		loomhttp.MountDerivedHead(mux, "/api/users/me/", userIdentityCompatibilityMiddleware(loomhttp.AsHandlerFunc(server.Me)))
		mountProtocolHandler(mux, http.MethodHead, "/api/users/me/profile/", djangoProtocolHeadHandler(userIdentityCompatibilityMiddleware, server.Profile))
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/session/", userIdentityOptionsHandler("User Session Endpoint", false))
		if pending.userAccountWrites == nil {
			//loom:vet ignore route-outside-design -- Django OPTIONS metadata stays on the identity service only when the account-write service is absent.
			loomhttp.MountHandler(mux, http.MethodOptions, "/api/users/me/", djangoProtocolOptionsWithoutActionsHandler(userIdentityCompatibilityMiddleware, "User Endpoint"))
		}
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/me/profile/", userIdentityOptionsHandler("Profile Endpoint", true))
	}
}

func mountAPIUserIdentityRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.apiUserIdentity != nil {
		identity := apiuseridentitysvr.New(pending.apiUserIdentity, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		identity.Use(stack)
		apiuseridentitysvr.MountGetAPIV1UsersMeHandler(mux, identity.GetAPIV1UsersMe, readServicesCORSPolicy)
	}
}
