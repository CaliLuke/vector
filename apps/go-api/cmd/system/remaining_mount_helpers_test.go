package main

import (
	"context"
	"io"
	"net/http"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type searchMountSessions struct{ *cycleMiddlewareSessions }

func (s searchMountSessions) InspectReadServiceSessionWithoutTimezone(_ context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	if _, err := request.Cookie("session-id"); err != nil {
		return false, false, nil, djangosession.Principal{}, nil
	}
	zone := s.userTimezone
	if s.invalidZone {
		zone = "Not/AZone"
	}
	return false, false, nil, djangosession.Principal{UserID: cycleMiddlewareActor, UserTimezone: zone}, nil
}

func extensionHTTPHandler(pending pendingReplacementEndpoints) http.Handler {
	return newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pending)
}
