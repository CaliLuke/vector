package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	publicfeedbacksvr "github.com/CaliLuke/go-plane/gen/http/public_feedback/server"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	loomhttp "github.com/CaliLuke/loom/http"
)

func publicFeedbackAllowMethods(request *http.Request) string {
	route, matched := classifyPublicFeedbackPath(request.URL.Path)
	if !matched {
		return "GET, HEAD, OPTIONS"
	}
	return route.allowMethods
}

var publicFeedbackPathPattern = regexp.MustCompile(`^/api/public/(?:anchor/[^/]+/(?:issues(?:/[^/]+(?:/votes)?)?|intakes/[^/]+/(?:inbox-issues|intake-issues(?:/[^/]+)?))|workspaces/[^/]+/project-boards)/?$`)

type publicFeedbackRoute struct {
	allowMethods    string
	requiresSession bool
	malformedUUID   bool
}

func classifyPublicFeedbackPath(requestPath string) (publicFeedbackRoute, bool) {
	path := strings.TrimPrefix(requestPath, "/")
	path = strings.TrimSuffix(path, "/")
	segments := strings.Split(path, "/")
	if len(segments) < 5 || segments[0] != "api" || segments[1] != "public" {
		return publicFeedbackRoute{}, false
	}
	switch {
	case len(segments) == 5 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "issues":
		return publicFeedbackRoute{allowMethods: "GET, HEAD, OPTIONS"}, true
	case len(segments) == 6 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "issues":
		return publicFeedbackRoute{allowMethods: "GET, HEAD, OPTIONS", malformedUUID: !djangoUUIDPathSegment(segments[5])}, true
	case len(segments) == 7 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "issues" && segments[6] == "votes":
		return publicFeedbackRoute{allowMethods: "GET, POST, DELETE, HEAD, OPTIONS", requiresSession: true, malformedUUID: !djangoUUIDPathSegment(segments[5])}, true
	case len(segments) == 7 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "intakes" && segments[5] != "" && (segments[6] == "inbox-issues" || segments[6] == "intake-issues"):
		return publicFeedbackRoute{allowMethods: "GET, POST, HEAD, OPTIONS", requiresSession: true, malformedUUID: !djangoUUIDPathSegment(segments[5])}, true
	case len(segments) == 8 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "intakes" && segments[6] == "intake-issues":
		return publicFeedbackRoute{allowMethods: "GET, PATCH, DELETE, HEAD, OPTIONS", requiresSession: true, malformedUUID: !djangoUUIDPathSegment(segments[5]) || !djangoUUIDPathSegment(segments[7])}, true
	case len(segments) == 5 && segments[2] == "workspaces" && segments[3] != "" && segments[4] == "project-boards":
		return publicFeedbackRoute{allowMethods: "GET, HEAD, OPTIONS"}, true
	default:
		return publicFeedbackRoute{}, false
	}
}

func publicFeedbackRouteRequiresSession(request *http.Request) bool {
	route, matched := classifyPublicFeedbackPath(request.URL.Path)
	return matched && route.requiresSession
}

func publicFeedbackRouteCompatibilityMiddleware(compatibility func(http.Handler) http.Handler, slashless http.Handler, malformed http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		methodNotAllowed := compatibility(publicFeedbackMethodNotAllowedHandler())
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			route, matched := classifyPublicFeedbackPath(request.URL.Path)
			if !matched {
				next.ServeHTTP(response, request)
				return
			}
			if route.malformedUUID {
				malformed.ServeHTTP(response, request)
				return
			}

			canonicalPath := strings.HasSuffix(request.URL.Path, "/")
			if !canonicalPath {
				slashless.ServeHTTP(response, request)
				return
			}

			method := strings.ToUpper(request.Method)
			if method != request.Method {
				request = request.Clone(request.Context())
				request.Method = method
			}
			if readServiceMethodAllowed(publicFeedbackAllowMethods(request), method) {
				next.ServeHTTP(response, request)
				return
			}
			methodNotAllowed.ServeHTTP(response, request)
		})
	}
}

func publicFeedbackMethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if publicFeedbackRouteRequiresSession(request) && publicprojectmetadatareadsstore.UserID(request.Context()) == "" {
			writeStickyJSON(response, http.StatusUnauthorized, map[string]string{
				"detail": "Authentication credentials were not provided.",
			})
			return
		}
		writeStickyJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}

func mountPublicFeedbackRoutes(dbg bool, pending pendingReplacementEndpoints, mux loomhttp.Muxer) func(http.Handler) http.Handler {
	var publicFeedbackRouteCompatibility func(http.Handler) http.Handler
	if pending.publicFeedback != nil {
		if pending.publicProjectMetadataThrottle == nil {
			panic("public feedback session throttle is required")
		}
		bodySizeLimit := pending.publicProjectMetadataBodyLimit
		if !pending.publicProjectMetadataBodySet {
			bodySizeLimit = defaultFileSizeLimit
		}
		compatibilityMiddleware := publicProjectMetadataReadsCompatibilityMiddleware(
			pending.publicProjectMetadataThrottle,
			pending.publicProjectMetadataCORS,
			pending.publicProjectMetadataCookie,
			pending.publicProjectMetadataActivity,
			pending.publicProjectMetadataSecret,
			bodySizeLimit,
			true,
			true,
			publicProjectMetadataRoutePolicy{
				allowMethods:    publicFeedbackAllowMethods,
				requiresSession: publicFeedbackRouteRequiresSession,
			},
		)
		publicFeedbackRouteCompatibility = publicFeedbackRouteCompatibilityMiddleware(
			compatibilityMiddleware,
			publicProjectMetadataNotFoundHandler(
				pending.publicProjectMetadataThrottle,
				pending.publicProjectMetadataCORS,
				pending.publicProjectMetadataCookie,
				pending.publicProjectMetadataActivity,
				pending.publicProjectMetadataSecret,
				bodySizeLimit,
				dbg,
				true,
			),
			publicProjectMetadataNotFoundHandler(
				pending.publicProjectMetadataThrottle,
				pending.publicProjectMetadataCORS,
				pending.publicProjectMetadataCookie,
				pending.publicProjectMetadataActivity,
				pending.publicProjectMetadataSecret,
				bodySizeLimit,
				dbg,
				false,
			),
		)
		server := publicfeedbacksvr.New(pending.publicFeedback, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(compatibilityMiddleware)
		publicfeedbacksvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/issues/", server.IssueList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/issues/{issue_id}/", server.IssueDetail)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/issues/{issue_id}/votes/", server.VoteList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/intakes/{intake_id}/inbox-issues/", server.InboxList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/", server.IntakeList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/", server.IntakeDetail)
		loomhttp.MountDerivedHead(mux, "/api/public/workspaces/{slug}/project-boards/", server.WorkspaceProjectBoards)
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/issues/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/issues/", false))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/issues/{issue_id}/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/issues/{issue_id}/", false))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/issues/{issue_id}/votes/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/issues/{issue_id}/votes/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/intakes/{intake_id}/inbox-issues/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/intakes/{intake_id}/inbox-issues/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/workspaces/{slug}/project-boards/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/workspaces/{slug}/project-boards/", false))
	}
	return publicFeedbackRouteCompatibility
}
