package mcpoauth_test

import (
	"errors"
	"testing"

	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
)

func TestProviderConfigurationRequiresSeparateClientsAndSecureEndpoints(t *testing.T) {
	base := oauth.Config{Issuer: "https://issuer.example/tenant", Resource: "https://vector.example/mcp", IntrospectionURL: "https://issuer.example/introspect", IntrospectionClientID: "introspect", IntrospectionClientSecret: "private", LinkClientID: "link", LinkClientSecret: "private-link"}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*oauth.Config)
	}{
		{"plain HTTP", func(c *oauth.Config) { c.Issuer = "http://issuer.example" }},
		{"URL credential", func(c *oauth.Config) { c.IntrospectionURL = "https://user:private@issuer.example/introspect" }},
		{"URL query", func(c *oauth.Config) { c.IntrospectionURL += "?secret=private" }},
		{"URL fragment", func(c *oauth.Config) { c.Issuer += "#fragment" }},
		{"wrong resource", func(c *oauth.Config) { c.Resource = "https://vector.example/" }},
		{"same client", func(c *oauth.Config) { c.LinkClientID = c.IntrospectionClientID }},
		{"missing introspection secret", func(c *oauth.Config) { c.IntrospectionClientSecret = "" }},
		{"missing link secret", func(c *oauth.Config) { c.LinkClientSecret = " " }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := base
			test.change(&config)
			if err := config.Validate(); !errors.Is(err, oauth.ErrInvalidConfiguration) {
				t.Fatalf("configuration error=%v", err)
			}
		})
	}
}
