package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	loomhttp "github.com/CaliLuke/loom/http"
	"golang.org/x/text/unicode/norm"
)

type djangoCSRFCookieConfig struct {
	Domain string
	Secure bool
}

type djangoCORSOriginalOriginContextKey struct{}

type djangoCORSOriginResponseWriter struct {
	http.ResponseWriter
	originalOrigin string
}

func (writer *djangoCORSOriginResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func (writer *djangoCORSOriginResponseWriter) restoreOrigin() {
	if writer.originalOrigin != "" && writer.Header().Get("Access-Control-Allow-Origin") != "" {
		writer.Header().Set("Access-Control-Allow-Origin", writer.originalOrigin)
	}
}

func (writer *djangoCORSOriginResponseWriter) WriteHeader(status int) {
	writer.restoreOrigin()
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *djangoCORSOriginResponseWriter) Write(body []byte) (int, error) {
	writer.restoreOrigin()
	return writer.ResponseWriter.Write(body)
}

type csrfTokenCORSHeaderWriter struct {
	headers http.Header
}

func (writer *csrfTokenCORSHeaderWriter) Header() http.Header {
	return writer.headers
}

func (writer *csrfTokenCORSHeaderWriter) Write(body []byte) (int, error) {
	return len(body), nil
}

func (writer *csrfTokenCORSHeaderWriter) WriteHeader(int) {}

func djangoCORSRequestCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		normalized := normalizeDjangoCORSOrigin(request)
		if normalized == request {
			next.ServeHTTP(response, request)
			return
		}
		writer := &djangoCORSOriginResponseWriter{
			ResponseWriter: response,
			originalOrigin: djangoCORSOriginalOrigin(normalized),
		}
		next.ServeHTTP(writer, normalized)
	})
}

func normalizeDjangoCORSOrigin(request *http.Request) *http.Request {
	if _, normalized := request.Context().Value(djangoCORSOriginalOriginContextKey{}).(string); normalized {
		return request
	}
	if len(request.Header.Values("Origin")) == 0 {
		return request
	}
	original := djangoJoinedHeader(request, "Origin")
	clone := request.Clone(context.WithValue(request.Context(), djangoCORSOriginalOriginContextKey{}, original))
	clone.Header = request.Header.Clone()
	clone.Header.Del("Origin")
	if len(request.Header.Values("Origin")) > 1 {
		return clone
	}
	parsed, valid := djangoURLSplit(original)
	if original == "" || !valid {
		return clone
	}
	normalized := original
	if original != "null" && parsed.scheme != "" && parsed.netloc != "" {
		normalized = parsed.scheme + "://" + parsed.netloc
	}
	clone.Header.Set("Origin", normalized)
	return clone
}

func djangoCORSOriginalOrigin(request *http.Request) string {
	if original, ok := request.Context().Value(djangoCORSOriginalOriginContextKey{}).(string); ok {
		return original
	}
	return djangoJoinedHeader(request, "Origin")
}

func csrfTokenHeadersCompatibilityMiddleware(corsPolicy loomhttp.RuntimeCORSPolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/auth/get-csrf-token/" {
				next.ServeHTTP(response, request)
				return
			}
			setCSRFTokenCompatibilityHeaders(response.Header(), request, corsPolicy)
			next.ServeHTTP(response, request)
		})
	}
}

func csrfTokenCORSPreflightMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/auth/get-csrf-token/" && request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0 {
			writer := &csrfTokenResponseWriter{ResponseWriter: response}
			// Django accepts any requested method. Normalize it before forwarding to
			// Loom's generated OPTIONS dispatcher, then adapt Loom's preflight response.
			policyRequest := request.Clone(request.Context())
			policyRequest.Header = request.Header.Clone()
			policyRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
			next.ServeHTTP(writer, policyRequest)
			mergeVary(writer.Header(), "Origin")
			origin := djangoCORSOriginalOrigin(request)
			if origin == "" || !djangoCORSOriginValid(origin) {
				removeCORSAllowHeaders(writer.Header())
			} else if writer.Header().Get("Access-Control-Allow-Origin") != "" {
				writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
			writer.status = http.StatusOK
			writer.wroteHeader = true
			writer.body.Reset()
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			removeVary(writer.Header(), "Access-Control-Request-Method", "Access-Control-Request-Headers")
			writer.finish()
			return
		}
		next.ServeHTTP(response, request)
	})
}

func removeCORSAllowHeaders(headers http.Header) {
	for _, name := range []string{
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Origin",
		"Access-Control-Max-Age",
	} {
		headers.Del(name)
	}
}

func csrfTokenResponseCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/auth/get-csrf-token/" {
			next.ServeHTTP(response, request)
			return
		}
		if request.Method == http.MethodGet || request.Method == http.MethodHead {
			request = normalizeDjangoCSRFCookie(request)
		}
		writer := &csrfTokenResponseWriter{ResponseWriter: response}
		next.ServeHTTP(writer, request)
		if writer.status == http.StatusOK && (request.Method == http.MethodGet || request.Method == http.MethodHead) {
			mergeVary(writer.Header(), "Cookie")
		} else if request.Method == http.MethodGet || request.Method == http.MethodHead {
			removeResponseCookie(writer.Header(), defaultCSRFCookieName)
		}
		if writer.status == http.StatusRequestEntityTooLarge {
			writer.Header().Del("Allow")
			if len(writer.Header().Values("Set-Cookie")) == 0 {
				removeVary(writer.Header(), "Cookie")
			}
		}
		writer.finish()
	})
}

func djangoCSRFResponseCookiePolicy(config djangoCSRFCookieConfig) loomhttp.ResponseCookiePolicy {
	return func(_ context.Context, cookie *http.Cookie) error {
		if cookie.Name != defaultCSRFCookieName {
			return nil
		}
		if cookie.MaxAge <= 0 {
			return errors.New("generated CSRF response cookie has no positive Max-Age")
		}
		cookie.Domain = config.Domain
		cookie.Secure = config.Secure
		cookie.Expires = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
		return nil
	}
}

func djangoCSRFTokenRuntimeCORSPolicy(allowedOrigins []string) (loomhttp.RuntimeCORSPolicy, error) {
	return djangoRuntimeCORSPolicy(allowedOrigins)
}

func djangoRuntimeCORSPolicy(allowedOrigins []string) (loomhttp.RuntimeCORSPolicy, error) {
	originPolicy := loomhttp.CORSOrigin{
		Methods:     []string{"DELETE", "GET", "OPTIONS", "PATCH", "POST", "PUT"},
		Headers:     []string{"accept", "authorization", "content-type", "user-agent", "x-csrftoken", "x-requested-with", "X-API-Key"},
		MaxAge:      86_400,
		Credentials: true,
	}
	origins := make([]loomhttp.CORSOrigin, 0, max(1, len(allowedOrigins)))
	if len(allowedOrigins) == 0 {
		origin := originPolicy
		origin.Pattern = ".+"
		origin.Regex = true
		origins = append(origins, origin)
	} else {
		seen := make(map[string]struct{}, len(allowedOrigins))
		for _, configured := range allowedOrigins {
			pattern := configured
			if configured != "null" {
				if parsed, valid := djangoURLSplit(configured); valid && parsed.scheme != "" && parsed.netloc != "" {
					pattern = parsed.scheme + "://" + parsed.netloc
				}
			}
			if pattern == "" {
				continue
			}
			if _, exists := seen[pattern]; exists {
				continue
			}
			seen[pattern] = struct{}{}
			origin := originPolicy
			origin.Pattern = pattern
			origins = append(origins, origin)
		}
	}
	return loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: origins})
}

// Go's request-cookie helper returns the first duplicate, while Django's cookie
// parser keeps the last. Prepending Django's effective value gives Loom's
// generated decoder the same input without replacing the generated decoder.
func normalizeDjangoCSRFCookie(request *http.Request) *http.Request {
	return normalizeDjangoCookie(request, defaultCSRFCookieName)
}

type csrfTokenResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func (writer *csrfTokenResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
}

