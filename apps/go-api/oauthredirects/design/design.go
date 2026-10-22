// Package design defines the public OAuth redirect endpoints.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/oauthredirects/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var redirectResponse = Type("OAuthRedirectResponse", func() {
	Attribute("content_type", String)
	Attribute("location", String)
	Required("content_type", "location")
})

func oauthRedirectMethod(name, operationID, route string) {
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		Payload(func() {
			Attribute("code", String)
			Attribute("state", String)
			Attribute("next_path", String)
		})
		Result(redirectResponse)
		HTTP(func() {
			GET(route)
			Param("code")
			Param("state")
			Param("next_path")
			SkipResponseBodyEncodeDecode()
			Response(302, func() {
				Description("Redirect to the OAuth provider or the configured application host.")
				Header("content_type:Content-Type")
				Header("location:Location")
			})
		})
	})
}

var _ = API("Plane OAuth Redirects", func() {
	Title("Plane OAuth Redirects")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Produces("text/html")
	})
})

var _ = Service("oauth_redirects", func() {
	Description("Initiates OAuth authorization and processes provider callbacks.")
	HTTP(func() { RuntimeCORS() })
	oauthRedirectMethod("GoogleInitiate", "get_auth_google", "/auth/google/")
	oauthRedirectMethod("GoogleCallback", "get_auth_google_callback", "/auth/google/callback/")
	oauthRedirectMethod("GoogleSpaceInitiate", "get_auth_spaces_google", "/auth/spaces/google/")
	oauthRedirectMethod("GoogleSpaceCallback", "get_auth_spaces_google_callback", "/auth/spaces/google/callback/")
	oauthRedirectMethod("GitHubInitiate", "get_auth_github", "/auth/github/")
	oauthRedirectMethod("GitHubCallback", "get_auth_github_callback", "/auth/github/callback/")
	oauthRedirectMethod("GitHubSpaceInitiate", "get_auth_spaces_github", "/auth/spaces/github/")
	oauthRedirectMethod("GitHubSpaceCallback", "get_auth_spaces_github_callback", "/auth/spaces/github/callback/")
	oauthRedirectMethod("GitLabInitiate", "get_auth_gitlab", "/auth/gitlab/")
	oauthRedirectMethod("GitLabCallback", "get_auth_gitlab_callback", "/auth/gitlab/callback/")
	oauthRedirectMethod("GitLabSpaceInitiate", "get_auth_spaces_gitlab", "/auth/spaces/gitlab/")
	oauthRedirectMethod("GitLabSpaceCallback", "get_auth_spaces_gitlab_callback", "/auth/spaces/gitlab/callback/")
	oauthRedirectMethod("GiteaInitiate", "get_auth_gitea", "/auth/gitea/")
	oauthRedirectMethod("GiteaCallback", "get_auth_gitea_callback", "/auth/gitea/callback/")
	oauthRedirectMethod("GiteaSpaceInitiate", "get_auth_spaces_gitea", "/auth/spaces/gitea/")
	oauthRedirectMethod("GiteaSpaceCallback", "get_auth_spaces_gitea_callback", "/auth/spaces/gitea/callback/")
})
