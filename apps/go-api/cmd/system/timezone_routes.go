package main

import (
	"net/http"
	"time"
)

const timezoneCacheDuration = 2 * time.Hour

func timezoneCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		setTimezoneProtocolHeaders(headers)
		headers.Set("Cache-Control", "max-age=7200")
		headers.Set("Expires", time.Now().UTC().Add(timezoneCacheDuration).Format(http.TimeFormat))
		next.ServeHTTP(response, request)
	})
}

func timezoneProtocolHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setTimezoneProtocolHeaders(response.Header())
		next.ServeHTTP(response, request)
	})
}

func setTimezoneProtocolHeaders(headers http.Header) {
	headers.Set("Allow", "GET, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	headers.Set("Vary", "Accept-Encoding, Cookie, origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}
