package main

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type sessionPrincipalInspector interface {
	InspectReadServiceSessionWithoutTimezone(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error)
}

type sessionServiceLifecycle struct{ readServicesSessionLifecycle }

func sessionServicePrincipalLifecycle(sessions readServicesSessionLifecycle) readServicesSessionLifecycle {
	if sessions == nil {
		return nil
	}
	return sessionServiceLifecycle{sessions}
}

func (s sessionServiceLifecycle) InspectReadServiceSession(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	protocol, _ := resolveReadServiceProtocol(request.URL.Path)
	permissionBeforeTimezone := protocol.permissionBeforeTimezone && !slices.Contains(protocol.permissionBeforeTimezoneExceptMethods, request.Method)
	if permissionBeforeTimezone || protocol.throttleBeforeTimezone {
		if inspector, ok := s.readServicesSessionLifecycle.(sessionPrincipalInspector); ok {
			return inspector.InspectReadServiceSessionWithoutTimezone(ctx, request)
		}
		return false, false, nil, djangosession.Principal{}, errors.New("session principal inspection without timezone is unavailable")
	}
	return s.readServicesSessionLifecycle.InspectReadServiceSession(ctx, request)
}
