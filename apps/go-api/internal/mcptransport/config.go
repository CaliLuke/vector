// Package mcptransport mounts Vector's generated remote MCP server.
package mcptransport

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	OAuthPath  = "/mcp"
	HeaderPath = "/mcp/api-key"
)

// Config is deliberately disabled by default. Authentication implementations
// must also be supplied before a selected route can be mounted.
type Config struct {
	Enabled         bool
	Mode            string
	ResourceURL     string
	TrustedOrigins  []string
	MaxRequestBytes int64
	RequestTimeout  time.Duration
}

func DefaultConfig() Config {
	return Config{MaxRequestBytes: 1 << 20, RequestTimeout: 30 * time.Second}
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Mode != "header" && c.Mode != "oauth" && c.Mode != "both" {
		return fmt.Errorf("MCP auth mode must be header, oauth or both")
	}
	u, err := url.Parse(c.ResourceURL)
	if err != nil || !validHTTPSOrigin(u) || u.Path != OAuthPath || u.RawPath != "" {
		return fmt.Errorf("MCP resource URL must be a canonical HTTPS URL ending in /mcp")
	}
	for _, origin := range c.TrustedOrigins {
		u, err := url.Parse(origin)
		if err != nil || !validHTTPSOrigin(u) || u.Path != "" || u.RawPath != "" {
			return fmt.Errorf("MCP trusted origins must be HTTPS origins without paths")
		}
	}
	if c.MaxRequestBytes <= 0 || c.MaxRequestBytes > 16<<20 {
		return fmt.Errorf("MCP request limit must be between 1 byte and 16 MiB")
	}
	if c.RequestTimeout <= 0 || c.RequestTimeout > 5*time.Minute {
		return fmt.Errorf("MCP request timeout must be positive and at most 5 minutes")
	}
	return nil
}

func validHTTPSOrigin(u *url.URL) bool {
	return u != nil && u.Scheme == "https" && u.Hostname() != "" &&
		u.Host == strings.ToLower(u.Host) && !strings.HasSuffix(u.Hostname(), ".") &&
		u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.Opaque == ""
}
