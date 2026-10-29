package main

import (
	"net/http"

	notificationactionssvr "github.com/CaliLuke/go-plane/notificationactions/gen/http/notification_actions/server"
	notificationssvr "github.com/CaliLuke/go-plane/notificationreads/gen/http/notifications/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountNotificationsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.notifications != nil {
		server := notificationssvr.New(pending.notifications, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		notificationssvr.Mount(mux, server)
		mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
			{"/api/workspaces/{slug}/users/notifications/", server.List},
			{"/api/workspaces/{slug}/users/notifications/unread/", server.Counts},
			{"/api/workspaces/{slug}/users/notifications/{pk}/", server.Get},
		}, readServicesCORSPolicy)

	}
}

func mountNotificationActionsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.notificationActions != nil {
		server := notificationactionssvr.New(pending.notificationActions, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		notificationactionssvr.MountMarkAllReadHandler(mux, server.MarkAllRead, readServicesCORSPolicy)
		notificationactionssvr.MountUpdateHandler(mux, server.Update, readServicesCORSPolicy)
		notificationactionssvr.MountDeleteHandler(mux, server.Delete, readServicesCORSPolicy)
		notificationactionssvr.MountUnarchiveHandler(mux, server.Unarchive, readServicesCORSPolicy)
		notificationactionssvr.MountArchiveHandler(mux, server.Archive, readServicesCORSPolicy)
		notificationactionssvr.MountMarkUnreadHandler(mux, server.MarkUnread, readServicesCORSPolicy)
		notificationactionssvr.MountMarkReadHandler(mux, server.MarkRead, readServicesCORSPolicy)
	}
}
