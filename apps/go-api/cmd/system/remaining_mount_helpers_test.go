package main

import (
	"context"
	"io"
	"net/http"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
)

type searchMountSessions struct{ *cycleMiddlewareSessions }

func (s searchMountSessions) InspectReadServiceSessionWithoutTimezone(_ context.Context, request *http.Request) (bool, bool, *browsersession.SessionRefresh, browsersession.Principal, error) {
	if len(request.CookiesNamed("session-id")) == 0 {
		return false, false, nil, browsersession.Principal{}, nil
	}
	zone := s.userTimezone
	if s.invalidZone {
		zone = "Not/AZone"
	}
	return false, false, nil, browsersession.Principal{UserID: cycleMiddlewareActor, UserTimezone: zone}, nil
}

func extensionHTTPHandler(pending pendingReplacementEndpoints) http.Handler {
	return newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pending)
}
