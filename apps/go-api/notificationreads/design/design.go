// Package design defines the reverse-imported notification contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/notificationreads/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var Notification = Type("Notification", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Attribute("data", Any, func() {
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("entity_identifier", String, func() {
		Nullable()
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Attribute("entity_name", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_inbox_issue", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_intake_issue", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_mentioned_notification", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("message", Any, func() {
	})
	Attribute("message_html", String)
	Attribute("message_stripped", String, func() {
		Nullable()
	})
	Attribute("project", String, func() {
		Nullable()
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Attribute("read_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("receiver", String, func() {
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Attribute("sender", String, func() {
		MaxLength(255)
	})
	Attribute("snoozed_till", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("title", String)
	Attribute("triggered_by", String, func() {
		Nullable()
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Attribute("triggered_by_details", NotificationActor, func() {
		Nullable()
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Attribute("workspace", String, func() {
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Required("id", "created_at", "updated_at", "created_by", "updated_by", "deleted_at", "workspace", "project", "data", "entity_identifier", "entity_name", "title", "message", "message_html", "message_stripped", "sender", "triggered_by", "receiver", "read_at", "snoozed_till", "archived_at", "triggered_by_details")
})

var NotificationActor = Type("NotificationActor", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("avatar", String)
	Attribute("avatar_url", String, func() {
		// Django returns a stored avatar string or a relative asset path.
		Meta("loom:vet:ignore", "string-format")
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		MaxLength(255)
	})
	Attribute("first_name", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_bot", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_name", String, func() {
		MaxLength(255)
	})
	Required("id", "first_name", "last_name", "avatar", "is_bot", "display_name")
})

var UnreadCounts = Type("UnreadCounts", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("mention_unread_notifications_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_unread_notifications_count", Int, func() {
		Meta("openapi:format", "")
	})
	Required("mention_unread_notifications_count", "total_unread_notifications_count")
})

var NotificationArray = Type("NotificationArray", ArrayOf(Notification))
var PaginatedNotifications = Type("PaginatedNotifications", func() {
	Attribute("count", Int)
	Attribute("next_cursor", String)
	Attribute("prev_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_page_results", Boolean)
	Attribute("total_count", Int)
	Attribute("total_results", Int)
	Attribute("total_pages", Int)
	Attribute("extra_stats", Any)
	Attribute("grouped_by", Any)
	Attribute("sub_grouped_by", Any)
	Attribute("results", NotificationArray)
	Required("count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results", "total_count", "total_results", "total_pages", "extra_stats", "grouped_by", "sub_grouped_by", "results")
})

var NotificationError = Type("NotificationError", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("snoozed_till", Any)
	Required("name")
})
var NotificationHTTPError = Type("NotificationHTTPError", func() {
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("snoozed_till", Any)
})

var ImportedAPIKeyAuthSecurity = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var ImportedDjangoCsrfCookieSecurity = APIKeySecurity("DjangoCsrfCookie", func() {
	Description("Django CSRF cookie for unsafe standard-session requests.")
})

var ImportedDjangoCsrfHeaderSecurity = APIKeySecurity("DjangoCsrfHeader", func() {
	Description("Django CSRF header for unsafe standard-session requests.")
})

var ImportedDjangoSessionCookieSecurity = APIKeySecurity("DjangoSessionCookie", func() {
	Description("Standard Django session authentication.")
})

var ImportedPlaneAdminSessionCookieSecurity = APIKeySecurity("PlaneAdminSessionCookie", func() {
	Description("Plane instance-administration session authentication without CSRF enforcement.")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var _ = API("Plane Internal Django Migration API", func() {
	Title("Plane Internal Django Migration API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("http:generate", "server")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:users")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("Notifications", func() {
	HTTP(func() { RuntimeCORS() })
	Method("List", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_users_notifications")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/users/notifications/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.notification.base.NotificationViewSet\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("snoozed", String)
			Attribute("archived", String)
			Attribute("read", String)
			Attribute("type", String)
			Attribute("mentioned", String)
			Attribute("per_page", String)
			Attribute("cursor", String)
			Attribute("order_by", String)
			Attribute("slug", String)
			Required("slug")
		})
		Result(OneOf(NotificationArray, PaginatedNotifications), func() {
			Untagged()
		})
		Error("status_400", NotificationError)
		Error("status_401", NotificationError)
		Error("status_403", NotificationError)
		Error("status_404", NotificationError)
		Error("status_405", NotificationError)
		Error("status_406", NotificationError)
		Error("status_415", NotificationError)
		Error("status_500", NotificationError, func() { Fault() })
		HTTP(func() {
			Response("status_400", 400, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_401", 401, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_403", 403, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_404", 404, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_405", 405, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_406", 406, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_415", 415, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_500", 500, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Meta("openapi:tag:users")
			GET("/api/workspaces/{slug}/users/notifications/")
			Param("search")
			Param("snoozed")
			Param("archived")
			Param("read")
			Param("type")
			Param("mentioned")
			Param("per_page")
			Param("cursor")
			Param("order_by")

			Param("slug")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("Counts", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_users_notifications_unread")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/users/notifications/unread/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.notification.base.UnreadNotificationEndpoint\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("slug", String)
			Required("slug")
		})
		Result(UnreadCounts)
		Error("status_400", NotificationError)
		Error("status_401", NotificationError)
		Error("status_403", NotificationError)
		Error("status_404", NotificationError)
		Error("status_405", NotificationError)
		Error("status_406", NotificationError)
		Error("status_415", NotificationError)
		Error("status_500", NotificationError, func() { Fault() })
		HTTP(func() {
			Response("status_400", 400, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_401", 401, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_403", 403, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_404", 404, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_405", 405, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_406", 406, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_415", 415, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_500", 500, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Meta("openapi:tag:users")
			GET("/api/workspaces/{slug}/users/notifications/unread/")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(UnreadCounts)
			})
		})
	})
	Method("Get", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_users_notifications_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/users/notifications/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.notification.base.NotificationViewSet\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("pk", String, func() { Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$") })
			Attribute("slug", String)
			Required("pk", "slug")
		})
		Result(Notification)
		Error("status_400", NotificationError)
		Error("status_401", NotificationError)
		Error("status_403", NotificationError)
		Error("status_404", NotificationError)
		Error("status_405", NotificationError)
		Error("status_406", NotificationError)
		Error("status_415", NotificationError)
		Error("status_500", NotificationError, func() { Fault() })
		HTTP(func() {
			Response("status_400", 400, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_401", 401, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_403", 403, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_404", 404, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_405", 405, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_406", 406, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_415", 415, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Response("status_500", 500, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(NotificationHTTPError) })
			Meta("openapi:tag:users")
			GET("/api/workspaces/{slug}/users/notifications/{pk}/")
			Param("pk")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(Notification)
			})
		})
	})
})

var ImportedPlaneSession = SessionAuth("PlaneSession", func() {
	CookieTransport(ImportedPlaneSessionCookieSecurity, "", func() { CookieName("session-id") })
})
