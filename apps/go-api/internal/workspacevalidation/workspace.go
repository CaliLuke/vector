// Package workspacevalidation holds validation rules shared by workspace
// creation and slug-availability endpoints.
package workspacevalidation

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// URLPattern is a Go regexp translation of plane.utils.url.URL_PATTERN. It
// intentionally detects URLs in workspace names; it is not a URL parser.
var URLPattern = regexp.MustCompile(`(?i)(?:https?://[^\s\p{Z}\x{85}\x{0b}\x{1c}-\x{1f}]+|www\.[a-zA-Z0-9İıſK](?:[a-zA-Z0-9İıſK-]{0,61}[a-zA-Z0-9İıſK])?(?:\.[a-zA-Z0-9İıſK](?:[a-zA-Z0-9İıſK-]{0,61}[a-zA-Z0-9İıſK])?)*|(?:[a-zA-Z0-9İıſK](?:[a-zA-Z0-9İıſK-]{0,61}[a-zA-Z0-9İıſK])?\.)+[a-zA-ZİıſK]{2,6}|(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))`)

// ContainsURL follows the source detector's length limits, line handling, and
// URL_PATTERN search behavior.
func ContainsURL(value string) bool {
	if utf8.RuneCountInString(value) > 1000 {
		return false
	}
	for _, line := range strings.Split(value, "\n") {
		runes := []rune(line)
		if len(runes) > 500 {
			runes = runes[:500]
		}
		if URLPattern.MatchString(string(runes)) {
			return true
		}
	}
	return false
}

var restrictedSlugs = []string{
	"404", "accounts", "api", "create-workspace", "god-mode", "installations", "invitations", "onboarding", "profile",
	"spaces", "workspace-invitations", "password", "flags", "monitor", "monitoring", "ingest", "plane-pro", "plane-ultimate",
	"enterprise", "plane-enterprise", "disco", "silo", "chat", "calendar", "drive", "channels", "upgrade", "billing",
	"sign-in", "sign-up", "signin", "signup", "config", "live", "admin", "m", "import", "importers", "integrations",
	"integration", "configuration", "initiatives", "initiative", "config", "workflow", "workflows", "epics", "epic", "story",
	"mobile", "dashboard", "desktop", "onload", "real-time", "one", "pages", "mobile", "business", "pro", "settings",
	"monitor", "license", "licenses", "instances", "instance",
}

var slugPattern = regexp.MustCompile(`\A[-a-zA-Z0-9_]+\z`)

// ValidSlug matches Django's default DRF SlugField character validator.
func ValidSlug(value string) bool { return slugPattern.MatchString(value) }

// RestrictedSlug matches the source's case-sensitive reserved-slug list.
func RestrictedSlug(value string) bool {
	for _, restricted := range restrictedSlugs {
		if value == restricted {
			return true
		}
	}
	return false
}

func RestrictedSlugs() []string { return append([]string(nil), restrictedSlugs...) }
