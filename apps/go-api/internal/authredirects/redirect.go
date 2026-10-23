package authredirects

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

type QueryParam struct {
	Name  string
	Value string
}

// ValidateNextPath follows plane.utils.path_validator.validate_next_path.
func ValidateNextPath(nextPath string) string {
	if nextPath == "" || utf8.RuneCountInString(nextPath) > 500 {
		return ""
	}
	nextPath = strings.ReplaceAll(nextPath, "\\", "")
	parsed, err := url.Parse(nextPath)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		nextPath = parsed.Path
	}
	if nextPath == "" || !strings.HasPrefix(nextPath, "/") || strings.Contains(nextPath, "..") {
		return ""
	}
	lower := strings.ToLower(nextPath)
	for _, pattern := range []string{
		"javascript:", "data:", "vbscript:", "file:", "ftp:", "%2e%2e", "%2f%2f", "%5c%5c",
		"<script", "<iframe", "<object", "<embed", "<form", "onload=", "onerror=", "onclick=",
	} {
		if strings.Contains(lower, pattern) {
			return ""
		}
	}
	return nextPath
}

// SafeRedirectURL preserves the destination as a single query value alongside
// ordered error parameters.
func SafeRedirectURL(baseURL, nextPath string, params []QueryParam, allowedHosts []string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	validated := ValidateNextPath(nextPath)
	queryParts := make([]string, 0, len(params)+1)
	if validated != "" {
		queryParts = append(queryParts, "next_path="+url.QueryEscape(validated))
	}
	encodedParams := encodeQuery(params)
	if encodedParams != "" {
		queryParts = append(queryParts, encodedParams)
	}
	url := baseURL
	if len(queryParts) > 0 {
		url += "/?" + strings.Join(queryParts, "&")
	}
	if URLHasAllowedHostAndScheme(url, allowedHosts) {
		return url
	}
	if encodedParams != "" {
		return baseURL + "?" + encodedParams
	}
	return baseURL
}

// SpaceRedirectURL matches the app/space redirect path validation sequence.
func SpaceRedirectURL(baseURL, nextPath string, allowedHosts []string) string {
	validatedBase := strings.TrimRight(baseURL, "/")
	url := validatedBase + ValidateNextPath(nextPath)
	if URLHasAllowedHostAndScheme(url, allowedHosts) {
		return url
	}
	return baseURL
}

func URLHasAllowedHostAndScheme(value string, allowedHosts []string) bool {
	if strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n") || strings.Contains(value, "\\") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" {
		return false
	}
	if parsed.Scheme == "" {
		return parsed.Host == ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	for _, allowed := range allowedHosts {
		if strings.EqualFold(parsed.Host, allowed) {
			return true
		}
	}
	return false
}

func encodeQuery(params []QueryParam) string {
	parts := make([]string, 0, len(params))
	for _, param := range params {
		parts = append(parts, url.QueryEscape(param.Name)+"="+url.QueryEscape(param.Value))
	}
	return strings.Join(parts, "&")
}
