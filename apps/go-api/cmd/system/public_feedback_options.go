package main

import (
	_ "embed"
	"encoding/json"
	"net/http"

	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
)

//go:embed public_feedback_options_metadata.json
var publicFeedbackOptionsMetadataJSON []byte

var publicFeedbackOptionsMetadata = func() map[string]json.RawMessage {
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(publicFeedbackOptionsMetadataJSON, &metadata); err != nil {
		panic(err)
	}
	return metadata
}()

func publicFeedbackOptionsHandler(middleware func(http.Handler) http.Handler, route string, requiresSession bool) http.Handler {
	metadata, ok := publicFeedbackOptionsMetadata[route]
	if !ok {
		panic("public feedback OPTIONS metadata is missing for " + route)
	}
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if requiresSession && publicprojectmetadatareadsstore.UserID(request.Context()) == "" {
			writeStickyJSON(response, http.StatusUnauthorized, map[string]string{"detail": "Authentication credentials were not provided."})
			return
		}
		writeStickyJSON(response, http.StatusOK, metadata)
	}))
}