func (writer *csrfTokenResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *csrfTokenResponseWriter) finish() {
	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	body := writer.body.Bytes()
	if status == http.StatusOK {
		body = bytes.TrimSuffix(body, []byte{'\n'})
		writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	writer.Header().Del("loom-error")
	writer.ResponseWriter.WriteHeader(status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

func setCSRFTokenCompatibilityHeaders(headers http.Header, request *http.Request, corsPolicy loomhttp.RuntimeCORSPolicy) {
	headers.Set("Allow", "GET, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	removeVary(headers, "Origin")
	mergeVary(headers, "Cookie", "origin")
	origin := djangoCORSOriginalOrigin(request)
	if headers.Get("Access-Control-Allow-Origin") != "" {
		headers.Set("Access-Control-Allow-Origin", origin)
		if request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) == 0 {
			policyRequest := request.Clone(request.Context())
			policyRequest.Header = request.Header.Clone()
			policyRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
			policyResponse := &csrfTokenCORSHeaderWriter{headers: make(http.Header)}
			corsPolicy.HandlePreflight(policyResponse, policyRequest, []string{http.MethodGet, http.MethodOptions})
			for _, name := range []string{
				"Access-Control-Allow-Headers",
				"Access-Control-Allow-Methods",
				"Access-Control-Max-Age",
			} {
				if value := policyResponse.Header().Get(name); value != "" {
					headers.Set(name, value)
				}
			}
		}
	}
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func publicProjectMetadataReadsCORSPreflight(headers http.Header, request *http.Request, allowedOrigins []string) bool {
	if request.Method != http.MethodOptions || len(request.Header.Values("Access-Control-Request-Method")) == 0 {
		return false
	}
	headers.Set("Content-Length", "0")
	headers.Set("Content-Type", "text/html; charset=utf-8")
	mergeVary(headers, "origin")
	if publicProjectMetadataReadsCORSHeaders(headers, request, allowedOrigins) {
		headers.Set("Access-Control-Allow-Headers", "accept, authorization, content-type, user-agent, x-csrftoken, x-requested-with, X-API-Key")
		headers.Set("Access-Control-Allow-Methods", "DELETE, GET, OPTIONS, PATCH, POST, PUT")
		headers.Set("Access-Control-Max-Age", "86400")
	}
	return true
}

func publicProjectMetadataReadsCORSHeaders(headers http.Header, request *http.Request, allowedOrigins []string) bool {
	origin := djangoJoinedHeader(request, "Origin")
	originURL, valid := djangoURLSplit(origin)
	if origin == "" || !valid {
		return false
	}
	allowed := len(allowedOrigins) == 0
	for _, candidate := range allowedOrigins {
		candidateURL, candidateValid := djangoURLSplit(candidate)
		if (origin == "null" && candidate == "null") ||
			(candidateValid && originURL.scheme == candidateURL.scheme && originURL.netloc == candidateURL.netloc) {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Credentials", "true")
	return true
}

func djangoCORSOriginValid(origin string) bool {
	_, valid := djangoURLSplit(origin)
	return valid
}

type djangoURLSplitResult struct {
	scheme string
	netloc string
}

func djangoURLSplit(origin string) (djangoURLSplitResult, bool) {
	// Python 3.12's urlsplit() strips these characters before examining the
	// authority and rejects malformed bracketed hosts and NFKC delimiters.
	origin = strings.TrimLeftFunc(strings.NewReplacer("\r", "", "\n", "", "\t", "").Replace(origin), func(value rune) bool {
		return value <= ' '
	})
	result := djangoURLSplitResult{}
	rest := origin
	if separator := strings.IndexByte(rest, ':'); separator > 0 {
		firstDelimiter := strings.IndexAny(rest, "/?#")
		if (firstDelimiter < 0 || separator < firstDelimiter) && djangoURLSchemeValid(rest[:separator]) {
			result.scheme = strings.ToLower(rest[:separator])
			rest = rest[separator+1:]
		}
	}
	if !strings.HasPrefix(rest, "//") {
		return result, true
	}
	authority := rest[2:]
	if end := strings.IndexAny(authority, "/?#"); end >= 0 {
		authority = authority[:end]
	}
	result.netloc = authority
	if strings.Contains(authority, "[") != strings.Contains(authority, "]") {
		return djangoURLSplitResult{}, false
	}
	if strings.Contains(authority, "[") {
		// Python 3.12 validates only the contents between the first '[' and
		// the first following ']'; it intentionally does not validate where
		// that pair appears in the authority or what follows it.
		host := strings.SplitN(strings.SplitN(authority, "[", 2)[1], "]", 2)[0]
		if strings.HasPrefix(host, "v") {
			version, suffix, found := strings.Cut(host[1:], ".")
			if !found || version == "" || suffix == "" || strings.IndexFunc(version, func(value rune) bool {
				return (value < '0' || value > '9') && (value < 'a' || value > 'f') && (value < 'A' || value > 'F')
			}) >= 0 {
				return djangoURLSplitResult{}, false
			}
		} else {
			address := host
			if base, zone, found := strings.Cut(host, "%"); found {
				if zone == "" || strings.Contains(zone, "%") {
					return djangoURLSplitResult{}, false
				}
				address = base
			}
			if net.ParseIP(address) == nil || !strings.Contains(address, ":") {
				return djangoURLSplitResult{}, false
			}
		}
	}
	check := strings.NewReplacer("@", "", ":", "", "#", "", "?", "").Replace(authority)
	normalized := norm.NFKC.String(check)
	if check != normalized && strings.ContainsAny(normalized, "/?#@:") {
		return djangoURLSplitResult{}, false
	}
	return result, true
}

func djangoURLSchemeValid(value string) bool {
	for index, character := range value {
		if index == 0 && ((character < 'a' || character > 'z') && (character < 'A' || character > 'Z')) {
			return false
		}
		if index > 0 && ((character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') && character != '+' && character != '-' && character != '.') {
			return false
		}
	}
	return value != ""
}
