package browsersession

import "net/http"

// CookieFromHeaderValues reads a named cookie using the HTTP request parser.
func CookieFromHeaderValues(headerValues []string, name string) *http.Cookie {
	request := &http.Request{Header: http.Header{"Cookie": headerValues}}
	cookie, err := request.Cookie(name)
	if err != nil {
		return nil
	}
	return cookie
}
