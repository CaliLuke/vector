package main

import (
	"net/http"

	preferencessvr "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/http/project_member_preferences/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountProjectMemberPreferencesRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.projectMemberPreferences != nil {
		server := preferencessvr.New(pending.projectMemberPreferences, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		preferencessvr.Mount(mux, server)
	}
}
