package main

import (
	"net/http"

	useraccountwritessvr "github.com/CaliLuke/go-plane/user_account_writes/gen/http/user_account_writes/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountUserAccountWritesRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.userAccountWrites != nil {
		server := useraccountwritessvr.New(pending.userAccountWrites, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		useraccountwritessvr.Mount(mux, server)
	}
}
